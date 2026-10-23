package com.acme.framework.web;

import java.io.IOException;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

import org.slf4j.MDC;
import org.springframework.web.filter.OncePerRequestFilter;

import com.acme.framework.core.constant.AcmeHeaders;

/**
 * 把当前链路 ID 写回响应头 {@code X-Trace-Id}。
 * 必须排在 Micrometer 的 ServerHttpObservationFilter 之后（此时 MDC 中已有 traceId）。
 */
public class TraceIdResponseFilter extends OncePerRequestFilter {

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        String traceId = MDC.get("traceId");
        if (traceId != null) {
            response.setHeader(AcmeHeaders.TRACE_ID, traceId);
        }
        chain.doFilter(request, response);
    }
}
