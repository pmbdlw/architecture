-- 消费幂等表（acme-starter-mq 约定的表结构，所有消费事件的服务都需要），可按 consumed_at 定期归档
CREATE TABLE consumed_event
(
    consumer    VARCHAR(64) NOT NULL COMMENT '消费者（服务名）',
    event_id    VARCHAR(36) NOT NULL COMMENT '事件 ID（acme_event_id）',
    consumed_at DATETIME(3) NOT NULL,
    PRIMARY KEY (consumer, event_id)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='已消费事件';
