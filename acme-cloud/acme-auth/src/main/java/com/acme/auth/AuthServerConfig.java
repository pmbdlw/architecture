package com.acme.auth;

import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.NoSuchAlgorithmException;
import java.security.interfaces.RSAPrivateKey;
import java.security.interfaces.RSAPublicKey;
import java.util.List;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.converter.RsaKeyConverters;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.userdetails.User;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.oauth2.server.authorization.OAuth2TokenType;
import org.springframework.security.oauth2.server.authorization.token.JwtEncodingContext;
import org.springframework.security.oauth2.server.authorization.token.OAuth2TokenCustomizer;
import org.springframework.security.provisioning.InMemoryUserDetailsManager;

import com.nimbusds.jose.jwk.JWKSet;
import com.nimbusds.jose.jwk.RSAKey;
import com.nimbusds.jose.jwk.source.ImmutableJWKSet;
import com.nimbusds.jose.jwk.source.JWKSource;
import com.nimbusds.jose.proc.SecurityContext;

/**
 * 认证中心配置：客户端注册见 application.yml（spring.security.oauth2.authorizationserver.client.*）。
 */
@Configuration(proxyBeanMethods = false)
public class AuthServerConfig {

    private static final Logger log = LoggerFactory.getLogger(AuthServerConfig.class);

    @Bean
    public JWKSource<SecurityContext> jwkSource(AuthProperties properties) throws Exception {
        AuthProperties.Jwk jwk = properties.jwk();
        RSAKey rsaKey;
        if (jwk != null && jwk.privateKey() != null && jwk.privateKey().exists()) {
            RSAPublicKey publicKey = RsaKeyConverters.x509().convert(jwk.publicKey().getInputStream());
            RSAPrivateKey privateKey = RsaKeyConverters.pkcs8().convert(jwk.privateKey().getInputStream());
            rsaKey = new RSAKey.Builder(publicKey).privateKey(privateKey).keyID(jwk.keyId()).build();
        } else {
            log.warn("未配置 acme.auth.jwk，使用临时生成的 RSA 密钥——仅限本地开发，多副本部署会导致令牌互不认可");
            KeyPair keyPair = generateRsaKey();
            rsaKey = new RSAKey.Builder((RSAPublicKey) keyPair.getPublic())
                    .privateKey((RSAPrivateKey) keyPair.getPrivate())
                    .keyID(jwk == null ? "acme-dev" : jwk.keyId())
                    .build();
        }
        return new ImmutableJWKSet<>(new JWKSet(rsaKey));
    }

    @Bean
    public UserDetailsService userDetailsService(AuthProperties properties) {
        List<UserDetails> users = properties.demoUsers().stream()
                .map(user -> User.withUsername(user.username())
                        .password(user.password())
                        .roles(user.roles().toArray(String[]::new))
                        .build())
                .toList();
        return new InMemoryUserDetailsManager(users);
    }

    /**
     * 在 access token 中加入平台约定的声明：
     * {@code uid}（用户 ID，资源服务以此识别当前用户）与 {@code roles}（映射为 ROLE_xxx）。
     */
    @Bean
    public OAuth2TokenCustomizer<JwtEncodingContext> acmeTokenCustomizer(AuthProperties properties) {
        return context -> {
            if (!OAuth2TokenType.ACCESS_TOKEN.equals(context.getTokenType())) {
                return;
            }
            String username = context.getPrincipal().getName();
            properties.demoUsers().stream()
                    .filter(user -> user.username().equals(username))
                    .findFirst()
                    .ifPresent(user -> context.getClaims()
                            .claim("uid", String.valueOf(user.uid()))
                            .claim("roles", context.getPrincipal().getAuthorities().stream()
                                    .map(GrantedAuthority::getAuthority)
                                    .filter(authority -> authority.startsWith("ROLE_"))
                                    .map(authority -> authority.substring("ROLE_".length()))
                                    .toList()));
        };
    }

    private static KeyPair generateRsaKey() throws NoSuchAlgorithmException {
        KeyPairGenerator generator = KeyPairGenerator.getInstance("RSA");
        generator.initialize(2048);
        return generator.generateKeyPair();
    }
}
