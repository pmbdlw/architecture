# acme-cloud 初始工程

Spring Boot 4.0.9 · Spring Cloud 2025.1.4 · Spring Cloud Alibaba 2025.1.0.0 · Java 21

完整说明见 [第 4 章 初始代码结构](../docs/04-code-structure.md)，团队规范见 [第 3 章](../docs/03-team-conventions.md)。

```bash
mvn verify                                               # 编译 + 全部测试（H2 + Stream Test Binder，无需外部中间件）
mvn -Prabbit -DskipTests package                         # 以 RabbitMQ Binder 打包（默认 Kafka）
docker compose -f deploy/local/docker-compose.yml up -d  # 本地中间件
```

| 目录 | 内容 |
| --- | --- |
| `acme-framework/` | 平台 starter：base / web / feign / mybatis / mq / security / test + common-core |
| `acme-gateway/` | Spring Cloud Gateway：路由、JWT、限流、熔断、灰度打标、屏蔽 /internal |
| `acme-auth/` | Spring Authorization Server |
| `acme-services/` | user-service、order-service、notification-service（每个服务 = api + server） |
| `deploy/` | Dockerfile、docker-compose、Helm Chart、GitOps values、K8s 平台清单、Nacos 配置样例、CI 模板 |
