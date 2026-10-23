package com.acme.notification.infrastructure.persistence;

import org.apache.ibatis.annotations.Mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;

@Mapper
public interface NotificationMapper extends BaseMapper<NotificationDO> {
}
