package com.acme.framework.base.shutdown;

import java.time.Duration;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.AutoConfiguration;
import org.springframework.boot.autoconfigure.condition.ConditionalOnBean;
import org.springframework.cloud.client.serviceregistry.AbstractAutoServiceRegistration;
import org.springframework.context.annotation.Bean;

/**
 * 优雅下线：先注销、再等待、后停机。等待时间 {@code acme.shutdown.drain-delay}（默认 5s），
 * 需大于 Nacos 推送延迟 + LoadBalancer 实例缓存 TTL，并且 preStop + drain-delay + 优雅停机超时 < terminationGracePeriodSeconds。
 */
@AutoConfiguration(afterName = "com.alibaba.cloud.nacos.registry.NacosServiceRegistryAutoConfiguration")
public class AcmeShutdownAutoConfiguration {

    @Bean
    @ConditionalOnBean(AbstractAutoServiceRegistration.class)
    public GracefulDeregistration acmeGracefulDeregistration(AbstractAutoServiceRegistration<?> registration,
            @Value("${acme.shutdown.drain-delay:5s}") Duration drainDelay) {
        return new GracefulDeregistration(registration, drainDelay);
    }
}
