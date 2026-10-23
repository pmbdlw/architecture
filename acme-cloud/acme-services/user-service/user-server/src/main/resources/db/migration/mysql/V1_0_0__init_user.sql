-- 命名：V<版本>__<描述>.sql；已发布的脚本禁止修改，只能追加新版本
CREATE TABLE user_account
(
    id         BIGINT       NOT NULL COMMENT '主键（雪花算法）',
    username   VARCHAR(64)  NOT NULL COMMENT '用户名',
    mobile     VARCHAR(20)  NULL COMMENT '手机号',
    email      VARCHAR(128) NULL COMMENT '邮箱',
    status     VARCHAR(16)  NOT NULL COMMENT '状态：ACTIVE/DISABLED',
    created_at DATETIME(3)  NOT NULL COMMENT '创建时间',
    updated_at DATETIME(3)  NOT NULL COMMENT '更新时间',
    deleted    TINYINT      NOT NULL DEFAULT 0 COMMENT '逻辑删除：0 否 1 是',
    version    INT          NOT NULL DEFAULT 0 COMMENT '乐观锁版本号',
    PRIMARY KEY (id),
    UNIQUE KEY uk_user_account_username (username)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='用户账号';
