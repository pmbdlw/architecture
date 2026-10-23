package com.acme.auth;

import java.util.List;

import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.boot.context.properties.bind.DefaultValue;
import org.springframework.core.io.Resource;

/**
 * @param jwk       签名密钥。多副本部署必须使用同一把密钥：PEM 文件由 K8s Secret（来自 Vault / 阿里云 KMS）挂载
 * @param demoUsers 演示用户，仅用于本地与测试环境；生产接入 user-service 或企业 IdP
 */
@ConfigurationProperties(prefix = "acme.auth")
public record AuthProperties(Jwk jwk, @DefaultValue List<DemoUser> demoUsers) {

    /**
     * @param keyId      kid，轮换密钥时递增
     * @param publicKey  PEM 公钥，如 file:/etc/acme-auth/jwk/public.pem
     * @param privateKey PEM 私钥（PKCS#8）
     */
    public record Jwk(@DefaultValue("acme-dev") String keyId, Resource publicKey, Resource privateKey) {
    }

    public record DemoUser(Long uid, String username, String password, @DefaultValue List<String> roles) {
    }
}
