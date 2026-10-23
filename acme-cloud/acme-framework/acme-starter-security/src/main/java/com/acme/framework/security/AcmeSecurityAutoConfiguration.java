package com.acme.framework.security;

import java.util.ArrayList;
import java.util.Collection;
import java.util.List;

import org.springframework.boot.autoconfigure.AutoConfiguration;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.security.config.annotation.method.configuration.EnableMethodSecurity;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationConverter;
import org.springframework.security.oauth2.server.resource.authentication.JwtGrantedAuthoritiesConverter;
import org.springframework.security.web.SecurityFilterChain;

/**
 * 资源服务器默认安全策略：
 * <ul>
 *   <li>无状态，禁用 Session/CSRF；</li>
 *   <li>/actuator/** 与 API 文档放行（管理端口不对外暴露，文档在生产关闭）；</li>
 *   <li>其余请求（包括 /internal/**）都必须携带有效 JWT——网关挡住外部访问 /internal，服务自身仍做校验（纵深防御）；</li>
 *   <li>权限：JWT 的 scope → {@code SCOPE_xxx}，roles → {@code ROLE_xxx}，配合 {@code @PreAuthorize} 使用。</li>
 * </ul>
 * 服务如需定制，声明自己的 {@link SecurityFilterChain} Bean 即可替换。
 */
@AutoConfiguration
@EnableMethodSecurity
public class AcmeSecurityAutoConfiguration {

    private static final String[] PUBLIC_PATHS = {
            "/actuator/**", "/v3/api-docs/**", "/swagger-ui/**", "/swagger-ui.html", "/error"
    };

    @Bean
    public SecurityExceptionHandler securityExceptionHandler() {
        return new SecurityExceptionHandler();
    }

    @Bean
    @ConditionalOnMissingBean(SecurityFilterChain.class)
    public SecurityFilterChain acmeSecurityFilterChain(HttpSecurity http) throws Exception {
        http.csrf(csrf -> csrf.disable())
                .sessionManagement(session -> session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
                .authorizeHttpRequests(auth -> auth
                        .requestMatchers(PUBLIC_PATHS).permitAll()
                        .anyRequest().authenticated())
                .oauth2ResourceServer(oauth2 -> oauth2
                        .jwt(jwt -> jwt.jwtAuthenticationConverter(jwtAuthenticationConverter()))
                        .authenticationEntryPoint(ResultAuthenticationHandlers.entryPoint())
                        .accessDeniedHandler(ResultAuthenticationHandlers.accessDeniedHandler()))
                .exceptionHandling(ex -> ex
                        .authenticationEntryPoint(ResultAuthenticationHandlers.entryPoint())
                        .accessDeniedHandler(ResultAuthenticationHandlers.accessDeniedHandler()));
        return http.build();
    }

    static JwtAuthenticationConverter jwtAuthenticationConverter() {
        JwtGrantedAuthoritiesConverter scopes = new JwtGrantedAuthoritiesConverter();
        JwtAuthenticationConverter converter = new JwtAuthenticationConverter();
        converter.setJwtGrantedAuthoritiesConverter(jwt -> {
            Collection<GrantedAuthority> authorities = new ArrayList<>(scopes.convert(jwt));
            authorities.addAll(roles(jwt));
            return authorities;
        });
        return converter;
    }

    private static List<GrantedAuthority> roles(Jwt jwt) {
        List<String> roles = jwt.getClaimAsStringList("roles");
        if (roles == null) {
            return List.of();
        }
        return roles.stream().<GrantedAuthority>map(role -> new SimpleGrantedAuthority("ROLE_" + role)).toList();
    }
}
