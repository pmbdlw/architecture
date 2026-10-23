package com.acme.gateway.web;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import com.acme.framework.core.exception.CommonErrorCode;
import com.acme.framework.core.result.Result;

/**
 * 路由熔断打开或下游超时后的降级响应（CircuitBreaker 过滤器的 fallbackUri）。
 */
@RestController
public class FallbackController {

    @RequestMapping("/fallback/{service}")
    public ResponseEntity<Result<Void>> fallback(@PathVariable String service) {
        CommonErrorCode errorCode = CommonErrorCode.REMOTE_SERVICE_UNAVAILABLE;
        return ResponseEntity.status(errorCode.httpStatus())
                .body(Result.fail(errorCode, service + " 暂不可用，请稍后重试", null));
    }
}
