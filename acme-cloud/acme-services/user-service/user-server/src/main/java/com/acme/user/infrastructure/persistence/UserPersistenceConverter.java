package com.acme.user.infrastructure.persistence;

import org.mapstruct.Mapper;
import org.mapstruct.Mapping;

import com.acme.user.domain.model.User;
import com.acme.user.domain.model.UserStatus;

/**
 * DO ↔ 领域对象转换（MapStruct 编译期生成，零反射）。
 */
@Mapper
public interface UserPersistenceConverter {

    default User toDomain(UserDO data) {
        return new User(data.getId(), data.getUsername(), data.getMobile(), data.getEmail(),
                UserStatus.valueOf(data.getStatus()));
    }

    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    @Mapping(target = "deleted", ignore = true)
    @Mapping(target = "version", ignore = true)
    UserDO toData(User user);
}
