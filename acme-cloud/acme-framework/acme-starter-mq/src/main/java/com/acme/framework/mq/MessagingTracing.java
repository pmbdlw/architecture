package com.acme.framework.mq;

import io.micrometer.tracing.Span;
import io.micrometer.tracing.TraceContext;
import io.micrometer.tracing.Tracer;
import io.micrometer.tracing.propagation.Propagator;
import java.util.HashMap;
import java.util.Map;

/**
 * 把链路上下文带过“事务发件箱 + 消息队列”这段异步链路。
 * <p>
 * 下单请求的 traceparent 随事件写入 outbox_event.trace_parent；Relay 投递时以它为父节点创建 PRODUCER Span，
 * 并把新的 traceparent 写进消息头；消费端再从消息头恢复上下文创建 CONSUMER Span。
 * 这样“HTTP 下单 → 发件箱 → MQ → 消费者”在链路系统里是同一条 Trace。
 */
public class MessagingTracing {

    /** W3C Trace Context 头。 */
    public static final String TRACEPARENT = "traceparent";

    private final Tracer tracer;
    private final Propagator propagator;

    public MessagingTracing(Tracer tracer, Propagator propagator) {
        this.tracer = tracer;
        this.propagator = propagator;
    }

    /** 未引入链路追踪时使用的空实现。 */
    public static MessagingTracing noop() {
        return new MessagingTracing(Tracer.NOOP, Propagator.NOOP);
    }

    public Tracer tracer() {
        return tracer;
    }

    /** 当前线程的 traceparent，没有活动链路时返回 null。 */
    public String currentTraceParent() {
        TraceContext context = tracer.currentTraceContext().context();
        return context == null ? null : inject(context).get(TRACEPARENT);
    }

    /** 以 traceparent 为父节点（为空则新起一条链路）创建 Span，调用方负责 start / end。 */
    public Span.Builder spanBuilder(String traceParent) {
        if (traceParent == null || traceParent.isBlank()) {
            return tracer.spanBuilder().setNoParent();
        }
        return propagator.extract(Map.of(TRACEPARENT, traceParent), Map::get);
    }

    /** 把 Span 的上下文序列化为消息头（traceparent 等）。 */
    public Map<String, String> inject(TraceContext context) {
        Map<String, String> carrier = new HashMap<>();
        propagator.inject(context, carrier, Map::put);
        return carrier;
    }
}
