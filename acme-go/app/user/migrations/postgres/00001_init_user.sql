-- +goose Up
CREATE TABLE user_account
(
    id         BIGINT       NOT NULL PRIMARY KEY,
    username   VARCHAR(64)  NOT NULL,
    mobile     VARCHAR(20),
    email      VARCHAR(128),
    status     VARCHAR(16)  NOT NULL,
    created_at TIMESTAMP(3) NOT NULL,
    updated_at TIMESTAMP(3) NOT NULL,
    deleted    SMALLINT     NOT NULL DEFAULT 0,
    version    BIGINT       NOT NULL DEFAULT 1,
    CONSTRAINT uk_user_account_username UNIQUE (username)
);
COMMENT ON TABLE user_account IS '用户账号';
COMMENT ON COLUMN user_account.status IS '状态：ACTIVE/DISABLED';

-- +goose Down
DROP TABLE user_account;
