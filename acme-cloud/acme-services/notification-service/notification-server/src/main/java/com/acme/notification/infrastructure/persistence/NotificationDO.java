package com.acme.notification.infrastructure.persistence;

import com.acme.framework.mybatis.BaseDO;
import com.baomidou.mybatisplus.annotation.TableName;

import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@TableName("notification")
public class NotificationDO extends BaseDO {

    private Long userId;
    private String channel;
    private String title;
    private String content;
    private String bizKey;
}
