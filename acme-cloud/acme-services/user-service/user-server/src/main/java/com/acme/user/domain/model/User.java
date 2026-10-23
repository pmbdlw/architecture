package com.acme.user.domain.model;

import java.util.Objects;

import com.acme.framework.core.exception.BizException;
import com.acme.user.domain.UserErrorCode;

/**
 * 用户聚合根。领域对象只包含业务状态与规则，不带任何持久化/Web 注解。
 */
public class User {

    private final Long id;
    private final String username;
    private final String mobile;
    private final String email;
    private UserStatus status;

    public User(Long id, String username, String mobile, String email, UserStatus status) {
        this.id = id;
        this.username = Objects.requireNonNull(username);
        this.mobile = mobile;
        this.email = email;
        this.status = Objects.requireNonNull(status);
    }

    public static User register(Long id, String username, String mobile, String email) {
        return new User(id, username, mobile, email, UserStatus.ACTIVE);
    }

    public void disable() {
        if (status == UserStatus.DISABLED) {
            throw new BizException(UserErrorCode.USER_ALREADY_DISABLED);
        }
        status = UserStatus.DISABLED;
    }

    public Long getId() {
        return id;
    }

    public String getUsername() {
        return username;
    }

    public String getMobile() {
        return mobile;
    }

    public String getEmail() {
        return email;
    }

    public UserStatus getStatus() {
        return status;
    }
}
