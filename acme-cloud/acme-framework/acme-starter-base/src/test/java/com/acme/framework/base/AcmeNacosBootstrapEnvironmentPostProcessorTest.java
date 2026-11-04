package com.acme.framework.base;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.Map;
import org.junit.jupiter.api.Test;
import org.springframework.core.env.MapPropertySource;
import org.springframework.core.env.StandardEnvironment;

class AcmeNacosBootstrapEnvironmentPostProcessorTest {

    @Test
    void nacosConnectionIsResolvableBeforeConfigDataImport() {
        StandardEnvironment environment = new StandardEnvironment();
        environment.getPropertySources().addFirst(new MapPropertySource("k8s-env",
                Map.of("NACOS_SERVER_ADDR", "nacos.middleware.svc.cluster.local:8848", "NACOS_NAMESPACE", "local")));

        new AcmeNacosBootstrapEnvironmentPostProcessor().postProcessEnvironment(environment, null);

        // spring.config.import 的 Nacos 客户端在此时就能读到集群地址，而不是 127.0.0.1:8848
        assertThat(environment.getProperty("spring.cloud.nacos.server-addr"))
                .isEqualTo("nacos.middleware.svc.cluster.local:8848");
        assertThat(environment.getProperty("spring.cloud.nacos.config.namespace")).isEqualTo("local");
        // 放在 defaultProperties 中，优先级最低
        assertThat(environment.getPropertySources().stream().reduce((a, b) -> b).orElseThrow().getName())
                .isEqualTo("defaultProperties");
        assertThat(environment.getProperty("server.port")).isNull();
    }
}
