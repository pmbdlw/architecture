package com.acme.order.domain;

import com.acme.framework.core.exception.ErrorCode;

/**
 * order-service 错误码，号段 A2000-A2999。
 */
public enum OrderErrorCode implements ErrorCode {

    ORDER_NOT_FOUND("A2001", "订单不存在: %s", 404),
    BUYER_NOT_AVAILABLE("A2002", "下单用户不存在或已被禁用", 422),
    ORDER_STATUS_INVALID("A2003", "订单状态不允许该操作: %s", 409);

    private final String code;
    private final String message;
    private final int httpStatus;

    OrderErrorCode(String code, String message, int httpStatus) {
        this.code = code;
        this.message = message;
        this.httpStatus = httpStatus;
    }

    @Override
    public String code() {
        return code;
    }

    @Override
    public String message() {
        return message;
    }

    @Override
    public int httpStatus() {
        return httpStatus;
    }
}
