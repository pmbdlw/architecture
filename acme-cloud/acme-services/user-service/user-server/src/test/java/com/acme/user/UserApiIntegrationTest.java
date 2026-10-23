package com.acme.user;

import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.jwt;
import static org.springframework.security.test.web.servlet.setup.SecurityMockMvcConfigurers.springSecurity;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.context.WebApplicationContext;

import com.acme.framework.test.AcmeServiceTest;

import tools.jackson.databind.JsonNode;
import tools.jackson.databind.json.JsonMapper;

@AcmeServiceTest
class UserApiIntegrationTest {

    @Autowired
    private WebApplicationContext context;

    private MockMvc mvc;

    private final JsonMapper json = JsonMapper.builder().build();

    @BeforeEach
    void setUp() {
        mvc = MockMvcBuilders.webAppContextSetup(context).apply(springSecurity()).build();
    }

    @Test
    void adminRegistersUserAndOtherServicesReadItViaInternalApi() throws Exception {
        String body = mvc.perform(post("/api/v1/users")
                        .with(jwt().authorities(new SimpleGrantedAuthority("ROLE_ADMIN")))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {"username":"alice","mobile":"13800001234","email":"alice@example.com"}
                                """))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.code").value("00000"))
                // 手机号脱敏
                .andExpect(jsonPath("$.data.mobile").value("138****1234"))
                // Long 型 ID 以字符串输出
                .andExpect(jsonPath("$.data.id").isString())
                .andReturn().getResponse().getContentAsString();
        JsonNode created = json.readTree(body);
        String id = created.path("data").path("id").asString();

        // 与 user-api 中 UserClient 的契约一致：GET /internal/v1/users/{id}
        mvc.perform(get("/internal/v1/users/{id}", id).with(jwt()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.username").value("alice"))
                .andExpect(jsonPath("$.data.status").value("ACTIVE"));
    }

    @Test
    void duplicateUsernameIsConflict() throws Exception {
        String request = """
                {"username":"bob"}
                """;
        mvc.perform(post("/api/v1/users").with(jwt().authorities(new SimpleGrantedAuthority("ROLE_ADMIN")))
                .contentType(MediaType.APPLICATION_JSON).content(request)).andExpect(status().isCreated());
        mvc.perform(post("/api/v1/users").with(jwt().authorities(new SimpleGrantedAuthority("ROLE_ADMIN")))
                        .contentType(MediaType.APPLICATION_JSON).content(request))
                .andExpect(status().isConflict())
                .andExpect(jsonPath("$.code").value("A1002"));
    }

    @Test
    void unknownUserIsNotFoundWithBusinessCode() throws Exception {
        mvc.perform(get("/api/v1/users/{id}", 42).with(jwt()))
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.code").value("A1001"));
    }

    @Test
    void invalidRequestIsBadRequest() throws Exception {
        mvc.perform(post("/api/v1/users").with(jwt().authorities(new SimpleGrantedAuthority("ROLE_ADMIN")))
                        .contentType(MediaType.APPLICATION_JSON).content("""
                                {"username":"x","mobile":"123"}
                                """))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.code").value("A0400"));
    }

    @Test
    void nonAdminCannotRegister() throws Exception {
        mvc.perform(post("/api/v1/users").with(jwt())
                        .contentType(MediaType.APPLICATION_JSON).content("""
                                {"username":"carol"}
                                """))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.code").value("A0403"));
    }

    @Test
    void anonymousRequestIsUnauthorized() throws Exception {
        mvc.perform(get("/api/v1/users/{id}", 1))
                .andExpect(status().isUnauthorized())
                .andExpect(header().exists("Content-Type"))
                .andExpect(jsonPath("$.code").value("A0401"));
    }
}
