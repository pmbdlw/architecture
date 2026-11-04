# acme-go 初始工程（Go 技术栈）

Go 1.26 · Kratos v2.9 · gRPC + Protobuf · Nacos（nacos-sdk-go v2）· GORM + goose · Watermill · OpenTelemetry

运行在 Kubernetes 上的中大型 Go 微服务系统初始工程：数据库 MySQL / PostgreSQL 二选一，消息 Kafka / RabbitMQ 二选一，包含平台库、API 网关、认证中心、三个示例业务服务与完整的部署清单。
完整说明见 [第 4 章 初始代码结构](../docs-go/04-code-structure.md)，团队规范见 [第 3 章](../docs-go/03-team-conventions.md)。

```bash
make test                                                # 单元 + 服务级集成测试（SQLite + 进程内消息，无需外部中间件）
make build                                               # 构建 5 个服务到 bin/
make init && make api                                    # 修改 proto 后重新生成代码
docker compose -f deploy/local/docker-compose.yml up -d  # 本地中间件
```

| 目录 | 内容 |
| --- | --- |
| `api/` | Protobuf 契约（对外 HTTP + 集群内 gRPC + 领域事件），生成代码一并提交 |
| `pkg/` | 平台公共库：conf / registry / selector/gray / server / client / auth / database / mq / errorx / result / observability / testkit / archtest |
| `app/gateway` | API 网关：路由、JWT、Redis 限流、熔断、灰度打标、屏蔽 /internal |
| `app/auth` | 认证中心：OAuth 2.1 授权码 + PKCE、客户端凭证、JWKS |
| `app/user` `app/order` `app/notification` | 业务服务（Kratos 布局：service / biz / data / server） |
| `deploy/` | Dockerfile、docker-compose、Helm Chart、GitOps values、K8s 平台清单、Nacos 配置样例、CI 模板 |
