package com.acme.order.interfaces.web;

import jakarta.validation.Valid;

import org.springframework.http.HttpStatus;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import com.acme.framework.core.result.Result;
import com.acme.order.application.CreateOrderCommand;
import com.acme.order.application.OrderAppService;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;

/**
 * 当前用户 ID 一律取自 JWT 的 {@code uid} 声明，绝不信任请求体/参数中的 userId。
 */
@Tag(name = "订单")
@RestController
@RequestMapping("/api/v1/orders")
public class OrderController {

    private final OrderAppService orderAppService;

    public OrderController(OrderAppService orderAppService) {
        this.orderAppService = orderAppService;
    }

    @Operation(summary = "下单")
    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public Result<OrderVO> create(@AuthenticationPrincipal Jwt jwt, @Valid @RequestBody CreateOrderRequest request) {
        var order = orderAppService.create(new CreateOrderCommand(currentUserId(jwt), request.productCode(),
                request.quantity(), request.amount()));
        return Result.ok(OrderVO.from(order));
    }

    @Operation(summary = "查询我的订单")
    @GetMapping("/{id}")
    public Result<OrderVO> get(@AuthenticationPrincipal Jwt jwt, @PathVariable Long id) {
        return Result.ok(OrderVO.from(orderAppService.getForBuyer(id, currentUserId(jwt))));
    }

    private static Long currentUserId(Jwt jwt) {
        return Long.valueOf(jwt.getClaimAsString("uid"));
    }
}
