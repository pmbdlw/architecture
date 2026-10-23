package com.acme.user.interfaces.internal;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import com.acme.framework.core.result.Result;
import com.acme.user.api.UserClient;
import com.acme.user.api.UserDTO;
import com.acme.user.application.UserAppService;
import com.acme.user.domain.model.User;

/**
 * 内部 API：实现 {@link UserClient} 契约，仅供集群内服务调用。
 * 不直接 implements UserClient，避免 Feign 注解与 MVC 注解耦合；契约一致性由 UserApiIntegrationTest 覆盖。
 */
@RestController
@RequestMapping("/internal/v1/users")
public class UserInternalController {

    private final UserAppService userAppService;

    public UserInternalController(UserAppService userAppService) {
        this.userAppService = userAppService;
    }

    @GetMapping("/{id}")
    public Result<UserDTO> getUser(@PathVariable Long id) {
        User user = userAppService.get(id);
        return Result.ok(new UserDTO(user.getId(), user.getUsername(), user.getStatus().name()));
    }
}
