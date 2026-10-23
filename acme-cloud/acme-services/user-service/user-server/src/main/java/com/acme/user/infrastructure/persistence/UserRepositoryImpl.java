package com.acme.user.infrastructure.persistence;

import java.util.Optional;

import org.springframework.stereotype.Repository;

import com.acme.user.domain.model.User;
import com.acme.user.domain.repository.UserRepository;
import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.core.toolkit.IdWorker;

@Repository
public class UserRepositoryImpl implements UserRepository {

    private final UserMapper userMapper;
    private final UserPersistenceConverter converter;

    public UserRepositoryImpl(UserMapper userMapper, UserPersistenceConverter converter) {
        this.userMapper = userMapper;
        this.converter = converter;
    }

    @Override
    public Long nextId() {
        return IdWorker.getId();
    }

    @Override
    public Optional<User> findById(Long id) {
        return Optional.ofNullable(userMapper.selectById(id)).map(converter::toDomain);
    }

    @Override
    public boolean existsByUsername(String username) {
        return userMapper.exists(new LambdaQueryWrapper<UserDO>().eq(UserDO::getUsername, username));
    }

    @Override
    public void add(User user) {
        userMapper.insert(converter.toData(user));
    }
}
