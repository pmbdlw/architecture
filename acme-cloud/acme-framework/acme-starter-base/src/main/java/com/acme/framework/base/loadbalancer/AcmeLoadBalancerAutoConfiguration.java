package com.acme.framework.base.loadbalancer;

import org.springframework.boot.autoconfigure.AutoConfiguration;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.cloud.loadbalancer.annotation.LoadBalancerClients;

/**
 * 为所有下游服务启用灰度负载均衡。可通过 {@code acme.loadbalancer.gray.enabled=false} 关闭。
 */
@AutoConfiguration
@ConditionalOnProperty(prefix = "acme.loadbalancer.gray", name = "enabled", havingValue = "true", matchIfMissing = true)
@LoadBalancerClients(defaultConfiguration = GrayLoadBalancerClientConfiguration.class)
public class AcmeLoadBalancerAutoConfiguration {
}
