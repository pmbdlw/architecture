package com.acme.framework.core.exception;

/**
 * 下游服务返回 5xx 或不可用时抛出。
 * <p>
 * 与 {@link BizException} 区分开：熔断器只统计本异常（及网络异常），
 * 不统计下游返回的 4xx 业务失败，避免“用户不存在”之类的正常业务结果把熔断器打开。
 */
public class RemoteServiceException extends RuntimeException {

    private final String service;
    private final int status;

    public RemoteServiceException(String service, int status, String message) {
        super("调用 " + service + " 失败(" + status + "): " + message);
        this.service = service;
        this.status = status;
    }

    public String getService() {
        return service;
    }

    public int getStatus() {
        return status;
    }
}
