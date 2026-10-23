package com.acme.framework.mybatis;

import com.acme.framework.core.result.PageResult;
import com.baomidou.mybatisplus.core.metadata.IPage;

/**
 * MyBatis-Plus 分页对象与平台 {@link PageResult} 之间的转换。
 * PageResult 属于接口层模型，不允许把 IPage 直接返回给调用方。
 */
public final class Pages {

    private Pages() {
    }

    public static <T> PageResult<T> of(IPage<T> page) {
        return new PageResult<>(page.getTotal(), page.getCurrent(), page.getSize(), page.getRecords());
    }
}
