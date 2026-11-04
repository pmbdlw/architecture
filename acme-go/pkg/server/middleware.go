// Package server 装配所有服务统一的 HTTP / gRPC / 管理端口服务器、中间件链与应用生命周期。
package server

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel/trace"

	"github.com/acme/acme-go/pkg/errorx"
	"github.com/acme/acme-go/pkg/observability"
	"github.com/acme/acme-go/pkg/result"
	"github.com/acme/acme-go/pkg/selector/gray"
)

// PropagatedPrefixes 需要在服务间透传的请求头：Kratos 默认的 x-md-*、灰度标签、身份令牌。
// 链路追踪头（traceparent）由 tracing 中间件处理。
var PropagatedPrefixes = []string{"x-md-", gray.Header, "authorization"}

// Middlewares 返回服务端标准中间件链。auth 为 nil 表示不校验 JWT（如认证中心自身）。
// 顺序：恢复 → 链路 → 回写 TraceId → 指标 → 访问日志 → 透传元数据 → 鉴权 → 错误规整 → 业务。
func Middlewares(t *observability.Telemetry, auth middleware.Middleware) []middleware.Middleware {
	mws := []middleware.Middleware{
		recovery.Recovery(),
		tracing.Server(),
		traceIDHeader(),
	}
	requests, errReq := metrics.DefaultRequestsCounter(t.Meter, metrics.DefaultServerRequestsCounterName)
	seconds, errSec := metrics.DefaultSecondsHistogram(t.Meter, metrics.DefaultServerSecondsHistogramName)
	if errReq == nil && errSec == nil {
		mws = append(mws, metrics.Server(metrics.WithRequests(requests), metrics.WithSeconds(seconds)))
	}
	mws = append(mws,
		accessLog(t.Logger),
		metadata.Server(metadata.WithPropagatedPrefix(PropagatedPrefixes...)),
	)
	if auth != nil {
		mws = append(mws, auth)
	}
	return append(mws, normalizeErrors(t.Logger))
}

// accessLog 访问日志：成功与 4xx 业务失败记 INFO，5xx 记 ERROR（不打印堆栈，原始错误由 normalizeErrors 记录）。
func accessLog(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			start := time.Now()
			reply, err := handler(ctx, req)
			kind, operation := "", ""
			if tr, ok := transport.FromServerContext(ctx); ok {
				kind, operation = tr.Kind().String(), tr.Operation()
			}
			code, reason := int32(200), ""
			if err != nil {
				se := errors.FromError(err)
				code, reason = se.Code, se.Reason
			}
			level := log.LevelInfo
			if code >= 500 {
				level = log.LevelError
			}
			_ = log.WithContext(ctx, logger).Log(level, "msg", "access", "kind", kind, "operation", operation,
				"code", code, "reason", reason, "latency_ms", time.Since(start).Milliseconds())
			return reply, err
		}
	}
}

// traceIDHeader 把当前链路 ID 写入响应头 X-Trace-Id。
func traceIDHeader() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
					tr.ReplyHeader().Set(result.TraceHeader, sc.TraceID().String())
				}
			}
			return handler(ctx, req)
		}
	}
}

// normalizeErrors 把业务代码返回的错误规整为平台错误码（gRPC 调用方据此还原业务错误）；
// 未知错误在这里以 ERROR 记录一次原始信息，对外只返回 B0001。
func normalizeErrors(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			reply, err := handler(ctx, req)
			if err == nil {
				return reply, nil
			}
			se := errorx.Normalize(err)
			if se.Reason == errorx.SystemError.Reason && !errorx.IsBizReason(errors.Reason(err)) {
				_ = log.WithContext(ctx, logger).Log(log.LevelError, "msg", "未处理异常", "error", err.Error())
			}
			return nil, se
		}
	}
}
