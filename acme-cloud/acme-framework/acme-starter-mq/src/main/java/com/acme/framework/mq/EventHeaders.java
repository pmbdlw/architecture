package com.acme.framework.mq;

/**
 * 事件消息头约定（参考 CloudEvents 的二进制模式，Kafka/RabbitMQ 通用）。
 * 消息体只放业务数据 JSON，元数据全部放在消息头中。
 */
public final class EventHeaders {

    /** 事件唯一 ID（UUID），消费端据此做幂等。 */
    public static final String ID = "ce_id";

    /** 事件类型，格式 {@code <领域>.<过去式动作>}，如 {@code order.created}。RabbitMQ 用作 routing key。 */
    public static final String TYPE = "ce_type";

    /** 事件来源服务，即 spring.application.name。 */
    public static final String SOURCE = "ce_source";

    /** 事件发生时间，ISO-8601。 */
    public static final String TIME = "ce_time";

    /** 分区键（聚合根 ID）。Kafka 用作消息 key，保证同一聚合的事件有序。 */
    public static final String KEY = "acme_key";

    private EventHeaders() {
    }
}
