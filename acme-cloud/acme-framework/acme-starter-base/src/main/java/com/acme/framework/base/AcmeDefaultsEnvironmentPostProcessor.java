package com.acme.framework.base;

import java.io.IOException;
import java.io.UncheckedIOException;
import java.util.Arrays;
import java.util.Comparator;
import java.util.List;

import org.springframework.boot.EnvironmentPostProcessor;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.context.config.ConfigDataEnvironmentPostProcessor;
import org.springframework.boot.env.YamlPropertySourceLoader;
import org.springframework.core.Ordered;
import org.springframework.core.env.ConfigurableEnvironment;
import org.springframework.core.env.MutablePropertySources;
import org.springframework.core.env.PropertySource;
import org.springframework.core.io.Resource;
import org.springframework.core.io.support.PathMatchingResourcePatternResolver;

/**
 * 加载各 starter 自带的平台默认配置 {@code classpath*:META-INF/acme/defaults/*.yml}。
 * <p>
 * 优先级最低（仅高于 Spring Boot 的 defaultProperties）：服务的 application.yml、Nacos 配置、
 * 环境变量都可以覆盖这里的值。这样平台规范（探针、优雅停机、追踪采样等）“默认正确”，
 * 而业务服务的 application.yml 只需要写自己特有的配置。
 */
public class AcmeDefaultsEnvironmentPostProcessor implements EnvironmentPostProcessor, Ordered {

    static final String LOCATION = "classpath*:META-INF/acme/defaults/*.yml";

    private static final String DEFAULT_PROPERTIES = "defaultProperties";

    private final YamlPropertySourceLoader loader = new YamlPropertySourceLoader();

    @Override
    public void postProcessEnvironment(ConfigurableEnvironment environment, SpringApplication application) {
        MutablePropertySources sources = environment.getPropertySources();
        for (Resource resource : resources()) {
            for (PropertySource<?> source : load(resource)) {
                if (sources.contains(source.getName())) {
                    continue;
                }
                if (sources.contains(DEFAULT_PROPERTIES)) {
                    sources.addBefore(DEFAULT_PROPERTIES, source);
                } else {
                    sources.addLast(source);
                }
            }
        }
    }

    private List<Resource> resources() {
        try {
            Resource[] found = new PathMatchingResourcePatternResolver().getResources(LOCATION);
            // 按文件名排序，保证不同类路径顺序下结果稳定
            return Arrays.stream(found)
                    .sorted(Comparator.comparing(r -> String.valueOf(r.getFilename())))
                    .toList();
        } catch (IOException ex) {
            throw new UncheckedIOException("加载平台默认配置失败: " + LOCATION, ex);
        }
    }

    private List<PropertySource<?>> load(Resource resource) {
        try {
            return loader.load("acme-defaults:" + resource.getFilename(), resource);
        } catch (IOException ex) {
            throw new UncheckedIOException("解析平台默认配置失败: " + resource, ex);
        }
    }

    @Override
    public int getOrder() {
        // 必须在 application.yml / Nacos 等 ConfigData 加载之后执行，才能排在它们后面
        return ConfigDataEnvironmentPostProcessor.ORDER + 1;
    }
}
