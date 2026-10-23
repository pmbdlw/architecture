package com.acme.auth;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.security.test.web.servlet.setup.SecurityMockMvcConfigurers.springSecurity;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.nio.charset.StandardCharsets;
import java.util.Base64;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.context.WebApplicationContext;

import tools.jackson.databind.JsonNode;
import tools.jackson.databind.json.JsonMapper;

@SpringBootTest(properties = {
        "spring.cloud.nacos.discovery.enabled=false",
        "spring.cloud.nacos.config.enabled=false",
        "spring.cloud.nacos.config.import-check.enabled=false",
        "spring.cloud.service-registry.auto-registration.enabled=false",
        "management.server.port=0",
        "management.tracing.export.enabled=false"
})
class AuthServerTest {

    @Autowired
    private WebApplicationContext context;

    private MockMvc mvc;

    private final JsonMapper json = JsonMapper.builder().build();

    @BeforeEach
    void setUp() {
        mvc = MockMvcBuilders.webAppContextSetup(context).apply(springSecurity()).build();
    }

    @Test
    void publishesOidcDiscoveryAndJwks() throws Exception {
        mvc.perform(get("/.well-known/openid-configuration"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.issuer").value("http://127.0.0.1:9000"));
        mvc.perform(get("/oauth2/jwks"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.keys[0].kid").value("acme-dev"));
    }

    @Test
    void issuesClientCredentialsToken() throws Exception {
        String basic = Base64.getEncoder()
                .encodeToString("acme-internal:local-only-secret".getBytes(StandardCharsets.UTF_8));
        String body = mvc.perform(post("/oauth2/token")
                        .header(HttpHeaders.AUTHORIZATION, "Basic " + basic)
                        .contentType(MediaType.APPLICATION_FORM_URLENCODED)
                        .content("grant_type=client_credentials&scope=internal"))
                .andExpect(status().isOk())
                .andReturn().getResponse().getContentAsString();

        String accessToken = json.readTree(body).path("access_token").asString();
        JsonNode claims = json.readTree(Base64.getUrlDecoder().decode(accessToken.split("\\.")[1]));
        assertThat(claims.path("iss").asString()).isEqualTo("http://127.0.0.1:9000");
        assertThat(claims.path("sub").asString()).isEqualTo("acme-internal");
    }
}
