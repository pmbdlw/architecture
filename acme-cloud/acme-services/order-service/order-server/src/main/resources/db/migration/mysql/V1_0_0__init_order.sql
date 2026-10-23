CREATE TABLE order_info
(
    id           BIGINT         NOT NULL COMMENT '主键（雪花算法）',
    order_no     VARCHAR(32)    NOT NULL COMMENT '订单号',
    user_id      BIGINT         NOT NULL COMMENT '买家用户 ID（user-service，非外键）',
    product_code VARCHAR(64)    NOT NULL COMMENT '商品编码',
    quantity     INT            NOT NULL COMMENT '数量',
    amount       DECIMAL(18, 2) NOT NULL COMMENT '金额（元）',
    status       VARCHAR(16)    NOT NULL COMMENT 'CREATED/PAID/CANCELLED',
    created_at   DATETIME(3)    NOT NULL,
    updated_at   DATETIME(3)    NOT NULL,
    deleted      TINYINT        NOT NULL DEFAULT 0,
    version      INT            NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uk_order_info_order_no (order_no),
    KEY idx_order_info_user_created (user_id, created_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='订单';
