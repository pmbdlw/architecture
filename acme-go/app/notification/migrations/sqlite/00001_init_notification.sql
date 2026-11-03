-- +goose Up
-- 仅用于测试（与 mysql / postgres 版本保持结构一致）
CREATE TABLE notification
(
    id         BIGINT        NOT NULL PRIMARY KEY,
    user_id    BIGINT        NOT NULL,
    channel    VARCHAR(16)   NOT NULL,
    title      VARCHAR(128)  NOT NULL,
    content    VARCHAR(1024) NOT NULL,
    biz_key    VARCHAR(64),
    created_at DATETIME      NOT NULL,
    updated_at DATETIME      NOT NULL,
    deleted    SMALLINT      NOT NULL DEFAULT 0,
    version    BIGINT        NOT NULL DEFAULT 1
);

-- +goose Down
DROP TABLE notification;
