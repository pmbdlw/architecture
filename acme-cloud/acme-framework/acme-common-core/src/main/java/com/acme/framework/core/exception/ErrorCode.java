package com.acme.framework.core.exception;

/**
 * 错误码契约。
 * <p>
 * 编码规则参考《阿里巴巴 Java 开发手册》：5 位字符串，首字母表示错误来源——
 * {@code A} 调用方/用户端错误、{@code B} 当前系统错误、{@code C} 第三方/下游服务错误；
 * 后四位数字中，平台保留 0000-0999，各服务按团队规范分配号段（如 user-service 使用 1000-1999）。
 */
public interface ErrorCode {

    /** 业务错误码，例如 {@code A1001}。 */
    String code();

    /** 默认提示语，可在抛出时带参数格式化。 */
    String message();

    /** 对应的 HTTP 状态码。错误码与 HTTP 状态同时表达语义，网关/监控只看状态码即可分类。 */
    int httpStatus();
}
