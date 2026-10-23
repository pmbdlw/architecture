package com.acme.user.interfaces.web;

import jakarta.validation.constraints.Email;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;

public record RegisterUserRequest(
        @NotBlank @Size(min = 3, max = 64) String username,
        @Pattern(regexp = "^1\\d{10}$", message = "手机号格式不正确") String mobile,
        @Email String email) {
}
