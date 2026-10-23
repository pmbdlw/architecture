package com.acme.framework.security;

import org.slf4j.MDC;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.http.ResponseEntity;
import org.springframework.security.authorization.AuthorizationDeniedException;
import org.springframework.security.core.AuthenticationException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

import com.acme.framework.core.exception.CommonErrorCode;
import com.acme.framework.core.result.Result;

/**
 * {@code @PreAuthorize} 等方法级鉴权失败发生在 Controller 内部，会先到达 MVC 异常处理器；
 * 优先于 GlobalExceptionHandler 的兜底分支，转换为 401/403。
 */
@RestControllerAdvice
@Order(Ordered.HIGHEST_PRECEDENCE)
public class SecurityExceptionHandler {

    @ExceptionHandler(AuthorizationDeniedException.class)
    public ResponseEntity<Result<Void>> handleDenied(AuthorizationDeniedException ex) {
        return build(CommonErrorCode.FORBIDDEN);
    }

    @ExceptionHandler(AuthenticationException.class)
    public ResponseEntity<Result<Void>> handleUnauthenticated(AuthenticationException ex) {
        return build(CommonErrorCode.UNAUTHORIZED);
    }

    private static ResponseEntity<Result<Void>> build(CommonErrorCode errorCode) {
        return ResponseEntity.status(errorCode.httpStatus())
                .body(Result.fail(errorCode, errorCode.message(), MDC.get("traceId")));
    }
}
