package com.acme.framework.web;

import com.acme.framework.core.result.Result;
import org.slf4j.MDC;
import org.springframework.core.MethodParameter;
import org.springframework.http.MediaType;
import org.springframework.http.converter.HttpMessageConverter;
import org.springframework.http.server.ServerHttpRequest;
import org.springframework.http.server.ServerHttpResponse;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.servlet.mvc.method.annotation.ResponseBodyAdvice;

/**
 * 为成功响应的 {@link Result} 补上 traceId（失败响应由 {@link GlobalExceptionHandler} 填写），
 * 统一响应体 {@code {code, message, data, traceId}} 中的 traceId 始终与日志、链路系统一致。
 */
@RestControllerAdvice
public class ResultTraceIdAdvice implements ResponseBodyAdvice<Object> {

    @Override
    public boolean supports(MethodParameter returnType, Class<? extends HttpMessageConverter<?>> converterType) {
        return true;
    }

    @Override
    public Object beforeBodyWrite(Object body, MethodParameter returnType, MediaType selectedContentType,
            Class<? extends HttpMessageConverter<?>> selectedConverterType, ServerHttpRequest request,
            ServerHttpResponse response) {
        if (body instanceof Result<?> result && result.traceId() == null) {
            String traceId = MDC.get("traceId");
            if (traceId != null) {
                return new Result<>(result.code(), result.message(), result.data(), traceId);
            }
        }
        return body;
    }
}
