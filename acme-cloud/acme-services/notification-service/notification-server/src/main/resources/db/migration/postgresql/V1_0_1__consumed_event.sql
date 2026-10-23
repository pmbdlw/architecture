-- 消费幂等表（acme-starter-mq 约定的表结构，所有消费事件的服务都需要），可按 consumed_at 定期归档
CREATE TABLE consumed_event
(
    consumer    VARCHAR(64)  NOT NULL,
    event_id    VARCHAR(36)  NOT NULL,
    consumed_at TIMESTAMP(3) NOT NULL,
    PRIMARY KEY (consumer, event_id)
);

COMMENT ON TABLE consumed_event IS '已消费事件';
