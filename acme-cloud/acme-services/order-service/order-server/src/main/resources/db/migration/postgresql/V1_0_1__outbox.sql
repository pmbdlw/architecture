-- 事务发件箱（acme-starter-mq 约定的表结构，所有发布事件的服务都需要）
CREATE TABLE outbox_event
(
    event_id     VARCHAR(36)  NOT NULL PRIMARY KEY,
    binding_name VARCHAR(128) NOT NULL,
    event_type   VARCHAR(128) NOT NULL,
    aggregate_id VARCHAR(64)  NOT NULL,
    payload      TEXT         NOT NULL,
    source       VARCHAR(64)  NOT NULL,
    status       SMALLINT     NOT NULL DEFAULT 0,
    retry_count  INT          NOT NULL DEFAULT 0,
    created_at   TIMESTAMP(3) NOT NULL,
    sent_at      TIMESTAMP(3)
);
CREATE INDEX idx_outbox_event_status_created ON outbox_event (status, created_at);

COMMENT ON TABLE outbox_event IS '事务发件箱';
