CREATE TABLE notification
(
    id         BIGINT       NOT NULL COMMENT '主键（雪花算法）',
    user_id    BIGINT       NOT NULL COMMENT '接收人',
    channel    VARCHAR(16)  NOT NULL COMMENT 'SITE/SMS/EMAIL',
    title      VARCHAR(128) NOT NULL,
    content    VARCHAR(1024) NOT NULL,
    biz_key    VARCHAR(64)  NULL COMMENT '业务键，如订单号',
    created_at DATETIME(3)  NOT NULL,
    updated_at DATETIME(3)  NOT NULL,
    deleted    TINYINT      NOT NULL DEFAULT 0,
    version    INT          NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_notification_user_created (user_id, created_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='通知';
