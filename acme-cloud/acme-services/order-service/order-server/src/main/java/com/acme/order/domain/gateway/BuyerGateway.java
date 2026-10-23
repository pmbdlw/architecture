package com.acme.order.domain.gateway;

/**
 * 防腐层接口：订单域只关心“买家是否可下单”，不依赖 user-service 的数据结构。
 * 实现位于 infrastructure.remote（通过 Feign 调用 user-service）。
 */
public interface BuyerGateway {

    boolean canPlaceOrder(Long userId);
}
