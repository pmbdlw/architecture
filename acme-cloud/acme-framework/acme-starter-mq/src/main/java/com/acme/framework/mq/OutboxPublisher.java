package com.acme.framework.mq;

import java.sql.Timestamp;
import java.time.Clock;
import java.time.Instant;
import java.util.UUID;

import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.transaction.support.TransactionSynchronizationManager;

import tools.jackson.databind.json.JsonMapper;

/**
 * 事务发件箱（Transactional Outbox）写入端。
 * <p>
 * 业务代码在<b>本地事务内</b>调用 {@link #publish}，事件与业务数据同库同事务落盘，
 * 由 {@link OutboxRelay} 异步投递到 Kafka/RabbitMQ。这样“改数据”与“发消息”要么都成功、要么都失败，
 * 无需分布式事务（Seata）即可实现最终一致性。
 */
public class OutboxPublisher {

    private static final String INSERT_SQL = """
            INSERT INTO outbox_event (event_id, binding_name, event_type, aggregate_id, payload, source, status,
                                      retry_count, created_at)
            VALUES (?, ?, ?, ?, ?, ?, 0, 0, ?)
            """;

    private final JdbcTemplate jdbcTemplate;
    private final JsonMapper jsonMapper;
    private final String source;
    private final Clock clock;

    public OutboxPublisher(JdbcTemplate jdbcTemplate, JsonMapper jsonMapper, String source, Clock clock) {
        this.jdbcTemplate = jdbcTemplate;
        this.jsonMapper = jsonMapper;
        this.source = source;
        this.clock = clock;
    }

    /**
     * @param binding     Spring Cloud Stream 输出绑定名，如 {@code orderEvents-out-0}
     * @param eventType   事件类型，如 {@code order.created}
     * @param aggregateId 聚合根 ID，作为分区键
     * @param payload     事件数据（record），序列化为 JSON
     * @return 事件 ID
     */
    public String publish(String binding, String eventType, String aggregateId, Object payload) {
        if (!TransactionSynchronizationManager.isActualTransactionActive()) {
            throw new IllegalStateException("OutboxPublisher.publish 必须在 @Transactional 方法内调用");
        }
        String eventId = UUID.randomUUID().toString();
        jdbcTemplate.update(INSERT_SQL, eventId, binding, eventType, aggregateId,
                jsonMapper.writeValueAsString(payload), source, Timestamp.from(Instant.now(clock)));
        return eventId;
    }
}
