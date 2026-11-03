-- +goose Up
-- 仅用于测试（与 mysql / postgres 版本保持结构一致）
CREATE TABLE user_account
(
    id         BIGINT       NOT NULL PRIMARY KEY,
    username   VARCHAR(64)  NOT NULL UNIQUE,
    mobile     VARCHAR(20),
    email      VARCHAR(128),
    status     VARCHAR(16)  NOT NULL,
    created_at DATETIME     NOT NULL,
    updated_at DATETIME     NOT NULL,
    deleted    SMALLINT     NOT NULL DEFAULT 0,
    version    BIGINT       NOT NULL DEFAULT 1
);

-- +goose Down
DROP TABLE user_account;
