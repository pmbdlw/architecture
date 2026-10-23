package com.acme.gateway.config;

import java.util.List;

import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.boot.context.properties.bind.DefaultValue;

/**
 * 灰度规则，放在 Nacos 的 acme-gateway.yaml 中，修改后实时生效。
 *
 * @param enabled           是否开启灰度
 * @param tag               灰度版本标签，与灰度 Deployment 的 GRAY_TAG 环境变量一致，如 v2
 * @param uids              白名单用户（JWT uid）
 * @param percentage        按用户 ID 哈希放量的百分比 0-100
 * @param allowClientHeader 是否信任客户端自带的 X-Gray-Tag（仅 test/staging 打开，便于测试同学指定版本）
 */
@ConfigurationProperties(prefix = "acme.gateway.gray")
public record GrayProperties(
        @DefaultValue("false") boolean enabled,
        String tag,
        @DefaultValue List<String> uids,
        @DefaultValue("0") int percentage,
        @DefaultValue("false") boolean allowClientHeader) {
}
