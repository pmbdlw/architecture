package com.acme.framework.base.loadbalancer;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.List;
import java.util.Map;

import org.junit.jupiter.api.Test;
import org.springframework.cloud.client.DefaultServiceInstance;
import org.springframework.cloud.client.ServiceInstance;

class GrayServiceInstanceListSupplierTest {

    private final ServiceInstance stable1 = instance("s1", Map.of());
    private final ServiceInstance stable2 = instance("s2", Map.of("gray-tag", ""));
    private final ServiceInstance grayV2 = instance("g1", Map.of("gray-tag", "v2"));

    @Test
    void normalTrafficOnlyHitsStableInstances() {
        assertThat(GrayServiceInstanceListSupplier.select(List.of(stable1, stable2, grayV2), null))
                .containsExactly(stable1, stable2);
    }

    @Test
    void taggedTrafficHitsMatchingGrayInstances() {
        assertThat(GrayServiceInstanceListSupplier.select(List.of(stable1, grayV2), "v2"))
                .containsExactly(grayV2);
    }

    @Test
    void taggedTrafficFallsBackToStableWhenNoGrayInstance() {
        assertThat(GrayServiceInstanceListSupplier.select(List.of(stable1, grayV2), "v3"))
                .containsExactly(stable1);
    }

    @Test
    void allTaggedInstancesStayAvailable() {
        assertThat(GrayServiceInstanceListSupplier.select(List.of(grayV2), null)).containsExactly(grayV2);
    }

    private static ServiceInstance instance(String id, Map<String, String> metadata) {
        return new DefaultServiceInstance(id, "user-service", "10.0.0." + id.length(), 8080, false, metadata);
    }
}
