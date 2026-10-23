package com.acme.notification.interfaces.web;

import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.Min;

import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import com.acme.framework.core.result.PageResult;
import com.acme.framework.core.result.Result;
import com.acme.notification.application.NotificationAppService;
import com.acme.notification.domain.model.Notification;

import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;

@Tag(name = "通知")
@RestController
@RequestMapping("/api/v1/notifications")
public class NotificationController {

    private final NotificationAppService notificationAppService;

    public NotificationController(NotificationAppService notificationAppService) {
        this.notificationAppService = notificationAppService;
    }

    @Operation(summary = "分页查询我的通知")
    @GetMapping
    public Result<PageResult<Notification>> mine(@AuthenticationPrincipal Jwt jwt,
            @RequestParam(defaultValue = "1") @Min(1) long pageNo,
            @RequestParam(defaultValue = "20") @Min(1) @Max(100) long pageSize) {
        Long userId = Long.valueOf(jwt.getClaimAsString("uid"));
        return Result.ok(notificationAppService.pageMine(userId, pageNo, pageSize));
    }
}
