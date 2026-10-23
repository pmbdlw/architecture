package com.acme.notification.domain.gateway;

import com.acme.notification.domain.model.Notification;

/**
 * 通知发送渠道。实现：自建短信网关 / 阿里云短信 / 邮件推送 / AWS SES / Azure Communication Services 等。
 */
public interface NotificationSender {

    void send(Notification notification);
}
