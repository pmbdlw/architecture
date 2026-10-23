package com.acme.notification.domain.repository;

import com.acme.framework.core.result.PageResult;
import com.acme.notification.domain.model.Notification;

public interface NotificationRepository {

    Long nextId();

    void add(Notification notification);

    PageResult<Notification> pageByUser(Long userId, long pageNo, long pageSize);
}
