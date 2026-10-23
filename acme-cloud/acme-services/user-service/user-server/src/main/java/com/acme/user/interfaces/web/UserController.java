package com.acme.user.interfaces.web;

import jakarta.validation.Valid;

import org.springframework.http.HttpStatus;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import com.acme.framework.core.result.Result;
import com.acme.user.application.RegisterUserCommand;
import com.acme.user.application.UserAppService;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;

/**
 * 对外 API：经网关路由 {@code /api/v1/users/**}。
 */
@Tag(name = "用户")
@RestController
@RequestMapping("/api/v1/users")
public class UserController {

    private final UserAppService userAppService;

    public UserController(UserAppService userAppService) {
        this.userAppService = userAppService;
    }

    @Operation(summary = "注册用户（管理员）")
    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    @PreAuthorize("hasRole('ADMIN')")
    public Result<UserVO> register(@Valid @RequestBody RegisterUserRequest request) {
        var user = userAppService.register(
                new RegisterUserCommand(request.username(), request.mobile(), request.email()));
        return Result.ok(UserVO.from(user));
    }

    @Operation(summary = "查询用户")
    @GetMapping("/{id}")
    public Result<UserVO> get(@PathVariable Long id) {
        return Result.ok(UserVO.from(userAppService.get(id)));
    }
}
