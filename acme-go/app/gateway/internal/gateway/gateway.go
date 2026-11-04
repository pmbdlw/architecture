// Package gateway API 网关：所有外部请求的统一入口。
//
// 过滤链（外 → 内）：链路追踪 → 屏蔽 /internal → CORS → 路由匹配 → JWT 校验 → 限流 → 灰度打标 → 熔断 → 反向代理。
package gateway

import (
	"context"
	"errors"
	"hash/fnv"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-redis/redis_rate/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/acme/acme-go/app/gateway/internal/conf"
	"github.com/acme/acme-go/pkg/auth"
	"github.com/acme/acme-go/pkg/errorx"
	"github.com/acme/acme-go/pkg/result"
	"github.com/acme/acme-go/pkg/selector/gray"
)

// Gateway 是网关的 http.Handler。配置通过 Update 热更新（Nacos 推送）。
type Gateway struct {
	cfg      atomic.Pointer[conf.Gateway]
	issuer   string
	keyFunc  jwt.Keyfunc
	limiter  *redis_rate.Limiter
	balancer *balancer
	log      *log.Helper
	ctx      context.Context

	mu       sync.Mutex
	breakers map[string]*gobreaker.CircuitBreaker[int]
	proxy    *httputil.ReverseProxy
	handler  http.Handler
}

type ctxKey struct{}

type target struct {
	route conf.Route
	addr  string
}

// New 创建网关。rdb 为 nil 时不限流。
func New(ctx context.Context, c conf.Gateway, issuer string, keyFunc jwt.Keyfunc, discovery registry.Discovery,
	rdb *redis.Client, logger log.Logger) *Gateway {
	g := &Gateway{issuer: issuer, keyFunc: keyFunc, balancer: newBalancer(discovery, logger),
		log: log.NewHelper(logger), ctx: ctx, breakers: map[string]*gobreaker.CircuitBreaker[int]{}}
	if rdb != nil {
		g.limiter = redis_rate.NewLimiter(rdb)
	}
	g.proxy = &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			t := pr.In.Context().Value(ctxKey{}).(*target)
			u, _ := url.Parse(t.addr)
			pr.SetURL(u)
			pr.SetXForwarded()
			pr.Out.Host = pr.In.Host
		},
		Transport: otelhttp.NewTransport(&http.Transport{
			Proxy:                 nil,
			DialContext:           (&net.Dialer{Timeout: time.Second, KeepAlive: 30 * time.Second}).DialContext,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: timeoutOr(c.Timeout.Duration, 10*time.Second),
		}),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			g.log.WithContext(r.Context()).Warnw("msg", "转发失败", "path", r.URL.Path, "error", err)
			result.Write(w, errorx.RemoteUnavailable, "")
		},
	}
	g.Update(c)
	g.handler = otelhttp.NewHandler(http.HandlerFunc(g.serve), "gateway",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string { return r.Method + " " + r.URL.Path }))
	return g
}

func timeoutOr(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}

// Update 热更新网关配置（路由、灰度、限流、熔断）。
func (g *Gateway) Update(c conf.Gateway) {
	for _, r := range c.Routes {
		g.balancer.watch(g.ctx, r.Service)
	}
	g.mu.Lock()
	g.breakers = map[string]*gobreaker.CircuitBreaker[int]{}
	g.mu.Unlock()
	g.cfg.Store(&c)
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) { g.handler.ServeHTTP(w, r) }

func (g *Gateway) serve(w http.ResponseWriter, r *http.Request) {
	c := g.cfg.Load()
	if sc := trace.SpanContextFromContext(r.Context()); sc.HasTraceID() {
		w.Header().Set(result.TraceHeader, sc.TraceID().String())
	}
	// /internal/** 只允许集群内调用：一律 404，且在鉴权之前判断，不暴露其存在
	if strings.Contains(r.URL.Path, "/internal/") {
		result.Write(w, errorx.NotFound, "")
		return
	}
	if g.cors(c, w, r) {
		return
	}
	route, ok := matchRoute(c.Routes, r.URL.Path)
	if !ok {
		result.Write(w, errorx.NotFound, "")
		return
	}

	var claims *auth.Claims
	if !isPublic(c.PublicPrefixes, r.URL.Path) {
		var err error
		if claims, err = g.authenticate(r); err != nil {
			result.Write(w, errorx.Unauthorized, "")
			return
		}
	}
	if !g.allow(r.Context(), c, r, claims) {
		result.Write(w, errorx.TooManyRequests, "")
		return
	}

	// 灰度打标：先移除客户端自带的值，防止外部伪造
	clientTag := r.Header.Get(gray.Header)
	r.Header.Del(gray.Header)
	uid := ""
	if claims != nil {
		uid = claims.UID
	}
	tag := ResolveGrayTag(c.Gray, clientTag, uid)
	if tag != "" {
		r.Header.Set(gray.Header, tag)
	}

	addr := g.balancer.pick(route.Service, tag)
	if addr == "" {
		result.Write(w, errorx.RemoteUnavailable, route.Service+" 暂无可用实例")
		return
	}
	g.forward(c, w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, &target{route: route, addr: addr})), route)
}

// forward 经熔断器转发：下游 5xx 或连接失败计为失败；熔断打开时直接返回 C0503 降级响应。
func (g *Gateway) forward(c *conf.Gateway, w http.ResponseWriter, r *http.Request, route conf.Route) {
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	_, err := g.breaker(c, route.ID).Execute(func() (int, error) {
		g.proxy.ServeHTTP(rec, r)
		if rec.status >= 500 {
			return rec.status, errors.New(http.StatusText(rec.status))
		}
		return rec.status, nil
	})
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		result.Write(w, errorx.RemoteUnavailable, route.Service+" 暂不可用，请稍后重试")
	}
}

func (g *Gateway) breaker(c *conf.Gateway, id string) *gobreaker.CircuitBreaker[int] {
	g.mu.Lock()
	defer g.mu.Unlock()
	if b, ok := g.breakers[id]; ok {
		return b
	}
	bc := c.Breaker
	minRequests := bc.MinRequests
	if minRequests == 0 {
		minRequests = 20
	}
	ratio := bc.FailureRatio
	if ratio == 0 {
		ratio = 0.5
	}
	b := gobreaker.NewCircuitBreaker[int](gobreaker.Settings{
		Name:        id,
		MaxRequests: 5,
		Interval:    timeoutOr(bc.Interval.Duration, 60*time.Second),
		Timeout:     timeoutOr(bc.OpenTimeout.Duration, 10*time.Second),
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.Requests >= minRequests && float64(counts.TotalFailures)/float64(counts.Requests) >= ratio
		},
	})
	g.breakers[id] = b
	return b
}

func (g *Gateway) authenticate(r *http.Request) (*auth.Claims, error) {
	raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || raw == "" {
		return nil, errors.New("missing token")
	}
	claims := &auth.Claims{}
	_, err := jwt.ParseWithClaims(raw, claims, g.keyFunc, jwt.WithIssuer(g.issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	return claims, err
}

// allow 限流：已登录按用户（sub），未登录按客户端 IP。Redis 故障时放行。
func (g *Gateway) allow(ctx context.Context, c *conf.Gateway, r *http.Request, claims *auth.Claims) bool {
	if g.limiter == nil || !c.RateLimit.Enabled {
		return true
	}
	key := "ip:" + clientIP(r)
	if claims != nil && claims.Subject != "" {
		key = "user:" + claims.Subject
	}
	res, err := g.limiter.Allow(ctx, "acme:gw:"+key, redis_rate.Limit{
		Rate: c.RateLimit.PerSecond, Burst: c.RateLimit.Burst, Period: time.Second})
	if err != nil {
		g.log.WithContext(ctx).Warnw("msg", "限流检查失败，放行", "error", err)
		return true
	}
	return res.Allowed > 0
}

func (g *Gateway) cors(c *conf.Gateway, w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || !slices.Contains(c.CORS.AllowedOrigins, origin) {
		return false
	}
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Access-Control-Allow-Credentials", "true")
	h.Add("Vary", "Origin")
	if r.Method == http.MethodOptions {
		h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		h.Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

// ResolveGrayTag 按规则计算请求的灰度标签。
func ResolveGrayTag(g conf.Gray, clientTag, uid string) string {
	if g.AllowClientHeader && clientTag != "" {
		return clientTag
	}
	if !g.Enabled || g.Tag == "" || uid == "" {
		return ""
	}
	if slices.Contains(g.UIDs, uid) {
		return g.Tag
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(uid))
	if int(h.Sum32()%100) < g.Percentage {
		return g.Tag
	}
	return ""
}

func matchRoute(routes []conf.Route, path string) (conf.Route, bool) {
	for _, r := range routes {
		if path == r.Prefix || strings.HasPrefix(path, strings.TrimSuffix(r.Prefix, "/")+"/") {
			return r, true
		}
	}
	return conf.Route{}, false
}

func isPublic(prefixes []string, path string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(ip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
