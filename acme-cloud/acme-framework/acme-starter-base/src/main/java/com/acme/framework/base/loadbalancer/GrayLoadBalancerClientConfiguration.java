package com.acme.framework.base.loadbalancer;

import org.springframework.cloud.client.discovery.ReactiveDiscoveryClient;
import org.springframework.cloud.loadbalancer.core.ServiceInstanceListSupplier;
import org.springframework.cloud.loadbalancer.core.ServiceInstanceListSupplierBuilder;
import org.springframework.context.ConfigurableApplicationContext;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * LoadBalancer 子上下文配置：服务发现 → 缓存 → 灰度筛选（网关与 Feign 共用）。
 * 注意：该类不能被业务服务的组件扫描扫到，只能通过 {@code @LoadBalancerClients} 引用。
 * <ul>
 *   <li>不能用 {@code @ConditionalOnWebApplication} 区分网关与普通服务：LoadBalancer 子上下文不是 Web 上下文，条件永远不成立，
 *       会悄悄退回默认的实例列表（灰度失效）。这里按父上下文中是否有响应式服务发现来选择；</li>
 *   <li>灰度筛选必须包在缓存外层：缓存只实现了不带请求参数的 {@code get()}，放在里层会丢失请求中的灰度标签。</li>
 * </ul>
 */
@Configuration(proxyBeanMethods = false)
public class GrayLoadBalancerClientConfiguration {

    @Bean
    public ServiceInstanceListSupplier grayServiceInstanceListSupplier(ConfigurableApplicationContext context) {
        ServiceInstanceListSupplierBuilder builder = ServiceInstanceListSupplier.builder();
        builder = context.getBeanProvider(ReactiveDiscoveryClient.class).getIfAvailable() != null
                ? builder.withDiscoveryClient()
                : builder.withBlockingDiscoveryClient();
        return builder.withCaching()
                .with((ctx, delegate) -> new GrayServiceInstanceListSupplier(delegate))
                .build(context);
    }
}
