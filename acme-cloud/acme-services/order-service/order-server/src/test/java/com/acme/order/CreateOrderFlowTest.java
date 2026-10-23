package com.acme.order;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.BDDMockito.given;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.jwt;
import static org.springframework.security.test.web.servlet.setup.SecurityMockMvcConfigurers.springSecurity;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.nio.charset.StandardCharsets;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.cloud.stream.binder.test.OutputDestination;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.messaging.Message;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.context.WebApplicationContext;

import com.acme.framework.core.exception.BizException;
import com.acme.framework.core.exception.RemoteErrorCode;
import com.acme.framework.core.exception.RemoteServiceException;
import com.acme.framework.core.result.Result;
import com.acme.framework.mq.EventHeaders;
import com.acme.framework.mq.OutboxRelay;
import com.acme.framework.test.AcmeServiceTest;
import com.acme.order.api.event.OrderEventTypes;
import com.acme.user.api.UserClient;
import com.acme.user.api.UserDTO;

import tools.jackson.databind.JsonNode;
import tools.jackson.databind.json.JsonMapper;

/**
 * 端到端验证：HTTP 下单 → Feign 校验买家 → 订单与发件箱同事务落库 → Relay 投递到 MQ（Test Binder）。
 */
@AcmeServiceTest
class CreateOrderFlowTest {

    @Autowired
    private WebApplicationContext context;

    @Autowired
    private JdbcTemplate jdbcTemplate;

    @Autowired
    private OutboxRelay outboxRelay;

    @Autowired
    private OutputDestination output;

    @MockitoBean
    private UserClient userClient;

    private MockMvc mvc;

    private final JsonMapper json = JsonMapper.builder().build();

    @BeforeEach
    void setUp() {
        mvc = MockMvcBuilders.webAppContextSetup(context).apply(springSecurity()).build();
        output.clear();
    }

    @Test
    void createOrderPublishesOrderCreatedEventThroughOutbox() throws Exception {
        given(userClient.getUser(1001L)).willReturn(Result.ok(new UserDTO(1001L, "alice", "ACTIVE")));

        String body = mvc.perform(post("/api/v1/orders")
                        .with(jwt().jwt(token -> token.claim("uid", "1001")))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {"productCode":"SKU-1","quantity":2,"amount":99.80}
                                """))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.data.status").value("CREATED"))
                .andReturn().getResponse().getContentAsString();
        String orderId = json.readTree(body).path("data").path("id").asString();

        // 订单与事件在同一事务中写入
        assertThat(jdbcTemplate.queryForObject(
                "SELECT COUNT(*) FROM outbox_event WHERE aggregate_id = ? AND status = 0", Integer.class, orderId))
                .isEqualTo(1);

        outboxRelay.relay();

        Message<byte[]> message = output.receive(1000, OrderEventTypes.DESTINATION);
        assertThat(message).isNotNull();
        assertThat(message.getHeaders().get(EventHeaders.TYPE)).isEqualTo(OrderEventTypes.ORDER_CREATED);
        assertThat(message.getHeaders().get(EventHeaders.SOURCE)).isEqualTo("order-service");
        JsonNode event = json.readTree(new String(message.getPayload(), StandardCharsets.UTF_8));
        assertThat(event.path("orderId").asString()).isEqualTo(orderId);
        assertThat(event.path("userId").asLong()).isEqualTo(1001L);
        assertThat(jdbcTemplate.queryForObject(
                "SELECT status FROM outbox_event WHERE aggregate_id = ?", Integer.class, orderId)).isEqualTo(1);

        // 只能看到自己的订单
        mvc.perform(get("/api/v1/orders/{id}", orderId).with(jwt().jwt(token -> token.claim("uid", "1001"))))
                .andExpect(status().isOk());
        mvc.perform(get("/api/v1/orders/{id}", orderId).with(jwt().jwt(token -> token.claim("uid", "2002"))))
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.code").value("A2001"));
    }

    @Test
    void unknownBuyerCannotPlaceOrder() throws Exception {
        given(userClient.getUser(anyLong()))
                .willThrow(new BizException(new RemoteErrorCode("A1001", "用户不存在", 404), "用户不存在", null));

        mvc.perform(post("/api/v1/orders")
                        .with(jwt().jwt(token -> token.claim("uid", "9")))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {"productCode":"SKU-1","quantity":1,"amount":1}
                                """))
                .andExpect(status().isUnprocessableContent())
                .andExpect(jsonPath("$.code").value("A2002"));
        assertThat(jdbcTemplate.queryForObject("SELECT COUNT(*) FROM outbox_event WHERE aggregate_id = '9'",
                Integer.class)).isZero();
    }

    @Test
    void userServiceOutageIsReportedAsUpstreamError() throws Exception {
        given(userClient.getUser(anyLong())).willThrow(new RemoteServiceException("user-service", 503, "down"));

        mvc.perform(post("/api/v1/orders")
                        .with(jwt().jwt(token -> token.claim("uid", "1001")))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {"productCode":"SKU-1","quantity":1,"amount":1}
                                """))
                .andExpect(status().isServiceUnavailable())
                .andExpect(jsonPath("$.code").value("C0503"));
    }
}
