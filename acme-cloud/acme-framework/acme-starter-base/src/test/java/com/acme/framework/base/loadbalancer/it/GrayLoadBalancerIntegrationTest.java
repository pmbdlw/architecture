package com.acme.framework.base.loadbalancer.it;

import static org.assertj.core.api.Assertions.assertThat;

import com.acme.framework.core.constant.AcmeHeaders;
import java.net.URI;
import java.util.HashMap;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.cloud.client.ServiceInstance;
import org.springframework.cloud.client.loadbalancer.DefaultRequest;
import org.springframework.cloud.client.loadbalancer.RequestData;
import org.springframework.cloud.client.loadbalancer.RequestDataContext;
import org.springframework.cloud.client.loadbalancer.Response;
import org.springframework.cloud.loadbalancer.support.LoadBalancerClientFactory;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.util.LinkedMultiValueMap;
import reactor.core.publisher.Mono;

/**
 * 通过真实的 LoadBalancer 子上下文验证灰度路由（网关与 Feign 走的都是这条路径）。
 * 测试应用放在子包中，避免组件扫描把 GrayLoadBalancerClientConfiguration 注册进主上下文。
 */
@SpringBootTest(classes = GrayLoadBalancerIntegrationTest.App.class, properties = {
        "spring.main.web-application-type=none",
        "spring.cloud.nacos.discovery.enabled=false",
        "spring.cloud.nacos.config.enabled=false",
        "spring.cloud.nacos.config.import-check.enabled=false",
        "spring.cloud.service-registry.auto-registration.enabled=false",
        "spring.cloud.discovery.client.simple.instances.user-service[0].uri=http://stable-1:8080",
        "spring.cloud.discovery.client.simple.instances.user-service[1].uri=http://stable-2:8080",
        "spring.cloud.discovery.client.simple.instances.user-service[2].uri=http://gray-1:8080",
        "spring.cloud.discovery.client.simple.instances.user-service[2].metadata.gray-tag=v2",
})
class GrayLoadBalancerIntegrationTest {

    @Autowired
    private LoadBalancerClientFactory factory;

    @Test
    void grayTaggedRequestsGoToGrayInstancesOthersNeverDo() {
        for (int i = 0; i < 6; i++) {
            assertThat(choose("v2")).isEqualTo("gray-1");
            assertThat(choose(null)).startsWith("stable-");
        }
        // 没有对应灰度实例时回落到稳定版
        assertThat(choose("v3")).startsWith("stable-");
    }

    private String choose(String grayTag) {
        HttpHeaders headers = new HttpHeaders();
        if (grayTag != null) {
            headers.set(AcmeHeaders.GRAY_TAG, grayTag);
        }
        RequestData data = new RequestData(HttpMethod.GET, URI.create("http://user-service/api"), headers,
                new LinkedMultiValueMap<>(), new HashMap<>());
        Response<ServiceInstance> response = Mono.from(factory.getInstance("user-service")
                .choose(new DefaultRequest<>(new RequestDataContext(data)))).block();
        assertThat(response).isNotNull();
        return response.getServer().getHost();
    }

    @SpringBootApplication
    static class App {
    }
}
