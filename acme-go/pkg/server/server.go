package server

import (
	stdhttp "net/http"
	"os"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	"github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/result"
)

// NewHTTPServer 创建对外 HTTP 服务器：统一响应体、统一错误码。
func NewHTTPServer(c conf.Listen, logger log.Logger, mws []middleware.Middleware, extra ...http.ServerOption) *http.Server {
	opts := []http.ServerOption{
		http.Middleware(mws...),
		http.ResponseEncoder(result.ResponseEncoder),
		http.ErrorEncoder(result.ErrorEncoder(logger)),
	}
	if c.Addr != "" {
		opts = append(opts, http.Address(c.Addr))
	}
	if c.Timeout.Duration > 0 {
		opts = append(opts, http.Timeout(c.Timeout.Duration))
	}
	return http.NewServer(append(opts, extra...)...)
}

// HeaderServedBy 处理请求的服务实例，如 user-service@user-service-gray-7d9f-x2v7k gray=v2。
const HeaderServedBy = "X-Served-By"

// ServedBy 在 app.servedByHeader=true 时为每个响应加上 X-Served-By 头，否则不做任何事。
func ServedBy(app conf.App) http.ServerOption {
	if !app.ServedByHeader {
		return func(*http.Server) {}
	}
	host, _ := os.Hostname()
	value := app.Name + "@" + host
	if app.GrayTag != "" {
		value += " gray=" + app.GrayTag
	}
	return http.Filter(func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			w.Header().Set(HeaderServedBy, value)
			next.ServeHTTP(w, r)
		})
	})
}

// NewGRPCServer 创建服务间调用的 gRPC 服务器（只注册 Internal 服务，网关不会转发到这里）。
func NewGRPCServer(c conf.Listen, mws []middleware.Middleware) *grpc.Server {
	opts := []grpc.ServerOption{grpc.Middleware(mws...)}
	if c.Addr != "" {
		opts = append(opts, grpc.Address(c.Addr))
	}
	if c.Timeout.Duration > 0 {
		opts = append(opts, grpc.Timeout(c.Timeout.Duration))
	}
	return grpc.NewServer(opts...)
}
