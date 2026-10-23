package com.acme.order.infrastructure.remote;

import org.springframework.stereotype.Component;

import com.acme.framework.core.exception.BizException;
import com.acme.framework.core.result.Result;
import com.acme.order.domain.gateway.BuyerGateway;
import com.acme.user.api.UserClient;
import com.acme.user.api.UserDTO;

/**
 * 通过 Feign 调用 user-service。
 * <ul>
 *   <li>下游 4xx（用户不存在）→ BizException，这里翻译为“不可下单”；</li>
 *   <li>下游 5xx / 超时 / 熔断打开 → RemoteServiceException 原样抛出，对外表现为 C0001/C0503。</li>
 * </ul>
 */
@Component
public class BuyerGatewayImpl implements BuyerGateway {

    private final UserClient userClient;

    public BuyerGatewayImpl(UserClient userClient) {
        this.userClient = userClient;
    }

    @Override
    public boolean canPlaceOrder(Long userId) {
        try {
            Result<UserDTO> result = userClient.getUser(userId);
            return result.isSuccess() && result.data() != null && result.data().isActive();
        } catch (BizException ex) {
            return false;
        }
    }
}
