CREATE TABLE notification
(
    id         BIGINT        NOT NULL PRIMARY KEY,
    user_id    BIGINT        NOT NULL,
    channel    VARCHAR(16)   NOT NULL,
    title      VARCHAR(128)  NOT NULL,
    content    VARCHAR(1024) NOT NULL,
    biz_key    VARCHAR(64),
    created_at TIMESTAMP(3)  NOT NULL,
    updated_at TIMESTAMP(3)  NOT NULL,
    deleted    SMALLINT      NOT NULL DEFAULT 0,
    version    INT           NOT NULL DEFAULT 0
);
CREATE INDEX idx_notification_user_created ON notification (user_id, created_at);

COMMENT ON TABLE notification IS '通知';
