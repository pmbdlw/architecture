package com.acme.framework.core.constant;

import java.util.List;

/**
 * 平台约定的 HTTP 头。网关负责生成/校验，服务间调用由 Feign 拦截器透传。
 */
public final class AcmeHeaders {

    /** 链路 ID，回写在响应头中，与日志中的 traceId 一致。 */
    public static final String TRACE_ID = "X-Trace-Id";

    /** 灰度标签。网关按规则打标，LoadBalancer 据此优先选择同标签实例。 */
    public static final String GRAY_TAG = "X-Gray-Tag";

    /** 客户端请求幂等键（写接口可选）。 */
    public static final String IDEMPOTENCY_KEY = "Idempotency-Key";

    /** 服务间调用需要原样透传的请求头。 */
    public static final List<String> RELAY_HEADERS = List.of("Authorization", GRAY_TAG, "Accept-Language");

    /** 服务实例元数据中的灰度标签 key（Nacos metadata / K8s Pod label 同名）。 */
    public static final String GRAY_TAG_METADATA = "gray-tag";

    private AcmeHeaders() {
    }
}
