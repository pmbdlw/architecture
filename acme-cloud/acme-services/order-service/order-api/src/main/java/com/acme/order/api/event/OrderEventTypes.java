package com.acme.order.api.event;

/**
 * 订单域事件类型与 Topic 定义。
 */
public final class OrderEventTypes {

    /** Kafka topic / RabbitMQ exchange 名，格式 {@code <系统>.<领域>.events}。 */
    public static final String DESTINATION = "acme.order.events";

    public static final String ORDER_CREATED = "order.created";

    private OrderEventTypes() {
    }
}
