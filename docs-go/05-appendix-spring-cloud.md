# 附录 与 Spring Cloud 版对照

> 本附录面向同时维护两种技术栈、或从 Spring Cloud 迁移到 Go 的团队。第 1~4 章不依赖本附录，可以单独阅读。
> Spring Cloud 版文档见 [docs/](../docs/01-runtime-architecture.md)，初始工程见 [acme-cloud/](../acme-cloud)。

## A.1 能力对照

| 能力 | Spring Cloud 版 | Go 版 |
| --- | --- | --- |
| 注册发现 | Spring Cloud Alibaba Nacos Discovery | Kratos registry（Nacos，nacos-sdk-go v2） |
| 配置中心 | Nacos Config（`spring.config.import`） | `pkg/conf` 分层加载 + Nacos 配置源 |
| 平台默认配置 | `EnvironmentPostProcessor` + `defaults/*.yml` | 内嵌 `pkg/conf/defaults.yaml` |
| API 网关 | Spring Cloud Gateway | Go 网关（`httputil.ReverseProxy`） |
| 服务调用 | OpenFeign（HTTP `/internal/**`） | gRPC（`*InternalService`） |
| 负载均衡 + 灰度 | Spring Cloud LoadBalancer + 灰度 Supplier | Kratos 选择器 + 灰度 `NodeFilter` |
| 熔断 | Spring Cloud CircuitBreaker（Resilience4j） | Kratos SRE 熔断（服务间）+ gobreaker（网关） |
| 消息驱动 | Spring Cloud Stream（Kafka / RabbitMQ Binder） | Watermill（Kafka / RabbitMQ） |
| 事务发件箱 / 消费幂等 | `OutboxPublisher` + `OutboxRelay` / `IdempotentConsumer` | `mq.Outbox` + `mq.Relay` / `mq.Idempotent` |
| 认证中心 | Spring Authorization Server | go-oauth2（生产推荐 Casdoor / Keycloak / IDaaS） |
| 资源服务器 | Spring Security OAuth2 Resource Server | Kratos JWT 中间件 + JWKS |
| 统一响应 / 错误码 | `Result` + `GlobalExceptionHandler` | `pkg/result` + `pkg/errorx` |
| 数据访问 / 迁移 | MyBatis-Plus + Flyway | GORM + goose |
| 指标 / 链路 / 日志 | Micrometer + OTel + 结构化日志 | OpenTelemetry + Prometheus + `log/slog` |
| 健康检查 | Actuator（`/actuator/health/*`） | 管理端口 `/healthz`、`/readyz` |
| 分层约束 | ArchUnit | `pkg/archtest` |
| 测试支撑 | `@AcmeServiceTest`（H2 + Stream Test Binder） | `pkg/testkit`（SQLite + gochannel + 本地 JWT） |
| 服务契约模块 | `xxx-api`（Feign 接口 + DTO） | `api/`（proto + 生成代码） |
| 平台公共库 | `acme-framework`（starter） | `pkg/` |

## A.2 运行差异

| 项目 | Spring Cloud 版 | Go 版 |
| --- | --- | --- |
| 镜像 | JRE + 分层 jar，约 250MB | 静态二进制 + distroless，约 20MB |
| 启动时间 | 10~30 秒 | 1 秒以内（不含数据库迁移） |
| 每副本内存 | 1~2GiB（堆 + 元空间） | 128~512MiB |
| 内存上限 | `-XX:MaxRAMPercentage=75` | `GOMEMLIMIT` = 内存上限 × 90% |
| 端口 | HTTP 8080、管理 8081 | HTTP 8080、gRPC 9090、管理 8081 |
| 消息中间件选择 | 构建时 Maven profile 决定 Binder | 运行时 `MQ_DRIVER` 决定 |
| 数据库切换 | 修改连接串 | 修改 `DB_DRIVER` 与 `DB_DSN` |

## A.3 混合部署注意事项

两套实现遵循同一套约定，可以在同一套环境中混合部署（例如存量 Java 服务 + 新增 Go 服务）：

| 约定 | 是否一致 | 说明 |
| --- | --- | --- |
| Nacos 命名空间 / 分组 | 一致 | 两边注册到同一个 Nacos |
| 服务名 | **不同** | Java 注册为 `order-service`，Go 注册为 `order-service.http` / `.grpc`。网关需要按实现类型配置路由；Java 服务调用 Go 服务时走网关，或在 Go 服务额外暴露 HTTP 内部接口 |
| 内部调用协议 | **不同** | Java 用 HTTP `/internal/**`，Go 用 gRPC |
| 统一响应体 / 错误码 | 一致 | `{code, message, data, traceId}`，A/B/C 错误码与号段规则相同 |
| JWT 声明 | 一致 | `uid`、`roles`、`scope`，两边可共用同一个认证中心 |
| 灰度 | 一致 | 实例元数据 `gray-tag` + 请求头 `x-gray-tag` |
| 事件头 / Topic | 一致 | `acme_event_*`、`acme.<领域>.events` |
| 事件载荷 | **注意** | Go 的 protojson 把 int64 编码为字符串（Jackson 可直接读取）；金额字段 Go 版为字符串、Java 版为数值。混合部署时应在事件契约中统一金额为字符串，或用 proto 定义跨语言共享的事件契约 |
| RabbitMQ 路由 | **不同** | Spring Cloud Stream 以事件类型作为 routing key，消费者只绑定关心的类型；Go 版每个 topic 一个 fanout 交换机，消费方按事件类型头过滤。混用时两边需约定同一种交换机类型 |

## A.4 从 Spring Cloud 迁移到 Go 的建议

1. 先迁移边缘、无状态、对延迟敏感的服务（如通知、查询类服务），核心交易服务最后迁移；
2. 按服务整体切换：Java 与 Go 实现在 Nacos 中的服务名不同，网关灰度无法在两者之间分流，切换期间在入口层（Ingress / ALB 按权重路由，或网关路由指向新服务）按比例把流量从 Java 服务切到 Go 服务，稳定后下线 Java 实现；
3. 数据库表结构与迁移历史保持不变：goose 脚本从当前表结构开始编写基线，不重放 Flyway 历史；
4. 先统一事件契约（金额、时间格式）再迁移消息生产方，避免消费方同时兼容两种载荷。
