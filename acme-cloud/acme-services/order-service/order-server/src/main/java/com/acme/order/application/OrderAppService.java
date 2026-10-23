package com.acme.order.application;

import java.time.Clock;
import java.time.Instant;

import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.support.TransactionTemplate;

import com.acme.framework.core.exception.BizException;
import com.acme.framework.mq.OutboxPublisher;
import com.acme.order.api.event.OrderCreatedEvent;
import com.acme.order.api.event.OrderEventTypes;
import com.acme.order.domain.OrderErrorCode;
import com.acme.order.domain.gateway.BuyerGateway;
import com.acme.order.domain.model.Order;
import com.acme.order.domain.repository.OrderRepository;

@Service
public class OrderAppService {

    /** Spring Cloud Stream 输出绑定名，对应配置 spring.cloud.stream.bindings.orderEvents-out-0 */
    static final String ORDER_EVENTS_BINDING = "orderEvents-out-0";

    private final OrderRepository orderRepository;
    private final BuyerGateway buyerGateway;
    private final OutboxPublisher outboxPublisher;
    private final TransactionTemplate transactionTemplate;
    private final Clock clock;

    public OrderAppService(OrderRepository orderRepository, BuyerGateway buyerGateway,
            OutboxPublisher outboxPublisher, TransactionTemplate transactionTemplate, Clock clock) {
        this.orderRepository = orderRepository;
        this.buyerGateway = buyerGateway;
        this.outboxPublisher = outboxPublisher;
        this.transactionTemplate = transactionTemplate;
        this.clock = clock;
    }

    /**
     * 下单。
     * <ol>
     *   <li>远程校验买家——放在事务之外，避免远程调用期间占用数据库连接；</li>
     *   <li>本地事务内：保存订单 + 写入发件箱，二者原子提交；</li>
     *   <li>OutboxRelay 异步把事件投递到 MQ，notification-service 等订阅方消费。</li>
     * </ol>
     */
    public Order create(CreateOrderCommand command) {
        if (!buyerGateway.canPlaceOrder(command.userId())) {
            throw new BizException(OrderErrorCode.BUYER_NOT_AVAILABLE);
        }
        return transactionTemplate.execute(status -> {
            Order order = Order.create(orderRepository.nextId(), command.userId(), command.productCode(),
                    command.quantity(), command.amount(), Instant.now(clock));
            orderRepository.add(order);
            outboxPublisher.publish(ORDER_EVENTS_BINDING, OrderEventTypes.ORDER_CREATED, order.getId().toString(),
                    new OrderCreatedEvent(order.getId(), order.getOrderNo(), order.getUserId(),
                            order.getProductCode(), order.getQuantity(), order.getAmount(), order.getCreatedAt()));
            return order;
        });
    }

    @Transactional(readOnly = true)
    public Order getForBuyer(Long orderId, Long buyerId) {
        return orderRepository.findById(orderId)
                .filter(order -> order.belongsTo(buyerId))
                // 不属于当前用户的订单同样返回“不存在”，避免泄露订单是否存在
                .orElseThrow(() -> new BizException(OrderErrorCode.ORDER_NOT_FOUND, orderId));
    }
}
