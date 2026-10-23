package com.acme.framework.test;

import java.lang.annotation.Documented;
import java.lang.annotation.ElementType;
import java.lang.annotation.Inherited;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;

import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.cloud.stream.binder.test.TestChannelBinderConfiguration;
import org.springframework.context.annotation.Import;
import org.springframework.test.context.TestPropertySource;

/**
 * 服务级集成测试注解：启动完整 Spring 上下文，但把外部依赖替换为进程内实现——
 * Nacos 关闭、数据库换成 H2（MySQL 模式，执行真实的 Flyway 脚本）、消息中间件换成 Stream Test Binder。
 * 需要真实 MySQL/PostgreSQL/Kafka 时，改用 Testcontainers 并在 CI 中提供 Docker。
 */
@Target(ElementType.TYPE)
@Retention(RetentionPolicy.RUNTIME)
@Documented
@Inherited
@SpringBootTest
@Import(TestChannelBinderConfiguration.class)
@TestPropertySource(locations = "classpath:acme-test.properties")
public @interface AcmeServiceTest {
}
