package com.acme.framework.mq;

import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.tracing.Span;
import io.micrometer.tracing.Tracer;
import java.nio.charset.StandardCharsets;
import java.sql.Timestamp;
import java.time.Clock;
import java.time.Instant;
import java.util.function.Consumer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.dao.DuplicateKeyException;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.messaging.Message;
import org.springframework.transaction.support.TransactionTemplate;

/**
 * 消费幂等模板：在同一个本地事务中“登记事件 ID + 执行业务”，
 * 重复投递的消息因主键冲突被识别并跳过。
 * <p>
 * 同时从消息头 {@code traceparent} 恢复链路，创建 CONSUMER Span，并记录
 * {@code acme_mq_consumed_total{consumer,type,result}}（result = processed / duplicate / failed）。
 *
 * <pre>{@code
 * @Bean
 * Consumer<Message<OrderCreatedEvent>> orderCreated(IdempotentConsumer idempotent) {
 *     return message -> idempotent.consume(message, event -> notificationService.notify(event));
 * }
 * }</pre>
 */
public class IdempotentConsumer {

    private static final Logger log = LoggerFactory.getLogger(IdempotentConsumer.class);

    private final JdbcTemplate jdbcTemplate;
    private final TransactionTemplate transactionTemplate;
    private final String consumer;
    private final Clock clock;
    private final MessagingTracing tracing;
    private final MeterRegistry meterRegistry;

    public IdempotentConsumer(JdbcTemplate jdbcTemplate, TransactionTemplate transactionTemplate, String consumer,
            Clock clock, MessagingTracing tracing, MeterRegistry meterRegistry) {
        this.jdbcTemplate = jdbcTemplate;
        this.transactionTemplate = transactionTemplate;
        this.consumer = consumer;
        this.clock = clock;
        this.tracing = tracing;
        this.meterRegistry = meterRegistry;
    }

    /**
     * @return true 表示本次执行了业务逻辑；false 表示重复消息已跳过
     */
    public <T> boolean consume(Message<T> message, Consumer<T> handler) {
        String eventId = message.getHeaders().get(EventHeaders.ID, String.class);
        if (eventId == null) {
            throw new IllegalArgumentException("消息缺少 " + EventHeaders.ID + " 头，无法保证幂等");
        }
        String eventType = String.valueOf(message.getHeaders().getOrDefault(EventHeaders.TYPE, "unknown"));
        Span span = tracing.spanBuilder(header(message, MessagingTracing.TRACEPARENT))
                .name("process " + eventType)
                .kind(Span.Kind.CONSUMER)
                .tag("messaging.consumer.group.name", consumer)
                .tag("messaging.message.id", eventId)
                .start();
        try (Tracer.SpanInScope ignored = tracing.tracer().withSpan(span)) {
            Boolean handled = transactionTemplate.execute(status -> {
                try {
                    jdbcTemplate.update(
                            "INSERT INTO consumed_event (consumer, event_id, consumed_at) VALUES (?, ?, ?)",
                            consumer, eventId, Timestamp.from(Instant.now(clock)));
                } catch (DuplicateKeyException ex) {
                    log.info("重复消息已跳过 consumer={} eventId={}", consumer, eventId);
                    return false;
                }
                handler.accept(message.getPayload());
                return true;
            });
            boolean processed = Boolean.TRUE.equals(handled);
            counter(eventType, processed ? "processed" : "duplicate").increment();
            return processed;
        } catch (RuntimeException ex) {
            span.error(ex);
            counter(eventType, "failed").increment();
            throw ex;
        } finally {
            span.end();
        }
    }

    private static String header(Message<?> message, String name) {
        Object value = message.getHeaders().get(name);
        if (value instanceof byte[] bytes) {
            return new String(bytes, StandardCharsets.UTF_8);
        }
        return value == null ? null : value.toString();
    }

    private Counter counter(String eventType, String result) {
        return Counter.builder("acme.mq.consumed")
                .description("消费事件次数")
                .tag("consumer", consumer)
                .tag("type", eventType)
                .tag("result", result)
                .register(meterRegistry);
    }
}
