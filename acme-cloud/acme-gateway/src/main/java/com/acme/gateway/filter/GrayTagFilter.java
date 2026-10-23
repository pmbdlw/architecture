package com.acme.gateway.filter;

import org.springframework.cloud.gateway.filter.GatewayFilterChain;
import org.springframework.cloud.gateway.filter.GlobalFilter;
import org.springframework.cloud.gateway.filter.ReactiveLoadBalancerClientFilter;
import org.springframework.core.Ordered;
import org.springframework.security.core.context.ReactiveSecurityContextHolder;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;
import org.springframework.stereotype.Component;
import org.springframework.util.StringUtils;
import org.springframework.web.server.ServerWebExchange;

import com.acme.framework.core.constant.AcmeHeaders;
import com.acme.gateway.config.GrayProperties;

import reactor.core.publisher.Mono;

/**
 * 灰度打标：按规则给请求加上 {@code X-Gray-Tag}，之后由灰度 LoadBalancer 选择对应版本实例，
 * 下游服务通过 Feign 拦截器继续透传，实现“全链路灰度”。
 */
@Component
public class GrayTagFilter implements GlobalFilter, Ordered {

    private final GrayProperties gray;

    public GrayTagFilter(GrayProperties gray) {
        this.gray = gray;
    }

    @Override
    public Mono<Void> filter(ServerWebExchange exchange, GatewayFilterChain chain) {
        String clientTag = exchange.getRequest().getHeaders().getFirst(AcmeHeaders.GRAY_TAG);
        return ReactiveSecurityContextHolder.getContext()
                .map(context -> context.getAuthentication())
                .filter(JwtAuthenticationToken.class::isInstance)
                .map(auth -> String.valueOf(((JwtAuthenticationToken) auth).getToken().getClaims().get("uid")))
                .defaultIfEmpty("")
                .flatMap(uid -> chain.filter(withTag(exchange, resolveTag(clientTag, uid))));
    }

    String resolveTag(String clientTag, String uid) {
        if (gray.allowClientHeader() && StringUtils.hasText(clientTag)) {
            return clientTag;
        }
        if (!gray.enabled() || !StringUtils.hasText(gray.tag()) || !StringUtils.hasText(uid)) {
            return null;
        }
        if (gray.uids().contains(uid) || Math.floorMod(uid.hashCode(), 100) < gray.percentage()) {
            return gray.tag();
        }
        return null;
    }

    private static ServerWebExchange withTag(ServerWebExchange exchange, String tag) {
        // 先移除客户端自带的值，防止外部伪造灰度标签
        return exchange.mutate().request(request -> request.headers(headers -> {
            headers.remove(AcmeHeaders.GRAY_TAG);
            if (tag != null) {
                headers.set(AcmeHeaders.GRAY_TAG, tag);
            }
        })).build();
    }

    @Override
    public int getOrder() {
        // 必须在负载均衡过滤器之前执行
        return ReactiveLoadBalancerClientFilter.LOAD_BALANCER_CLIENT_FILTER_ORDER - 100;
    }
}
