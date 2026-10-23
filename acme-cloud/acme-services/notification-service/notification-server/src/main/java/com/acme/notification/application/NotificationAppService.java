package com.acme.notification.application;

import java.time.Clock;
import java.time.Instant;

import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.acme.framework.core.result.PageResult;
import com.acme.notification.domain.gateway.NotificationSender;
import com.acme.notification.domain.model.Notification;
import com.acme.notification.domain.repository.NotificationRepository;
import com.acme.order.api.event.OrderCreatedEvent;

@Service
public class NotificationAppService {

    private final NotificationRepository notificationRepository;
    private final NotificationSender notificationSender;
    private final Clock clock;

    public NotificationAppService(NotificationRepository notificationRepository,
            NotificationSender notificationSender, Clock clock) {
        this.notificationRepository = notificationRepository;
        this.notificationSender = notificationSender;
        this.clock = clock;
    }

    /**
     * 处理“订单已创建”事件。调用方（消费者）已保证同一事件只会成功执行一次。
     */
    public void onOrderCreated(OrderCreatedEvent event) {
        Notification notification = new Notification(notificationRepository.nextId(), event.userId(), "SITE",
                "下单成功", "您的订单 " + event.orderNo() + " 已创建，金额 " + event.amount() + " 元",
                event.orderNo(), Instant.now(clock));
        notificationRepository.add(notification);
        notificationSender.send(notification);
    }

    @Transactional(readOnly = true)
    public PageResult<Notification> pageMine(Long userId, long pageNo, long pageSize) {
        return notificationRepository.pageByUser(userId, pageNo, pageSize);
    }
}
