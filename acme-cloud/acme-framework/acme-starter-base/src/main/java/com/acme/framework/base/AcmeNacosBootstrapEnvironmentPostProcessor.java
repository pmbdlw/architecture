package com.acme.framework.base;

import java.util.LinkedHashMap;
import java.util.Map;
import org.springframework.boot.EnvironmentPostProcessor;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.context.config.ConfigDataEnvironmentPostProcessor;
import org.springframework.core.Ordered;
import org.springframework.core.env.ConfigurableEnvironment;
import org.springframework.core.env.EnumerablePropertySource;
import org.springframework.core.env.MapPropertySource;
import org.springframework.core.env.MutablePropertySources;
import org.springframework.core.env.PropertySource;

/**
 * 在 ConfigData 处理之前，把平台默认配置中的 Nacos 连接参数（{@code spring.cloud.nacos.*}）放进 defaultProperties。
 * <p>
 * {@code spring.config.import: optional:nacos:...} 在 ConfigData 阶段就要连接 Nacos，而完整的平台默认配置
 * （{@link AcmeDefaultsEnvironmentPostProcessor}）要等 ConfigData 之后才能加入，否则会压过 application.yml。
 * 缺少这一步时，配置中心客户端拿不到 NACOS_SERVER_ADDR，会连到 127.0.0.1:8848 —— 本机开发看不出问题，
 * 一到集群里就读不到 Nacos 配置。defaultProperties 的优先级最低，不影响任何覆盖关系。
 */
public class AcmeNacosBootstrapEnvironmentPostProcessor implements EnvironmentPostProcessor, Ordered {

    static final String PREFIX = "spring.cloud.nacos.";

    private static final String DEFAULT_PROPERTIES = "defaultProperties";

    @Override
    public void postProcessEnvironment(ConfigurableEnvironment environment, SpringApplication application) {
        Map<String, Object> nacos = new LinkedHashMap<>();
        for (PropertySource<?> source : AcmeDefaultsEnvironmentPostProcessor.loadAll()) {
            if (source instanceof EnumerablePropertySource<?> enumerable) {
                for (String name : enumerable.getPropertyNames()) {
                    if (name.startsWith(PREFIX)) {
                        nacos.putIfAbsent(name, enumerable.getProperty(name));
                    }
                }
            }
        }
        if (nacos.isEmpty()) {
            return;
        }
        MutablePropertySources sources = environment.getPropertySources();
        PropertySource<?> existing = sources.get(DEFAULT_PROPERTIES);
        if (existing instanceof MapPropertySource map) {
            nacos.forEach((k, v) -> map.getSource().putIfAbsent(k, v));
        } else if (existing == null) {
            sources.addLast(new MapPropertySource(DEFAULT_PROPERTIES, nacos));
        }
    }

    @Override
    public int getOrder() {
        return ConfigDataEnvironmentPostProcessor.ORDER - 1;
    }
}
