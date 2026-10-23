package com.acme.order.infrastructure.persistence;

import java.time.ZoneId;
import java.util.Optional;

import org.springframework.stereotype.Repository;

import com.acme.order.domain.model.Order;
import com.acme.order.domain.model.OrderStatus;
import com.acme.order.domain.repository.OrderRepository;
import com.baomidou.mybatisplus.core.toolkit.IdWorker;

@Repository
public class OrderRepositoryImpl implements OrderRepository {

    private final OrderMapper orderMapper;

    public OrderRepositoryImpl(OrderMapper orderMapper) {
        this.orderMapper = orderMapper;
    }

    @Override
    public Long nextId() {
        return IdWorker.getId();
    }

    @Override
    public Optional<Order> findById(Long id) {
        return Optional.ofNullable(orderMapper.selectById(id)).map(OrderRepositoryImpl::toDomain);
    }

    @Override
    public void add(Order order) {
        OrderDO data = new OrderDO();
        data.setId(order.getId());
        data.setOrderNo(order.getOrderNo());
        data.setUserId(order.getUserId());
        data.setProductCode(order.getProductCode());
        data.setQuantity(order.getQuantity());
        data.setAmount(order.getAmount());
        data.setStatus(order.getStatus().name());
        orderMapper.insert(data);
    }

    private static Order toDomain(OrderDO data) {
        return new Order(data.getId(), data.getOrderNo(), data.getUserId(), data.getProductCode(),
                data.getQuantity(), data.getAmount(), OrderStatus.valueOf(data.getStatus()),
                data.getCreatedAt().atZone(ZoneId.systemDefault()).toInstant());
    }
}
