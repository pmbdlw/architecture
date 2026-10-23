package com.acme.framework.core.exception;

/**
 * 业务异常：表示“可预期的失败”，由全局异常处理器转换为统一响应，不打印堆栈。
 * 非预期错误（空指针、SQL 异常等）不要包装成 BizException，交给全局处理器按 B0001 处理并告警。
 */
public class BizException extends RuntimeException {

    private final transient ErrorCode errorCode;

    public BizException(ErrorCode errorCode) {
        super(errorCode.message());
        this.errorCode = errorCode;
    }

    /**
     * @param messageArgs 用于格式化 {@link ErrorCode#message()} 中 {@code %s} 占位符的参数
     */
    public BizException(ErrorCode errorCode, Object... messageArgs) {
        super(String.format(errorCode.message(), messageArgs));
        this.errorCode = errorCode;
    }

    public BizException(ErrorCode errorCode, String message, Throwable cause) {
        super(message, cause);
        this.errorCode = errorCode;
    }

    public ErrorCode getErrorCode() {
        return errorCode;
    }
}
