package com.acme.framework.web;

import java.util.stream.Collectors;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.MDC;
import org.springframework.http.ResponseEntity;
import org.springframework.http.converter.HttpMessageNotReadableException;
import org.springframework.validation.FieldError;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.MissingServletRequestParameterException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.method.annotation.HandlerMethodValidationException;
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException;
import org.springframework.web.servlet.resource.NoResourceFoundException;

import com.acme.framework.core.exception.BizException;
import com.acme.framework.core.exception.CommonErrorCode;
import com.acme.framework.core.exception.ErrorCode;
import com.acme.framework.core.exception.RemoteServiceException;
import com.acme.framework.core.result.Result;

/**
 * 全局异常处理：把所有异常收敛为 {@link Result} + 正确的 HTTP 状态码。
 * <ul>
 *   <li>BizException：可预期的业务失败，记 INFO/WARN，不打堆栈；</li>
 *   <li>参数类异常：A0400；</li>
 *   <li>RemoteServiceException：下游 5xx/熔断，C0001/C0503；</li>
 *   <li>其它异常：B0001，记 ERROR 并带堆栈，触发告警。</li>
 * </ul>
 */
@RestControllerAdvice
public class GlobalExceptionHandler {

    private static final Logger log = LoggerFactory.getLogger(GlobalExceptionHandler.class);

    @ExceptionHandler(BizException.class)
    public ResponseEntity<Result<Void>> handleBiz(BizException ex) {
        ErrorCode errorCode = ex.getErrorCode();
        if (errorCode.httpStatus() >= 500) {
            log.warn("业务异常 [{}] {}", errorCode.code(), ex.getMessage(), ex.getCause());
        } else {
            log.info("业务异常 [{}] {}", errorCode.code(), ex.getMessage());
        }
        return build(errorCode, ex.getMessage());
    }

    @ExceptionHandler(RemoteServiceException.class)
    public ResponseEntity<Result<Void>> handleRemote(RemoteServiceException ex) {
        log.warn("下游服务异常: {}", ex.getMessage());
        CommonErrorCode errorCode = ex.getStatus() == 503
                ? CommonErrorCode.REMOTE_SERVICE_UNAVAILABLE
                : CommonErrorCode.REMOTE_SERVICE_ERROR;
        return build(errorCode, errorCode.message());
    }

    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ResponseEntity<Result<Void>> handleInvalidBody(MethodArgumentNotValidException ex) {
        String message = ex.getBindingResult().getFieldErrors().stream()
                .map(GlobalExceptionHandler::describe)
                .collect(Collectors.joining("; "));
        return build(CommonErrorCode.BAD_REQUEST, message);
    }

    @ExceptionHandler(HandlerMethodValidationException.class)
    public ResponseEntity<Result<Void>> handleInvalidParam(HandlerMethodValidationException ex) {
        String message = ex.getAllErrors().stream()
                .map(error -> error.getDefaultMessage())
                .collect(Collectors.joining("; "));
        return build(CommonErrorCode.BAD_REQUEST, message);
    }

    @ExceptionHandler({HttpMessageNotReadableException.class, MissingServletRequestParameterException.class,
            MethodArgumentTypeMismatchException.class})
    public ResponseEntity<Result<Void>> handleBadRequest(Exception ex) {
        return build(CommonErrorCode.BAD_REQUEST, CommonErrorCode.BAD_REQUEST.message());
    }

    @ExceptionHandler(NoResourceFoundException.class)
    public ResponseEntity<Result<Void>> handleNotFound(NoResourceFoundException ex) {
        return build(CommonErrorCode.NOT_FOUND, CommonErrorCode.NOT_FOUND.message());
    }

    @ExceptionHandler(Exception.class)
    public ResponseEntity<Result<Void>> handleUnknown(Exception ex) {
        log.error("未处理异常", ex);
        return build(CommonErrorCode.SYSTEM_ERROR, CommonErrorCode.SYSTEM_ERROR.message());
    }

    static ResponseEntity<Result<Void>> build(ErrorCode errorCode, String message) {
        return ResponseEntity.status(errorCode.httpStatus())
                .body(Result.fail(errorCode, message, MDC.get("traceId")));
    }

    private static String describe(FieldError error) {
        return error.getField() + " " + error.getDefaultMessage();
    }
}
