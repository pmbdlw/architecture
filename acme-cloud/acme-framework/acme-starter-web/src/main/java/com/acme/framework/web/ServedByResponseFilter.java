package com.acme.framework.web;

import com.acme.framework.core.constant.AcmeHeaders;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import org.springframework.web.filter.OncePerRequestFilter;

/**
 * 在响应头 {@code X-Served-By} 中写出处理本次请求的实例，如 {@code user-service@user-service-gray-7d9f-x2v7k gray=v2}。
 * <p>
 * 用于本地集群 / 测试环境观察负载均衡与灰度路由；会暴露 Pod 名，生产环境保持关闭（acme.web.served-by-header=false）。
 */
public class ServedByResponseFilter extends OncePerRequestFilter {

    private final String value;

    public ServedByResponseFilter(String application, String instance, String grayTag) {
        this.value = application + "@" + instance + (grayTag == null || grayTag.isBlank() ? "" : " gray=" + grayTag);
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        response.setHeader(AcmeHeaders.SERVED_BY, value);
        chain.doFilter(request, response);
    }
}
