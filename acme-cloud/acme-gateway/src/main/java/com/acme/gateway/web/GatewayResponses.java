package com.acme.gateway.web;

import java.nio.charset.StandardCharsets;

import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.server.reactive.ServerHttpResponse;

import com.acme.framework.core.exception.ErrorCode;
import com.acme.framework.core.result.Result;

import reactor.core.publisher.Mono;
import tools.jackson.databind.json.JsonMapper;

/**
 * 网关直接返回的响应也使用统一 {@link Result} 结构。
 */
public final class GatewayResponses {

    private static final JsonMapper JSON = JsonMapper.builder().build();

    private GatewayResponses() {
    }

    public static Mono<Void> write(ServerHttpResponse response, ErrorCode errorCode) {
        response.setStatusCode(HttpStatus.valueOf(errorCode.httpStatus()));
        response.getHeaders().setContentType(MediaType.APPLICATION_JSON);
        byte[] body = JSON.writeValueAsString(Result.fail(errorCode, errorCode.message(), null))
                .getBytes(StandardCharsets.UTF_8);
        return response.writeWith(Mono.just(response.bufferFactory().wrap(body)));
    }
}
