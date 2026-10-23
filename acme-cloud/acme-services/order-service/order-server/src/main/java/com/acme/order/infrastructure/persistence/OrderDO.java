package com.acme.order.infrastructure.persistence;

import java.math.BigDecimal;

import com.acme.framework.mybatis.BaseDO;
import com.baomidou.mybatisplus.annotation.TableName;

import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@TableName("order_info")
public class OrderDO extends BaseDO {

    private String orderNo;
    private Long userId;
    private String productCode;
    private Integer quantity;
    private BigDecimal amount;
    private String status;
}
