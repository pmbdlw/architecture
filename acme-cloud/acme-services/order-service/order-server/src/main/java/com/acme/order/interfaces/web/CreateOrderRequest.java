package com.acme.order.interfaces.web;

import java.math.BigDecimal;

import jakarta.validation.constraints.DecimalMin;
import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;

public record CreateOrderRequest(
        @NotBlank String productCode,
        @Min(1) @Max(999) int quantity,
        @NotNull @DecimalMin("0.01") BigDecimal amount) {
}
