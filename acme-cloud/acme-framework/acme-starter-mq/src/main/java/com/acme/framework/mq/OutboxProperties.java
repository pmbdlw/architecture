package com.acme.framework.mq;

import java.time.Duration;

import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.boot.context.properties.bind.DefaultValue;

/**
 * @param relayEnabled 是否启动定时投递（测试中关闭后可手动调用 {@link OutboxRelay#relay()}）
 * @param pollInterval 扫描间隔
 * @param batchSize  每批投递条数
 * @param maxRetries 超过该重试次数的事件不再自动投递，需人工介入（有告警）
 * @param lockClause 行锁子句。MySQL 8 / PostgreSQL 使用 SKIP LOCKED；不支持的数据库（如测试用 H2）可改为 FOR UPDATE
 */
@ConfigurationProperties(prefix = "acme.mq.outbox")
public record OutboxProperties(
        @DefaultValue("true") boolean relayEnabled,
        @DefaultValue("1s") Duration pollInterval,
        @DefaultValue("100") int batchSize,
        @DefaultValue("10") int maxRetries,
        @DefaultValue("FOR UPDATE SKIP LOCKED") String lockClause) {
}
