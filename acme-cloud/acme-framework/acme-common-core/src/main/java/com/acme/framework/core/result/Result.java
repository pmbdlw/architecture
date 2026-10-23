package com.acme.framework.core.result;

import com.acme.framework.core.exception.CommonErrorCode;
import com.acme.framework.core.exception.ErrorCode;

/**
 * 统一响应体。所有对外（/api/**）与对内（/internal/**）HTTP 接口都返回该结构，
 * 同时 HTTP 状态码与 {@link ErrorCode#httpStatus()} 保持一致。
 *
 * @param code    业务码，成功为 {@code 00000}
 * @param message 提示语
 * @param data    业务数据
 * @param traceId 链路 ID，便于用户反馈问题时定位日志
 */
public record Result<T>(String code, String message, T data, String traceId) {

    public static final String SUCCESS_CODE = CommonErrorCode.SUCCESS.code();

    public static <T> Result<T> ok(T data) {
        return new Result<>(SUCCESS_CODE, CommonErrorCode.SUCCESS.message(), data, null);
    }

    public static Result<Void> ok() {
        return ok(null);
    }

    public static <T> Result<T> fail(ErrorCode errorCode, String message, String traceId) {
        return new Result<>(errorCode.code(), message, null, traceId);
    }

    public boolean isSuccess() {
        return SUCCESS_CODE.equals(code);
    }
}
