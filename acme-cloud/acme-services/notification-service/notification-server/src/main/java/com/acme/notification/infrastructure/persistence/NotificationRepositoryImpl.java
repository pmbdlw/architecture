package com.acme.notification.infrastructure.persistence;

import java.time.ZoneId;

import org.springframework.stereotype.Repository;

import com.acme.framework.core.result.PageResult;
import com.acme.framework.mybatis.Pages;
import com.acme.notification.domain.model.Notification;
import com.acme.notification.domain.repository.NotificationRepository;
import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.core.toolkit.IdWorker;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;

@Repository
public class NotificationRepositoryImpl implements NotificationRepository {

    private final NotificationMapper notificationMapper;

    public NotificationRepositoryImpl(NotificationMapper notificationMapper) {
        this.notificationMapper = notificationMapper;
    }

    @Override
    public Long nextId() {
        return IdWorker.getId();
    }

    @Override
    public void add(Notification notification) {
        NotificationDO data = new NotificationDO();
        data.setId(notification.id());
        data.setUserId(notification.userId());
        data.setChannel(notification.channel());
        data.setTitle(notification.title());
        data.setContent(notification.content());
        data.setBizKey(notification.bizKey());
        notificationMapper.insert(data);
    }

    @Override
    public PageResult<Notification> pageByUser(Long userId, long pageNo, long pageSize) {
        Page<NotificationDO> page = notificationMapper.selectPage(Page.of(pageNo, pageSize),
                new LambdaQueryWrapper<NotificationDO>()
                        .eq(NotificationDO::getUserId, userId)
                        .orderByDesc(NotificationDO::getCreatedAt));
        return Pages.of(page).map(NotificationRepositoryImpl::toDomain);
    }

    private static Notification toDomain(NotificationDO data) {
        return new Notification(data.getId(), data.getUserId(), data.getChannel(), data.getTitle(),
                data.getContent(), data.getBizKey(), data.getCreatedAt().atZone(ZoneId.systemDefault()).toInstant());
    }
}
