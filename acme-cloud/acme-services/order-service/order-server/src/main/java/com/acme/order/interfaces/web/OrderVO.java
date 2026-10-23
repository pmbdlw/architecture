package com.acme.order.interfaces.web;

import java.math.BigDecimal;
import java.time.Instant;

import com.acme.order.domain.model.Order;

public record OrderVO(Long id, String orderNo, String productCode, int quantity, BigDecimal amount, String status,
        Instant createdAt) {

    static OrderVO from(Order order) {
        return new OrderVO(order.getId(), order.getOrderNo(), order.getProductCode(), order.getQuantity(),
                order.getAmount(), order.getStatus().name(), order.getCreatedAt());
    }
}
