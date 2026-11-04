// Package client 创建服务间 gRPC 连接：
// 服务发现 + 灰度实例筛选 + 超时 + 熔断 + 元数据透传 + 错误还原。
package client

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/circuitbreaker"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/registry"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"google.golang.org/grpc"

	"github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/errorx"
	"github.com/acme/acme-go/pkg/selector/gray"
	"github.com/acme/acme-go/pkg/server"
)

// Dial 连接下游服务：
//   - 有注册中心时按 Nacos 中的 <服务名>.grpc 发现实例，并按灰度标签筛选；
//   - 本地开发未启用 Nacos 时，按 client.endpoints 中配置的地址直连。
func Dial(ctx context.Context, discovery registry.Discovery, c conf.Client, service string) (*grpc.ClientConn, error) {
	opts := []kgrpc.ClientOption{
		kgrpc.WithTimeout(c.Timeout.Duration),
		kgrpc.WithMiddleware(Middlewares()...),
	}
	if discovery != nil {
		opts = append(opts,
			kgrpc.WithEndpoint("discovery:///"+service+".grpc"),
			kgrpc.WithDiscovery(discovery),
			kgrpc.WithNodeFilter(gray.Filter()),
		)
	} else {
		endpoint, ok := c.Endpoints[service]
		if !ok {
			return nil, fmt.Errorf("未启用 Nacos 时需要配置 client.endpoints.%s", service)
		}
		opts = append(opts, kgrpc.WithEndpoint(endpoint))
	}
	return kgrpc.DialInsecure(ctx, opts...)
}

// Middlewares 客户端中间件链。顺序很重要：remoteErrors 在熔断器外层，熔断器看到原始错误，
// 只有 5xx / 超时 / 不可用才计入失败，下游返回的 4xx 业务错误不会打开熔断器。
func Middlewares() []middleware.Middleware {
	return []middleware.Middleware{
		recovery.Recovery(),
		tracing.Client(),
		metadata.Client(metadata.WithPropagatedPrefix(server.PropagatedPrefixes...)),
		remoteErrors(),
		circuitbreaker.Client(),
	}
}

// remoteErrors 把下游错误还原为平台语义：
//   - 4xx 且是平台错误码：原样返回（调用方可按错误码翻译为本服务语义）；
//   - 熔断打开 / 不可用 / 超时：C0503；
//   - 其它：C0001。
func remoteErrors() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			reply, err := handler(ctx, req)
			if err == nil {
				return reply, nil
			}
			se := errors.FromError(err)
			switch {
			case se.Code < 500 && errorx.IsBizReason(se.Reason):
				return nil, se
			case stderrors.Is(err, circuitbreaker.ErrNotAllowed), stderrors.Is(err, context.DeadlineExceeded),
				se.Code == 503, se.Code == 504:
				return nil, errorx.RemoteUnavailable.New()
			default:
				return nil, errorx.RemoteError.New()
			}
		}
	}
}
