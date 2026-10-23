package com.acme.order.domain.repository;

import java.util.Optional;

import com.acme.order.domain.model.Order;

public interface OrderRepository {

    Long nextId();

    Optional<Order> findById(Long id);

    void add(Order order);
}
