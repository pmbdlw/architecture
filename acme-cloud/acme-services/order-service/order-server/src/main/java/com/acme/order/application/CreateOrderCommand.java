package com.acme.order.application;

import java.math.BigDecimal;

public record CreateOrderCommand(Long userId, String productCode, int quantity, BigDecimal amount) {
}
