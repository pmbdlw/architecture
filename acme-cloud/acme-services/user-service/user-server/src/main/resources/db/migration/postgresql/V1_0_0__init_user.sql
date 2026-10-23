-- 命名：V<版本>__<描述>.sql；已发布的脚本禁止修改，只能追加新版本
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
    version    INT          NOT NULL DEFAULT 0,
    CONSTRAINT uk_user_account_username UNIQUE (username)
);

COMMENT ON TABLE user_account IS '用户账号';
COMMENT ON COLUMN user_account.id IS '主键（雪花算法）';
COMMENT ON COLUMN user_account.status IS '状态：ACTIVE/DISABLED';
COMMENT ON COLUMN user_account.deleted IS '逻辑删除：0 否 1 是';
COMMENT ON COLUMN user_account.version IS '乐观锁版本号';
