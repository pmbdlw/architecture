package com.acme.gateway;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.context.ApplicationContext;
import org.springframework.test.web.reactive.server.WebTestClient;

@SpringBootTest(properties = {
        "spring.cloud.nacos.discovery.enabled=false",
        "spring.cloud.nacos.config.enabled=false",
        "spring.cloud.nacos.config.import-check.enabled=false",
        "spring.cloud.service-registry.auto-registration.enabled=false",
        "management.server.port=0",
        "management.tracing.export.enabled=false"
})
class GatewayApplicationTest {

    @Autowired
    private ApplicationContext context;

    private WebTestClient client;

    @BeforeEach
    void setUp() {
        client = WebTestClient.bindToApplicationContext(context).build();
    }

    @Test
    void internalApisAreNeverExposed() {
        client.get().uri("/api/v1/users/internal/1").exchange()
                .expectStatus().isNotFound()
                .expectBody().jsonPath("$.code").isEqualTo("A0404");
    }

    @Test
    void anonymousRequestsAreRejectedAtTheEdge() {
        client.get().uri("/api/v1/orders/1").exchange()
                .expectStatus().isUnauthorized()
                .expectBody().jsonPath("$.code").isEqualTo("A0401");
    }

    @Test
    void fallbackReturnsUnifiedResult() {
        client.get().uri("/fallback/order-service").exchange()
                .expectStatus().isEqualTo(503)
                .expectBody().jsonPath("$.code").isEqualTo("C0503");
    }
}
