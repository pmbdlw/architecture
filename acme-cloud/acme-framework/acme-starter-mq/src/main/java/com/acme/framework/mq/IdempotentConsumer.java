package com.acme.framework.mq;

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

    public IdempotentConsumer(JdbcTemplate jdbcTemplate, TransactionTemplate transactionTemplate, String consumer,
            Clock clock) {
        this.jdbcTemplate = jdbcTemplate;
        this.transactionTemplate = transactionTemplate;
        this.consumer = consumer;
        this.clock = clock;
    }

    /**
     * @return true 表示本次执行了业务逻辑；false 表示重复消息已跳过
     */
    public <T> boolean consume(Message<T> message, Consumer<T> handler) {
        String eventId = message.getHeaders().get(EventHeaders.ID, String.class);
        if (eventId == null) {
            throw new IllegalArgumentException("消息缺少 " + EventHeaders.ID + " 头，无法保证幂等");
        }
        Boolean handled = transactionTemplate.execute(status -> {
            try {
                jdbcTemplate.update("INSERT INTO consumed_event (consumer, event_id, consumed_at) VALUES (?, ?, ?)",
                        consumer, eventId, Timestamp.from(Instant.now(clock)));
            } catch (DuplicateKeyException ex) {
                log.info("重复消息已跳过 consumer={} eventId={}", consumer, eventId);
                return false;
            }
            handler.accept(message.getPayload());
            return true;
        });
        return Boolean.TRUE.equals(handled);
    }
}
