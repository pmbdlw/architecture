package com.acme.framework.feign;

import org.springframework.boot.autoconfigure.AutoConfiguration;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;

import feign.codec.ErrorDecoder;
import tools.jackson.databind.json.JsonMapper;

/**
 * Feign 全局默认配置：所有 {@code @FeignClient} 共享。
 * 单个客户端需要特殊配置时，用 {@code spring.cloud.openfeign.client.config.<contextId>} 覆盖。
 */
@AutoConfiguration
public class AcmeFeignAutoConfiguration {

    @Bean
    public HeaderRelayRequestInterceptor headerRelayRequestInterceptor() {
        return new HeaderRelayRequestInterceptor();
    }

    @Bean
    @ConditionalOnMissingBean(ErrorDecoder.class)
    public ErrorDecoder acmeFeignErrorDecoder() {
        return new AcmeFeignErrorDecoder(JsonMapper.builder().build());
    }
}
