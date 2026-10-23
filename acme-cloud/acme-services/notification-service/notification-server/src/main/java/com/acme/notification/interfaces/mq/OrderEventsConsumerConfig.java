package com.acme.notification.interfaces.mq;

import java.util.function.Consumer;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.messaging.Message;

import com.acme.framework.mq.EventHeaders;
import com.acme.framework.mq.IdempotentConsumer;
import com.acme.notification.application.NotificationAppService;
import com.acme.order.api.event.OrderCreatedEvent;
import com.acme.order.api.event.OrderEventTypes;

/**
 * 消息入口（与 Controller 同属 interfaces 层）。
 * <p>
 * Spring Cloud Stream 函数式模型：Bean 名 {@code orderCreated} → 绑定 {@code orderCreated-in-0}，
 * 对应 topic 与消费组见 application.yml。Kafka 与 RabbitMQ 共用同一份代码。
 */
@Configuration(proxyBeanMethods = false)
public class OrderEventsConsumerConfig {

    private static final Logger log = LoggerFactory.getLogger(OrderEventsConsumerConfig.class);

    @Bean
    public Consumer<Message<OrderCreatedEvent>> orderCreated(IdempotentConsumer idempotentConsumer,
            NotificationAppService notificationAppService) {
        return message -> {
            String type = message.getHeaders().get(EventHeaders.TYPE, String.class);
            if (!OrderEventTypes.ORDER_CREATED.equals(type)) {
                // 同一 topic 中的其它订单事件，本消费者不关心
                return;
            }
            boolean handled = idempotentConsumer.consume(message, notificationAppService::onOrderCreated);
            log.info("消费 {} eventId={} handled={}", type, message.getHeaders().get(EventHeaders.ID), handled);
        };
    }
}
