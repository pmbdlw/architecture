package com.acme.order.api.event;

import java.math.BigDecimal;
import java.time.Instant;

/**
 * 订单已创建事件（v1）。新增字段只能追加且消费方需容忍未知字段；删除/改名字段必须发布新事件类型。
 */
public record OrderCreatedEvent(Long orderId, String orderNo, Long userId, String productCode, int quantity,
        BigDecimal amount, Instant createdAt) {
}
