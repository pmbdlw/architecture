package com.acme.user.domain;

import com.acme.framework.core.exception.ErrorCode;

/**
 * user-service 错误码，号段 A1000-A1999。
 */
public enum UserErrorCode implements ErrorCode {

    USER_NOT_FOUND("A1001", "用户不存在: %s", 404),
    USERNAME_TAKEN("A1002", "用户名已被占用: %s", 409),
    USER_ALREADY_DISABLED("A1003", "用户已被禁用", 409);

    private final String code;
    private final String message;
    private final int httpStatus;

    UserErrorCode(String code, String message, int httpStatus) {
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
