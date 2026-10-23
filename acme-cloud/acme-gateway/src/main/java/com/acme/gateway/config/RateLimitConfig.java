package com.acme.gateway.config;

import org.springframework.cloud.gateway.filter.ratelimit.KeyResolver;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import reactor.core.publisher.Mono;

/**
 * 限流维度：已登录按用户（JWT sub），未登录按客户端 IP。
 */
@Configuration(proxyBeanMethods = false)
public class RateLimitConfig {

    @Bean
    public KeyResolver principalOrIpKeyResolver() {
        return exchange -> exchange.getPrincipal()
                .map(principal -> "user:" + principal.getName())
                .switchIfEmpty(Mono.fromSupplier(() -> {
                    var address = exchange.getRequest().getRemoteAddress();
                    return "ip:" + (address == null ? "unknown" : address.getAddress().getHostAddress());
                }));
    }
}
