package com.acme.framework.mq;

import java.time.Clock;

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

    @Bean
    public OutboxPublisher outboxPublisher(JdbcTemplate jdbcTemplate, JsonMapper jsonMapper,
            @Value("${spring.application.name}") String applicationName, Clock clock) {
        return new OutboxPublisher(jdbcTemplate, jsonMapper, applicationName, clock);
    }

    @Bean
    public IdempotentConsumer idempotentConsumer(JdbcTemplate jdbcTemplate, TransactionTemplate transactionTemplate,
            @Value("${spring.application.name}") String applicationName, Clock clock) {
        return new IdempotentConsumer(jdbcTemplate, transactionTemplate, applicationName, clock);
    }

    @Bean
    public OutboxRelay outboxRelay(JdbcTemplate jdbcTemplate, TransactionTemplate transactionTemplate,
            StreamBridge streamBridge, OutboxProperties properties, Clock clock) {
        return new OutboxRelay(jdbcTemplate, transactionTemplate, streamBridge, properties, clock);
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
