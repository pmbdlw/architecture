package com.acme.framework.core.result;

import java.util.List;
import java.util.function.Function;

/**
 * 分页结果。页码从 1 开始。
 */
public record PageResult<T>(long total, long pageNo, long pageSize, List<T> records) {

    public <R> PageResult<R> map(Function<? super T, ? extends R> mapper) {
        return new PageResult<>(total, pageNo, pageSize, records.stream().<R>map(mapper).toList());
    }
}
