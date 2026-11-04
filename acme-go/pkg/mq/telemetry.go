package mq

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HeaderTraceParent W3C Trace Context 头。
//
// 下单请求的 traceparent 随事件写入 outbox_event.trace_parent；Relay 投递时以它为父节点创建 PRODUCER Span，
// 并把新的 traceparent 写进消息头；消费端再从消息头恢复上下文创建 CONSUMER Span。
// 这样“HTTP 下单 → 发件箱 → MQ → 消费者”在链路系统里是同一条 Trace。
const HeaderTraceParent = "traceparent"

const instrumentationName = "github.com/acme/acme-go/pkg/mq"

func tracer() trace.Tracer { return otel.Tracer(instrumentationName) }

// currentTraceParent 返回 ctx 中的 traceparent，没有活动链路时返回空串。
func currentTraceParent(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get(HeaderTraceParent)
}

// contextFromTraceParent 以 traceparent 为父节点构造上下文；为空时返回原 ctx（新起一条链路）。
func contextFromTraceParent(ctx context.Context, traceParent string) context.Context {
	if traceParent == "" {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier{HeaderTraceParent: traceParent})
}

// 指标名与 Prometheus 中的名称：
//
//	acme.outbox.published → acme_outbox_published_total{topic,type,result}
//	acme.mq.consumed      → acme_mq_consumed_total{consumer,type,result}
//	acme.outbox.pending   → acme_outbox_pending（见 Relay.RegisterMetrics）
type instruments struct {
	published metric.Int64Counter
	consumed  metric.Int64Counter
}

var (
	inst     instruments
	instOnce sync.Once
)

// metrics 延迟创建指标（在 observability.Setup 设置全局 MeterProvider 之后）。
func metrics() instruments {
	instOnce.Do(func() {
		m := otel.Meter(instrumentationName)
		inst.published, _ = m.Int64Counter("acme.outbox.published", metric.WithDescription("发件箱投递次数"))
		inst.consumed, _ = m.Int64Counter("acme.mq.consumed", metric.WithDescription("消费事件次数"))
	})
	return inst
}

func attrs(kv ...string) metric.MeasurementOption {
	a := make([]attribute.KeyValue, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		a = append(a, attribute.String(kv[i], kv[i+1]))
	}
	return metric.WithAttributes(a...)
}
