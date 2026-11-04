package com.acme.framework.mq;

import io.micrometer.core.instrument.Gauge;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.binder.MeterBinder;
import io.micrometer.core.instrument.simple.SimpleMeterRegistry;
import io.micrometer.tracing.Tracer;
import io.micrometer.tracing.propagation.Propagator;
import java.time.Clock;

import org.springframework.beans.factory.ObjectProvider;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.AutoConfiguration;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.cloud.stream.function.StreamBridge;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.scheduling.annotation.EnableScheduling;
import org.springframework.scheduling.annotation.SchedulingConfigurer;
import org.springframework.scheduling.config.ScheduledTaskRegistrar;
import org.springframework.transaction.support.TransactionTemplate;

import tools.jackson.databind.json.JsonMapper;

/**
 * 消息规范自动装配。依赖服务库中存在 outbox_event / consumed_event 两张表（见服务的 Flyway 脚本）。
 */
@AutoConfiguration(afterName = "org.springframework.boot.jdbc.autoconfigure.DataSourceTransactionManagerAutoConfiguration")
@EnableConfigurationProperties(OutboxProperties.class)
public class AcmeMqAutoConfiguration {

    @Bean
    @ConditionalOnMissingBean(Clock.class)
    public Clock clock() {
        return Clock.systemDefaultZone();
    }

    /** 有链路追踪（spring-boot-starter-opentelemetry）时透传 traceparent，否则为空实现。 */
    @Bean
    @ConditionalOnMissingBean
    public MessagingTracing messagingTracing(ObjectProvider<Tracer> tracer, ObjectProvider<Propagator> propagator) {
        Tracer t = tracer.getIfAvailable();
        Propagator p = propagator.getIfAvailable();
        return t == null || p == null ? MessagingTracing.noop() : new MessagingTracing(t, p);
    }

    @Bean
    public OutboxPublisher outboxPublisher(JdbcTemplate jdbcTemplate, JsonMapper jsonMapper,
            @Value("${spring.application.name}") String applicationName, Clock clock, MessagingTracing tracing) {
        return new OutboxPublisher(jdbcTemplate, jsonMapper, applicationName, clock, tracing);
    }

    @Bean
    public IdempotentConsumer idempotentConsumer(JdbcTemplate jdbcTemplate, TransactionTemplate transactionTemplate,
            @Value("${spring.application.name}") String applicationName, Clock clock, MessagingTracing tracing,
            ObjectProvider<MeterRegistry> meterRegistry) {
        return new IdempotentConsumer(jdbcTemplate, transactionTemplate, applicationName, clock, tracing,
                meterRegistry.getIfAvailable(SimpleMeterRegistry::new));
    }

    @Bean
    public OutboxRelay outboxRelay(JdbcTemplate jdbcTemplate, TransactionTemplate transactionTemplate,
            StreamBridge streamBridge, OutboxProperties properties, Clock clock, MessagingTracing tracing,
            ObjectProvider<MeterRegistry> meterRegistry) {
        return new OutboxRelay(jdbcTemplate, transactionTemplate, streamBridge, properties, clock, tracing,
                meterRegistry.getIfAvailable(SimpleMeterRegistry::new));
    }

    /** 发件箱积压量 acme_outbox_pending：持续上涨说明 MQ 不可用或投递失败，需要告警。 */
    @Bean
    @ConditionalOnProperty(prefix = "acme.mq.outbox", name = "relay-enabled", havingValue = "true",
            matchIfMissing = true)
    public MeterBinder outboxPendingMetrics(OutboxRelay outboxRelay) {
        return registry -> Gauge.builder("acme.outbox.pending", () -> {
                    try {
                        return outboxRelay.pendingCount();
                    } catch (RuntimeException ex) {
                        return Double.NaN;
                    }
                })
                .description("发件箱中待投递的事件数")
                .register(registry);
    }

    @Configuration(proxyBeanMethods = false)
    @EnableScheduling
    @ConditionalOnProperty(prefix = "acme.mq.outbox", name = "relay-enabled", havingValue = "true",
            matchIfMissing = true)
    static class OutboxRelaySchedulingConfiguration implements SchedulingConfigurer {

        private final OutboxRelay outboxRelay;
        private final OutboxProperties properties;

        OutboxRelaySchedulingConfiguration(OutboxRelay outboxRelay, OutboxProperties properties) {
            this.outboxRelay = outboxRelay;
            this.properties = properties;
        }

        @Override
        public void configureTasks(ScheduledTaskRegistrar registrar) {
            registrar.addFixedDelayTask(outboxRelay::relay, properties.pollInterval());
        }
    }
}
