package com.acme.user.application;

public record RegisterUserCommand(String username, String mobile, String email) {
}
