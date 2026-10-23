package com.acme.notification.infrastructure.sender;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import com.acme.notification.domain.gateway.NotificationSender;
import com.acme.notification.domain.model.Notification;

/**
 * 占位实现：只记录日志。接入真实渠道时新增实现类并按渠道路由。
 */
@Component
public class LogNotificationSender implements NotificationSender {

    private static final Logger log = LoggerFactory.getLogger(LogNotificationSender.class);

    @Override
    public void send(Notification notification) {
        log.info("发送通知 channel={} userId={} bizKey={}", notification.channel(), notification.userId(),
                notification.bizKey());
    }
}
