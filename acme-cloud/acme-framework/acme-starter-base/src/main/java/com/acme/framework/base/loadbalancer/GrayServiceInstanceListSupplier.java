package com.acme.framework.base.loadbalancer;

import java.util.List;

import org.springframework.cloud.client.ServiceInstance;
import org.springframework.cloud.client.loadbalancer.Request;
import org.springframework.cloud.client.loadbalancer.RequestDataContext;
import org.springframework.cloud.loadbalancer.core.DelegatingServiceInstanceListSupplier;
import org.springframework.cloud.loadbalancer.core.ServiceInstanceListSupplier;
import org.springframework.util.StringUtils;

import com.acme.framework.core.constant.AcmeHeaders;

import reactor.core.publisher.Flux;

/**
 * 灰度实例筛选（网关与 Feign 共用）。
 * <ul>
 *   <li>请求带 {@code X-Gray-Tag: v2}：优先选择 metadata {@code gray-tag=v2} 的实例，没有则回落到稳定实例；</li>
 *   <li>请求不带灰度标签：只选择未打标签的稳定实例，灰度实例不接收普通流量。</li>
 * </ul>
 * 灰度标签由 Deployment 环境变量 {@code GRAY_TAG} 写入 Nacos 实例元数据。
 */
public class GrayServiceInstanceListSupplier extends DelegatingServiceInstanceListSupplier {

    public GrayServiceInstanceListSupplier(ServiceInstanceListSupplier delegate) {
        super(delegate);
    }

    @Override
    public Flux<List<ServiceInstance>> get() {
        return delegate.get().map(instances -> select(instances, null));
    }

    @Override
    public Flux<List<ServiceInstance>> get(Request request) {
        String grayTag = grayTagOf(request);
        return delegate.get(request).map(instances -> select(instances, grayTag));
    }

    static List<ServiceInstance> select(List<ServiceInstance> instances, String grayTag) {
        List<ServiceInstance> stable = instances.stream().filter(i -> !StringUtils.hasText(tagOf(i))).toList();
        if (StringUtils.hasText(grayTag)) {
            List<ServiceInstance> gray = instances.stream().filter(i -> grayTag.equals(tagOf(i))).toList();
            if (!gray.isEmpty()) {
                return gray;
            }
        }
        // 全部实例都打了标签（例如整体切换中）时不做过滤，保证可用性优先
        return stable.isEmpty() ? instances : stable;
    }

    private static String tagOf(ServiceInstance instance) {
        return instance.getMetadata().get(AcmeHeaders.GRAY_TAG_METADATA);
    }

    private static String grayTagOf(Request request) {
        if (request != null && request.getContext() instanceof RequestDataContext context
                && context.getClientRequest() != null) {
            return context.getClientRequest().getHeaders().getFirst(AcmeHeaders.GRAY_TAG);
        }
        return null;
    }
}
