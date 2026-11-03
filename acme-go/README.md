# acme-go 初始工程（Go 技术栈）

Go 1.26 · Kratos v2.9 · gRPC + Protobuf · Nacos（nacos-sdk-go v2）· GORM + goose · Watermill · OpenTelemetry

与 [Spring Cloud 版](../acme-cloud) 实现同一套架构：同样的环境、同样的部署流程、同样的接口约定（统一响应体、错误码、JWT 声明、事件头），两套服务可以混合部署在同一个 Nacos / Kafka / 网关体系中。
完整说明见 [Go 版第 4 章 初始代码结构](../docs-go/04-code-structure.md)，团队规范见 [Go 版第 3 章](../docs-go/03-team-conventions.md)。

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
