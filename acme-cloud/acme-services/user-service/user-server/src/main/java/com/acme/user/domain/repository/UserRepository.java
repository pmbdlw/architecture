package com.acme.user.domain.repository;

import java.util.Optional;

import com.acme.user.domain.model.User;

/**
 * 仓储接口定义在领域层，实现放在 infrastructure.persistence。
 */
public interface UserRepository {

    Long nextId();

    Optional<User> findById(Long id);

    boolean existsByUsername(String username);

    void add(User user);
}
