package com.acme.notification.domain.model;

import java.time.Instant;

/**
 * 站内通知。
 *
 * @param channel   发送渠道：SITE / SMS / EMAIL
 * @param bizKey    业务键（如订单号），便于排查
 */
public record Notification(Long id, Long userId, String channel, String title, String content, String bizKey,
        Instant createdAt) {
}
