-- +goose Up
-- 仅用于测试（与 mysql / postgres 版本保持结构一致）
CREATE TABLE order_info
(
    id           BIGINT         NOT NULL PRIMARY KEY,
    order_no     VARCHAR(32)    NOT NULL UNIQUE,
    user_id      BIGINT         NOT NULL,
    product_code VARCHAR(64)    NOT NULL,
    quantity     INT            NOT NULL,
    amount       DECIMAL(18, 2) NOT NULL,
    status       VARCHAR(16)    NOT NULL,
    created_at   DATETIME       NOT NULL,
    updated_at   DATETIME       NOT NULL,
    deleted      SMALLINT       NOT NULL DEFAULT 0,
    version      BIGINT         NOT NULL DEFAULT 1
);

-- +goose Down
DROP TABLE order_info;
