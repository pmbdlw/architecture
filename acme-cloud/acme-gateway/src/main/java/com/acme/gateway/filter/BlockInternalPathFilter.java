package com.acme.gateway.filter;

import org.springframework.core.Ordered;
import org.springframework.stereotype.Component;
import org.springframework.web.server.ServerWebExchange;
import org.springframework.web.server.WebFilter;
import org.springframework.web.server.WebFilterChain;

import com.acme.framework.core.exception.CommonErrorCode;
import com.acme.gateway.web.GatewayResponses;

import reactor.core.publisher.Mono;

/**
 * 服务的 /internal/** 接口只允许集群内调用，任何经网关进来的请求一律 404（不暴露其存在）。
 * <p>
 * 实现为 WebFilter 且优先级最高：要早于 Spring Security 的认证过滤器，否则未登录请求会先得到 401，
 * 间接暴露了内部接口的存在。
 */
@Component
public class BlockInternalPathFilter implements WebFilter, Ordered {

    @Override
    public Mono<Void> filter(ServerWebExchange exchange, WebFilterChain chain) {
        String path = exchange.getRequest().getPath().value();
        if (path.contains("/internal/")) {
            return GatewayResponses.write(exchange.getResponse(), CommonErrorCode.NOT_FOUND);
        }
        return chain.filter(exchange);
    }

    @Override
    public int getOrder() {
        return Ordered.HIGHEST_PRECEDENCE;
    }
}
