package com.acme.user.application;

import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import com.acme.framework.core.exception.BizException;
import com.acme.user.domain.UserErrorCode;
import com.acme.user.domain.model.User;
import com.acme.user.domain.repository.UserRepository;

/**
 * 应用服务：用例编排与事务边界。不写业务规则（规则在领域对象里），不处理 HTTP 细节。
 */
@Service
public class UserAppService {

    private final UserRepository userRepository;

    public UserAppService(UserRepository userRepository) {
        this.userRepository = userRepository;
    }

    @Transactional
    public User register(RegisterUserCommand command) {
        if (userRepository.existsByUsername(command.username())) {
            throw new BizException(UserErrorCode.USERNAME_TAKEN, command.username());
        }
        User user = User.register(userRepository.nextId(), command.username(), command.mobile(), command.email());
        userRepository.add(user);
        return user;
    }

    @Transactional(readOnly = true)
    public User get(Long id) {
        return userRepository.findById(id).orElseThrow(() -> new BizException(UserErrorCode.USER_NOT_FOUND, id));
    }
}
