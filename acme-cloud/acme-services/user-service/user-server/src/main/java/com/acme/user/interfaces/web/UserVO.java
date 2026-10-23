package com.acme.user.interfaces.web;

import com.acme.user.domain.model.User;

/**
 * 对外视图对象。手机号脱敏输出（日志、接口均不得输出完整个人敏感信息）。
 */
public record UserVO(Long id, String username, String mobile, String email, String status) {

    static UserVO from(User user) {
        return new UserVO(user.getId(), user.getUsername(), mask(user.getMobile()), user.getEmail(),
                user.getStatus().name());
    }

    private static String mask(String mobile) {
        if (mobile == null || mobile.length() < 7) {
            return mobile;
        }
        return mobile.substring(0, 3) + "****" + mobile.substring(mobile.length() - 4);
    }
}
