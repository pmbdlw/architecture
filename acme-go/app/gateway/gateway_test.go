package gateway_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/redis/go-redis/v9"

	"github.com/acme/acme-go/app/gateway/internal/conf"
	"github.com/acme/acme-go/app/gateway/internal/gateway"
	pconf "github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/testkit"
)

// backend 模拟下游服务：返回统一响应，并回显网关转发过来的关键请求头。
func backend(t *testing.T, name string, status int) *httptest.Server {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": "00000", "message": "成功", "data": map[string]string{
			"instance": name, "grayTag": r.Header.Get("X-Gray-Tag"), "path": r.URL.Path,
			"auth": r.Header.Get("Authorization")}})
	}))
	t.Cleanup(s.Close)
	return s
}

type fixture struct {
	gw     *gateway.Gateway
	signer *testkit.Signer
	reg    *testkit.Registry
	redis  *miniredis.Miniredis
	cfg    conf.Gateway
}

func setup(t *testing.T) *fixture {
	f := &fixture{signer: testkit.NewSigner(t), reg: testkit.NewRegistry(), redis: miniredis.RunT(t)}
	register := func(service, name, grayTag string, status int) {
		md := map[string]string{"kind": "http"}
		if grayTag != "" {
			md["gray-tag"] = grayTag
		}
		_ = f.reg.Register(context.Background(), &registry.ServiceInstance{ID: name, Name: service + ".http",
			Metadata: md, Endpoints: []string{backend(t, name, status).URL}})
	}
	register("user-service", "user-stable", "", 200)
	register("user-service", "user-v2", "v2", 200)
	register("order-service", "order-broken", "", 500)

	f.cfg = conf.Gateway{
		Routes: []conf.Route{
			{ID: "user-service", Prefix: "/api/v1/users", Service: "user-service"},
			{ID: "order-service", Prefix: "/api/v1/orders", Service: "order-service"},
			{ID: "notification-service", Prefix: "/api/v1/notifications", Service: "notification-service"},
		},
		RateLimit: conf.RateLimit{Enabled: true, PerSecond: 100, Burst: 100},
		Breaker:   conf.Breaker{MinRequests: 3, FailureRatio: 0.5, OpenTimeout: pconf.Duration{Duration: time.Minute}},
		Gray:      conf.Gray{Enabled: true, Tag: "v2", UIDs: []string{"1001"}},
	}
	rdb := redis.NewClient(&redis.Options{Addr: f.redis.Addr()})
	f.gw = gateway.New(t.Context(), f.cfg, testkit.Issuer, f.signer.KeyFunc, f.reg, rdb, log.DefaultLogger)
	time.Sleep(50 * time.Millisecond) // 等待实例列表同步
	return f
}

func (f *fixture) do(t *testing.T, path, token string, header map[string]string) testkit.Response {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	f.gw.ServeHTTP(rec, req)
	resp := testkit.Response{Status: rec.Code, Header: rec.Header()}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("%s: %q", path, rec.Body.String())
	}
	return resp
}

func TestRoutingSecurityAndGray(t *testing.T) {
	f := setup(t)
	alice := f.signer.Token(t, "1001", "USER")
	bob := f.signer.Token(t, "2002", "USER")

	cases := []struct {
		name, path, token string
		header            map[string]string
		status            int
		code, instance    string
	}{
		{"内部接口不对外（未登录也返回 404）", "/api/v1/users/internal/1", "", nil, 404, "A0404", ""},
		{"未登录在边缘拒绝", "/api/v1/users/1", "", nil, 401, "A0401", ""},
		{"伪造令牌", "/api/v1/users/1", "not-a-jwt", nil, 401, "A0401", ""},
		{"未知路由", "/api/v1/unknown", bob, nil, 404, "A0404", ""},
		{"普通用户只到稳定实例", "/api/v1/users/1", bob, nil, 200, "00000", "user-stable"},
		{"白名单用户进入灰度实例", "/api/v1/users/1", alice, nil, 200, "00000", "user-v2"},
		{"客户端伪造灰度头无效", "/api/v1/users/1", bob, map[string]string{"X-Gray-Tag": "v2"}, 200, "00000", "user-stable"},
		{"服务无实例时降级", "/api/v1/notifications", bob, nil, 503, "C0503", ""},
	}
	for _, c := range cases {
		resp := f.do(t, c.path, c.token, c.header)
		if resp.Status != c.status || resp.Code != c.code {
			t.Errorf("%s: got %d %s (%s)", c.name, resp.Status, resp.Code, resp.Message)
			continue
		}
		if c.instance != "" && resp.DataField(t, "instance") != c.instance {
			t.Errorf("%s: routed to %v", c.name, resp.DataField(t, "instance"))
		}
	}
	// 令牌原样透传给下游，下游服务再次校验（纵深防御）
	if got := f.do(t, "/api/v1/users/1", bob, nil).DataField(t, "auth"); got != "Bearer "+bob {
		t.Errorf("authorization not relayed: %v", got)
	}
	if got := f.do(t, "/api/v1/users/1", alice, nil).DataField(t, "grayTag"); got != "v2" {
		t.Errorf("gray tag not propagated downstream: %v", got)
	}
}

func TestCircuitBreakerOpensOnUpstreamErrors(t *testing.T) {
	f := setup(t)
	token := f.signer.Token(t, "2002")
	for range 3 {
		if r := f.do(t, "/api/v1/orders/1", token, nil); r.Status != 500 {
			t.Fatalf("upstream error should pass through before breaker opens: %d", r.Status)
		}
	}
	if r := f.do(t, "/api/v1/orders/1", token, nil); r.Status != 503 || r.Code != "C0503" {
		t.Fatalf("breaker should be open: %d %s", r.Status, r.Code)
	}
}

func TestRateLimit(t *testing.T) {
	f := setup(t)
	f.cfg.RateLimit = conf.RateLimit{Enabled: true, PerSecond: 1, Burst: 2}
	f.gw.Update(f.cfg)
	token := f.signer.Token(t, "3003")
	codes := map[int]int{}
	for range 5 {
		codes[f.do(t, "/api/v1/users/1", token, nil).Status]++
	}
	if codes[429] == 0 || codes[200] == 0 {
		t.Fatalf("expected some requests limited: %v", codes)
	}
	// Redis 故障时放行（可用性优先）
	f.redis.Close()
	if r := f.do(t, "/api/v1/users/1", token, nil); r.Status != 200 {
		t.Fatalf("limiter must fail open: %d", r.Status)
	}
}

func TestResolveGrayTag(t *testing.T) {
	rules := conf.Gray{Enabled: true, Tag: "v2", UIDs: []string{"1001"}}
	if gateway.ResolveGrayTag(rules, "", "1001") != "v2" || gateway.ResolveGrayTag(rules, "v9", "2002") != "" {
		t.Fatal("whitelist rules wrong")
	}
	if gateway.ResolveGrayTag(conf.Gray{AllowClientHeader: true}, "v9", "") != "v9" {
		t.Fatal("test env should trust client header")
	}
	all := conf.Gray{Enabled: true, Tag: "v2", Percentage: 100}
	if gateway.ResolveGrayTag(all, "", "42") != "v2" {
		t.Fatal("100% rollout should tag every user")
	}
}
