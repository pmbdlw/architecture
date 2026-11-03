-- +goose Up
-- 事务发件箱（pkg/mq 约定的表结构，所有发布事件的服务都需要）
CREATE TABLE outbox_event
(
    event_id     VARCHAR(36)  NOT NULL COMMENT '事件 ID（UUID），即消息头 acme_event_id',
    topic        VARCHAR(128) NOT NULL COMMENT 'Kafka topic / RabbitMQ exchange',
    event_type   VARCHAR(128) NOT NULL COMMENT '事件类型，如 order.created',
    aggregate_id VARCHAR(64)  NOT NULL COMMENT '聚合根 ID / 分区键',
    payload      TEXT         NOT NULL COMMENT '事件 JSON（protojson）',
    source       VARCHAR(64)  NOT NULL COMMENT '来源服务',
    status       TINYINT      NOT NULL DEFAULT 0 COMMENT '0 待发送 1 已发送',
    retry_count  INT          NOT NULL DEFAULT 0,
    created_at   DATETIME(3)  NOT NULL,
    sent_at      DATETIME(3)  NULL,
    PRIMARY KEY (event_id),
    KEY idx_outbox_event_status_created (status, created_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='事务发件箱';

-- +goose Down
DROP TABLE outbox_event;
