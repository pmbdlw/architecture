package com.acme.framework.security;

import java.io.IOException;

import jakarta.servlet.http.HttpServletResponse;

import org.slf4j.MDC;
import org.springframework.http.MediaType;
import org.springframework.security.web.AuthenticationEntryPoint;
import org.springframework.security.web.access.AccessDeniedHandler;

import com.acme.framework.core.exception.CommonErrorCode;
import com.acme.framework.core.result.Result;

import tools.jackson.databind.json.JsonMapper;

/**
 * 401 / 403 也输出统一的 {@link Result} 结构。
 */
final class ResultAuthenticationHandlers {

    private static final JsonMapper JSON = JsonMapper.builder().build();

    private ResultAuthenticationHandlers() {
    }

    static AuthenticationEntryPoint entryPoint() {
        return (request, response, ex) -> write(response, CommonErrorCode.UNAUTHORIZED);
    }

    static AccessDeniedHandler accessDeniedHandler() {
        return (request, response, ex) -> write(response, CommonErrorCode.FORBIDDEN);
    }

    private static void write(HttpServletResponse response, CommonErrorCode errorCode) throws IOException {
        response.setStatus(errorCode.httpStatus());
        response.setContentType(MediaType.APPLICATION_JSON_VALUE);
        response.setCharacterEncoding("UTF-8");
        JSON.writeValue(response.getOutputStream(), Result.fail(errorCode, errorCode.message(), MDC.get("traceId")));
    }
}
