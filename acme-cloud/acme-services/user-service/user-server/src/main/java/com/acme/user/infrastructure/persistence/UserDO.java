package com.acme.user.infrastructure.persistence;

import com.acme.framework.mybatis.BaseDO;
import com.baomidou.mybatisplus.annotation.TableName;

import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@TableName("user_account")
public class UserDO extends BaseDO {

    private String username;
    private String mobile;
    private String email;
    private String status;
}
