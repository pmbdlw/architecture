package com.acme.framework.mq;

import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.tracing.Span;
import io.micrometer.tracing.Tracer;
import java.nio.charset.StandardCharsets;
import java.sql.Timestamp;
import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import java.util.List;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.cloud.stream.function.StreamBridge;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.messaging.support.MessageBuilder;
import org.springframework.transaction.TransactionDefinition;
import org.springframework.transaction.support.TransactionTemplate;
import org.springframework.util.MimeTypeUtils;

/**
 * 发件箱投递器：定时扫描未发送事件，通过 {@link StreamBridge} 发送到绑定的 Kafka topic / RabbitMQ exchange。
 * <p>
 * 多副本部署时依靠 {@code SELECT ... FOR UPDATE SKIP LOCKED}（MySQL 8+ / PostgreSQL 9.5+）互不抢占同一批事件。
 * 投递语义为 at-least-once，消费端必须幂等（见 {@link IdempotentConsumer}）。
 * <p>
 * 指标：{@code acme_outbox_published_total{binding,type,result}}；积压量见 {@code acme_outbox_pending}。
 */
public class OutboxRelay {

    private static final Logger log = LoggerFactory.getLogger(OutboxRelay.class);

    private final JdbcTemplate jdbcTemplate;
    private final TransactionTemplate transactionTemplate;
    private final StreamBridge streamBridge;
    private final OutboxProperties properties;
    private final Clock clock;
    private final MessagingTracing tracing;
    private final MeterRegistry meterRegistry;

    public OutboxRelay(JdbcTemplate jdbcTemplate, TransactionTemplate transactionTemplate, StreamBridge streamBridge,
            OutboxProperties properties, Clock clock, MessagingTracing tracing, MeterRegistry meterRegistry) {
        this.jdbcTemplate = jdbcTemplate;
        this.transactionTemplate = readCommitted(transactionTemplate);
        this.streamBridge = streamBridge;
        this.properties = properties;
        this.clock = clock;
        this.tracing = tracing;
        this.meterRegistry = meterRegistry;
    }

    /**
     * 投递事务使用 READ COMMITTED。MySQL 默认的 REPEATABLE READ 下，{@code SELECT ... FOR UPDATE} 会对
     * (status, created_at) 索引加间隙锁，投递期间（MQ 变慢或不可用时可能持续数十秒）会阻塞业务事务插入新事件，
     * 表现为下单超时。READ COMMITTED 只锁定读到的行，配合 SKIP LOCKED 多副本互不阻塞。PostgreSQL 默认即为 READ COMMITTED。
     */
    private static TransactionTemplate readCommitted(TransactionTemplate template) {
        TransactionTemplate copy = new TransactionTemplate(template.getTransactionManager(), template);
        copy.setIsolationLevel(TransactionDefinition.ISOLATION_READ_COMMITTED);
        return copy;
    }

    /** MQ 持续不可用时两次投递尝试之间的最长间隔。 */
    static final Duration MAX_BACKOFF = Duration.ofSeconds(30);

    private Duration backoff = Duration.ZERO;
    private Instant nextAttemptAt = Instant.MIN;

    /**
     * 投递所有待发送事件，由调度器按 {@code acme.mq.outbox.poll-interval} 周期调用。
     * 投递失败（通常是 MQ 不可用）时指数退避，最长 {@link #MAX_BACKOFF}；成功后恢复正常节奏。
     */
    public void relay() {
        Instant now = Instant.now(clock);
        if (now.isBefore(nextAttemptAt)) {
            return;
        }
        BatchResult result;
        do {
            result = transactionTemplate.execute(status -> relayBatch());
        } while (result != null && !result.failed() && result.sent() == properties.batchSize());
        if (result != null && result.failed()) {
            backoff = backoff.isZero() ? properties.pollInterval() : backoff.multipliedBy(2);
            if (backoff.compareTo(MAX_BACKOFF) > 0) {
                backoff = MAX_BACKOFF;
            }
            nextAttemptAt = Instant.now(clock).plus(backoff);
            log.warn("发件箱投递失败，{} 秒后重试", backoff.toSeconds());
        } else {
            backoff = Duration.ZERO;
            nextAttemptAt = Instant.MIN;
        }
    }

    /** 当前待发送（含重试中）的事件数，供 {@code acme_outbox_pending} 指标使用。 */
    public long pendingCount() {
        Long count = jdbcTemplate.queryForObject("SELECT COUNT(*) FROM outbox_event WHERE status = 0", Long.class);
        return count == null ? 0 : count;
    }

    /**
     * 在一个事务中锁定并投递一批事件。遇到第一条失败即停止：MQ 不可用时整批都会失败，
     * 逐条重试只会白白耗尽每条事件的重试次数，也会打乱同一聚合的事件顺序。
     */
    BatchResult relayBatch() {
        List<OutboxEvent> events = jdbcTemplate.query("""
                SELECT event_id, binding_name, event_type, aggregate_id, payload, source, created_at, trace_parent
                FROM outbox_event
                WHERE status = 0 AND retry_count < ?
                ORDER BY created_at
                LIMIT ?
                """ + properties.lockClause(),
                (rs, i) -> new OutboxEvent(rs.getString(1), rs.getString(2), rs.getString(3), rs.getString(4),
                        rs.getString(5), rs.getString(6), rs.getTimestamp(7).toInstant(), rs.getString(8)),
                properties.maxRetries(), properties.batchSize());
        int sent = 0;
        for (OutboxEvent event : events) {
            boolean ok = send(event);
            counter(event, ok ? "success" : "failure").increment();
            if (ok) {
                jdbcTemplate.update("UPDATE outbox_event SET status = 1, sent_at = ? WHERE event_id = ?",
                        Timestamp.from(Instant.now(clock)), event.eventId());
                sent++;
            } else {
                jdbcTemplate.update("UPDATE outbox_event SET retry_count = retry_count + 1 WHERE event_id = ?",
                        event.eventId());
                return new BatchResult(sent, true);
            }
        }
        return new BatchResult(sent, false);
    }

    record BatchResult(int sent, boolean failed) {
    }


    private boolean send(OutboxEvent event) {
        // 以下单请求的链路为父节点，投递动作在链路图中显示为 order-service 的一个 PRODUCER Span
        Span span = tracing.spanBuilder(event.traceParent())
                .name("publish " + event.eventType())
                .kind(Span.Kind.PRODUCER)
                .tag("messaging.destination.name", event.bindingName())
                .tag("messaging.message.id", event.eventId())
                .start();
        try (Tracer.SpanInScope ignored = tracing.tracer().withSpan(span)) {
            MessageBuilder<byte[]> builder = MessageBuilder.withPayload(event.payload().getBytes(StandardCharsets.UTF_8))
                    .setHeader("contentType", MimeTypeUtils.APPLICATION_JSON_VALUE)
                    .setHeader(EventHeaders.ID, event.eventId())
                    .setHeader(EventHeaders.TYPE, event.eventType())
                    .setHeader(EventHeaders.SOURCE, event.source())
                    .setHeader(EventHeaders.TIME, event.createdAt().toString())
                    .setHeader(EventHeaders.KEY, event.aggregateId());
            tracing.inject(span.context()).forEach(builder::setHeader);
            boolean ok = streamBridge.send(event.bindingName(), builder.build());
            if (!ok) {
                span.error(new IllegalStateException("StreamBridge.send 返回 false"));
            }
            return ok;
        } catch (RuntimeException ex) {
            span.error(ex);
            log.warn("投递发件箱事件失败 eventId={} type={}", event.eventId(), event.eventType(), ex);
            return false;
        } finally {
            span.end();
        }
    }

    private Counter counter(OutboxEvent event, String result) {
        return Counter.builder("acme.outbox.published")
                .description("发件箱投递次数")
                .tag("binding", event.bindingName())
                .tag("type", event.eventType())
                .tag("result", result)
                .register(meterRegistry);
    }

    record OutboxEvent(String eventId, String bindingName, String eventType, String aggregateId, String payload,
            String source, Instant createdAt, String traceParent) {
    }
}
