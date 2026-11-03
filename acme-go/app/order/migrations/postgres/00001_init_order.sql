-- +goose Up
CREATE TABLE order_info
(
    id           BIGINT         NOT NULL PRIMARY KEY,
    order_no     VARCHAR(32)    NOT NULL,
    user_id      BIGINT         NOT NULL,
    product_code VARCHAR(64)    NOT NULL,
    quantity     INT            NOT NULL,
    amount       NUMERIC(18, 2) NOT NULL,
    status       VARCHAR(16)    NOT NULL,
    created_at   TIMESTAMP(3)   NOT NULL,
    updated_at   TIMESTAMP(3)   NOT NULL,
    deleted      SMALLINT       NOT NULL DEFAULT 0,
    version      BIGINT         NOT NULL DEFAULT 1,
    CONSTRAINT uk_order_info_order_no UNIQUE (order_no)
);
CREATE INDEX idx_order_info_user_created ON order_info (user_id, created_at);
COMMENT ON TABLE order_info IS '订单';

-- +goose Down
DROP TABLE order_info;
