package com.acme.framework.core.exception;

/**
 * 平台通用错误码（号段 0000-0999），所有服务共用。
 */
public enum CommonErrorCode implements ErrorCode {

    SUCCESS("00000", "成功", 200),

    BAD_REQUEST("A0400", "请求参数错误", 400),
    UNAUTHORIZED("A0401", "未登录或令牌已失效", 401),
    FORBIDDEN("A0403", "无访问权限", 403),
    NOT_FOUND("A0404", "资源不存在", 404),
    CONFLICT("A0409", "资源状态冲突", 409),
    TOO_MANY_REQUESTS("A0429", "请求过于频繁", 429),

    SYSTEM_ERROR("B0001", "系统繁忙，请稍后再试", 500),

    REMOTE_SERVICE_ERROR("C0001", "下游服务调用失败", 502),
    REMOTE_SERVICE_UNAVAILABLE("C0503", "下游服务暂不可用", 503);

    private final String code;
    private final String message;
    private final int httpStatus;

    CommonErrorCode(String code, String message, int httpStatus) {
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
