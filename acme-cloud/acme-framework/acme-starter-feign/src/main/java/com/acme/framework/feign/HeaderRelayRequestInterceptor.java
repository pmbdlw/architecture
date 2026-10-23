package com.acme.framework.feign;

import org.springframework.web.context.request.RequestAttributes;
import org.springframework.web.context.request.RequestContextHolder;
import org.springframework.web.context.request.ServletRequestAttributes;

import com.acme.framework.core.constant.AcmeHeaders;

import feign.RequestInterceptor;
import feign.RequestTemplate;

/**
 * 服务间调用时透传身份令牌、灰度标签等请求头（见 {@link AcmeHeaders#RELAY_HEADERS}）。
 * <p>
 * 链路追踪头（traceparent）由 Micrometer 自动注入，这里不处理。
 * 只能在发起请求的线程中读取到上游请求；异步线程调用 Feign 时需自行传递上下文。
 */
public class HeaderRelayRequestInterceptor implements RequestInterceptor {

    @Override
    public void apply(RequestTemplate template) {
        RequestAttributes attributes = RequestContextHolder.getRequestAttributes();
        if (!(attributes instanceof ServletRequestAttributes servletAttributes)) {
            return;
        }
        var request = servletAttributes.getRequest();
        for (String header : AcmeHeaders.RELAY_HEADERS) {
            String value = request.getHeader(header);
            if (value != null && !template.headers().containsKey(header)) {
                template.header(header, value);
            }
        }
    }
}
