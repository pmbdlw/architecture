package com.acme.framework.base.shutdown;

import java.time.Duration;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.cloud.client.serviceregistry.AbstractAutoServiceRegistration;
import org.springframework.context.ApplicationListener;
import org.springframework.context.event.ContextClosedEvent;

/**
 * 停机时先从注册中心注销，再等待 drain-delay，最后才进入 Web 容器的优雅停机。
 * <p>
 * Spring Cloud 默认在销毁 Bean 时才注销实例——此时 Web 容器已经停止接收新请求，
 * 而调用方（网关、Feign）手里还有缓存的实例列表，滚动发布期间就会出现少量失败请求。
 * 提前注销并等待调用方刷新实例列表（Nacos 推送 + LoadBalancer 缓存 TTL），发布即可做到零失败。
 */
public class GracefulDeregistration implements ApplicationListener<ContextClosedEvent> {

    private static final Logger log = LoggerFactory.getLogger(GracefulDeregistration.class);

    private final AbstractAutoServiceRegistration<?> registration;
    private final Duration drainDelay;

    public GracefulDeregistration(AbstractAutoServiceRegistration<?> registration, Duration drainDelay) {
        this.registration = registration;
        this.drainDelay = drainDelay;
    }

    @Override
    public void onApplicationEvent(ContextClosedEvent event) {
        if (event.getApplicationContext().getParent() != null || !registration.isRunning()) {
            return; // 只处理根上下文（忽略 Feign / LoadBalancer 子上下文）
        }
        registration.stop();
        log.info("已从注册中心注销，{} ms 后开始优雅停机", drainDelay.toMillis());
        try {
            Thread.sleep(drainDelay);
        } catch (InterruptedException ex) {
            Thread.currentThread().interrupt();
        }
    }
}
