package com.acme.user.api;

import org.springframework.cloud.openfeign.FeignClient;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;

import com.acme.framework.core.result.Result;

/**
 * user-service 内部接口（/internal/**，网关不对外暴露）。
 * <p>
 * name 为 Nacos 中的服务名，经 LoadBalancer 解析为 Pod IP；contextId 全局唯一，用于单独配置超时与熔断。
 * 使用方：{@code @EnableFeignClients(clients = UserClient.class)}。
 */
@FeignClient(name = UserClient.SERVICE_NAME, contextId = "userClient", path = "/internal/v1/users")
public interface UserClient {

    String SERVICE_NAME = "user-service";

    @GetMapping("/{id}")
    Result<UserDTO> getUser(@PathVariable("id") Long id);
}
