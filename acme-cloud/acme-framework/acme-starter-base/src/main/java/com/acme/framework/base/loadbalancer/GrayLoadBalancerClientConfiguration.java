package com.acme.framework.base.loadbalancer;

import org.springframework.boot.autoconfigure.condition.ConditionalOnWebApplication;
import org.springframework.cloud.loadbalancer.core.ServiceInstanceListSupplier;
import org.springframework.context.ConfigurableApplicationContext;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * LoadBalancer 子上下文配置：服务发现 → 灰度筛选 → 缓存。
 * 注意：该类不能被业务服务的组件扫描扫到，只能通过 {@code @LoadBalancerClients} 引用。
 */
@Configuration(proxyBeanMethods = false)
public class GrayLoadBalancerClientConfiguration {

    @Bean
    @ConditionalOnWebApplication(type = ConditionalOnWebApplication.Type.SERVLET)
    public ServiceInstanceListSupplier blockingGrayServiceInstanceListSupplier(
            ConfigurableApplicationContext context) {
        return ServiceInstanceListSupplier.builder()
                .withBlockingDiscoveryClient()
                .with((ctx, delegate) -> new GrayServiceInstanceListSupplier(delegate))
                .withCaching()
                .build(context);
    }

    @Bean
    @ConditionalOnWebApplication(type = ConditionalOnWebApplication.Type.REACTIVE)
    public ServiceInstanceListSupplier reactiveGrayServiceInstanceListSupplier(
            ConfigurableApplicationContext context) {
        return ServiceInstanceListSupplier.builder()
                .withDiscoveryClient()
                .with((ctx, delegate) -> new GrayServiceInstanceListSupplier(delegate))
                .withCaching()
                .build(context);
    }
}
