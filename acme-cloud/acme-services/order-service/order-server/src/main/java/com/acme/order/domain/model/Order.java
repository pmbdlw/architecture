package com.acme.order.domain.model;

import java.math.BigDecimal;
import java.time.Instant;

import com.acme.framework.core.exception.BizException;
import com.acme.order.domain.OrderErrorCode;

/**
 * 订单聚合根。
 */
public class Order {

    private final Long id;
    private final String orderNo;
    private final Long userId;
    private final String productCode;
    private final int quantity;
    private final BigDecimal amount;
    private OrderStatus status;
    private final Instant createdAt;

    public Order(Long id, String orderNo, Long userId, String productCode, int quantity, BigDecimal amount,
            OrderStatus status, Instant createdAt) {
        this.id = id;
        this.orderNo = orderNo;
        this.userId = userId;
        this.productCode = productCode;
        this.quantity = quantity;
        this.amount = amount;
        this.status = status;
        this.createdAt = createdAt;
    }

    public static Order create(Long id, Long userId, String productCode, int quantity, BigDecimal amount,
            Instant now) {
        return new Order(id, "O" + id, userId, productCode, quantity, amount, OrderStatus.CREATED, now);
    }

    public void cancel() {
        if (status != OrderStatus.CREATED) {
            throw new BizException(OrderErrorCode.ORDER_STATUS_INVALID, status);
        }
        status = OrderStatus.CANCELLED;
    }

    public boolean belongsTo(Long buyerId) {
        return userId.equals(buyerId);
    }

    public Long getId() {
        return id;
    }

    public String getOrderNo() {
        return orderNo;
    }

    public Long getUserId() {
        return userId;
    }

    public String getProductCode() {
        return productCode;
    }

    public int getQuantity() {
        return quantity;
    }

    public BigDecimal getAmount() {
        return amount;
    }

    public OrderStatus getStatus() {
        return status;
    }

    public Instant getCreatedAt() {
        return createdAt;
    }
}
