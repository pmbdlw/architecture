package com.acme.framework.core.exception;

/**
 * 下游服务返回的业务错误码（4xx），由 Feign ErrorDecoder 还原后原样向上抛出，
 * 调用方可按 code 判断并转换为自己的错误码。
 */
public record RemoteErrorCode(String code, String message, int httpStatus) implements ErrorCode {
}
