package com.acme.notification;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.jwt;
import static org.springframework.security.test.web.servlet.setup.SecurityMockMvcConfigurers.springSecurity;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import java.util.UUID;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.cloud.stream.binder.test.InputDestination;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.messaging.Message;
import org.springframework.messaging.support.MessageBuilder;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.context.WebApplicationContext;

import com.acme.framework.mq.EventHeaders;
import com.acme.framework.test.AcmeServiceTest;
import com.acme.order.api.event.OrderEventTypes;

@AcmeServiceTest
class OrderCreatedConsumerTest {

    @Autowired
    private InputDestination input;

    @Autowired
    private JdbcTemplate jdbcTemplate;

    @Autowired
    private WebApplicationContext context;

    @Test
    void consumesOrderCreatedExactlyOnceEvenIfRedelivered() throws Exception {
        String eventId = UUID.randomUUID().toString();
        Message<byte[]> message = MessageBuilder.withPayload("""
                        {"orderId":1,"orderNo":"O1","userId":1001,"productCode":"SKU-1","quantity":1,
                         "amount":9.90,"createdAt":"2026-10-01T08:00:00Z","newFieldFromFutureVersion":"ignored"}
                        """.getBytes())
                .setHeader("contentType", "application/json")
                .setHeader(EventHeaders.ID, eventId)
                .setHeader(EventHeaders.TYPE, OrderEventTypes.ORDER_CREATED)
                .build();

        input.send(message, OrderEventTypes.DESTINATION);
        // MQ 至少一次投递：同一事件重复到达
        input.send(message, OrderEventTypes.DESTINATION);

        assertThat(jdbcTemplate.queryForObject("SELECT COUNT(*) FROM notification WHERE biz_key = 'O1'",
                Integer.class)).isEqualTo(1);
        assertThat(jdbcTemplate.queryForObject("SELECT COUNT(*) FROM consumed_event WHERE event_id = ?",
                Integer.class, eventId)).isEqualTo(1);

        MockMvc mvc = MockMvcBuilders.webAppContextSetup(context).apply(springSecurity()).build();
        mvc.perform(get("/api/v1/notifications").with(jwt().jwt(token -> token.claim("uid", "1001"))))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.data.total").value("1"))
                .andExpect(jsonPath("$.data.records[0].title").value("下单成功"));
    }

    @Test
    void ignoresOtherEventTypesOnSameTopic() {
        String eventId = UUID.randomUUID().toString();
        input.send(MessageBuilder.withPayload("{}".getBytes())
                .setHeader("contentType", "application/json")
                .setHeader(EventHeaders.ID, eventId)
                .setHeader(EventHeaders.TYPE, "order.cancelled")
                .build(), OrderEventTypes.DESTINATION);

        assertThat(jdbcTemplate.queryForObject("SELECT COUNT(*) FROM consumed_event WHERE event_id = ?",
                Integer.class, eventId)).isZero();
    }
}
