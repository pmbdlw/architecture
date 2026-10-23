package com.acme.user.api;

/**
 * 服务间传输的用户信息（不含敏感字段）。
 *
 * @param status ACTIVE / DISABLED
 */
public record UserDTO(Long id, String username, String status) {

    public boolean isActive() {
        return "ACTIVE".equals(status);
    }
}
