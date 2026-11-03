package server

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	"github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/result"
)

// NewHTTPServer 创建对外 HTTP 服务器：统一响应体、统一错误码。
func NewHTTPServer(c conf.Listen, logger log.Logger, mws []middleware.Middleware) *http.Server {
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
	return http.NewServer(opts...)
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
