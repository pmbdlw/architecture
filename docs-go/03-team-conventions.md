# 第 3 章 Go 团队开发规范

> 与语言无关的规范（服务拆分、分支模型、错误码、数据库设计、消息、配置、安全、发布）与 [Spring Cloud 版第 3 章](../docs/03-team-conventions.md) 保持一致，本章只在需要时简述；**Go 特有的规范**（工程布局、编码、并发、Protobuf、GORM、测试、工具链）完整给出。
> 标记说明：🔒 由工程或 CI 强制；👀 代码评审检查。

## 3.1 服务拆分与命名

拆分原则同 Spring Cloud 版：按限界上下文拆分、一个服务独占一个数据库、宁粗勿细。

| 对象 | 规则 | 示例 |
| --- | --- | --- |
| 服务名（`app.name`、K8s、Nacos 前缀） | `<领域>-service` | `order-service`（Nacos 中为 `order-service.http` / `.grpc`） |
| 代码目录 | `app/<领域>` | `app/order` |
| 可执行入口 | `app/<领域>/cmd/<领域>/main.go` | `app/order/cmd/order` |
| Go 包名 | 小写单词，不用下划线和复数 | `biz`、`data`、`gray`、`orderv1` |
| Proto 包 | `<领域>.v<n>`，`go_package` 别名 `<领域>v<n>` | `order.v1` / `orderv1` |
| 对外 / 集群内服务 | `<名称>Service` / `<名称>InternalService` | `UserService`、`UserInternalService` |
| 数据库 / 表 | `acme_<领域>` / 单数小写下划线 | `acme_order` / `order_info` |
| Topic / 事件类型 / 消费组 / 死信 | 同 Spring Cloud 版 | `acme.order.events` / `order.created` / 服务名 / `<topic>.<消费组>.dlq` |
| 错误码号段 | 每个服务 1000 个 | user `A1xxx`、order `A2xxx` |
| 本地端口 | 服务 HTTP 81xx、gRPC 91xx、管理 71xx；网关 8080、认证 9000；K8s 中统一 8080 / 9090 / 8081 | order：8102 / 9102 / 7102 |

## 3.2 仓库与分支

- **单仓单模块（本工程默认）**：`acme-go` 是一个 Go module，所有服务共享 `go.mod`，依赖版本天然一致，跨服务重构一次提交完成；CI 按变更路径只构建受影响的服务。
- 服务超过 30 个或团队跨组织时，可把 `pkg/` 拆成独立模块发布（SemVer tag），业务仓库通过 `go get` 依赖固定版本；`api/` 契约可单独成仓并用 buf 管理。
- 分支模型、MR 规则、Conventional Commits、版本号规则同 Spring Cloud 版（主干开发 + 短分支，`main` 受保护，CI 通过才能合入 🔒）。

## 3.3 工程布局

```text
app/order/
├── cmd/order/main.go        # 入口：加载配置 → 连接中间件 → server.New → App.Run
├── configs/config.yaml      # 本服务配置（只写特有项）
├── migrations/              # goose 脚本：mysql/ postgres/ sqlite/（sqlite 仅测试）
├── internal/
│   ├── conf/                # 配置结构体（内嵌 pconf.Bootstrap）
│   ├── service/             # 接口层：实现 proto 接口、消息消费入口
│   ├── biz/                 # 业务层：领域模型、用例、仓储/网关/事务/事件接口、错误码
│   ├── data/                # 数据层：GORM 仓储、gRPC 下游适配、发件箱
│   └── server/              # 组装：data → biz → service → HTTP / gRPC 服务器
└── order_test.go            # 服务级集成测试 + 分层规则测试
```

- 服务代码全部放在 `internal/` 下，Go 编译器保证其它服务无法导入 🔒；服务间只能依赖 `api/`；
- 共用能力放 `pkg/`，由平台组维护，业务团队提 MR 评审后合入；
- 禁止出现 `common`、`util`、`helper` 这类含义模糊的包 👀。

## 3.4 分层规范

![Go 服务内部分层](images/service-layers.svg)

| 层 | 可以做 | 不可以做 |
| --- | --- | --- |
| service | 参数校验、`auth.UID(ctx)` 解析身份、proto ↔ 领域对象转换、调用用例 | 写业务规则、开启事务、导入 `internal/data` 或 `gorm.io` 🔒 |
| biz | 领域模型与规则、用例编排、定义仓储 / 网关 / 事务 / 事件接口、定义错误码 | 导入 `internal/data`、`internal/service`、`gorm.io`、`net/http`、`google.golang.org/grpc`、Watermill、Redis 🔒 |
| data | 实现 biz 中的接口：GORM 仓储、gRPC 下游适配（防腐层）、发件箱 | 导入 `internal/service`、`internal/server` 🔒；包含业务规则 👀 |
| server | 创建依赖、注册接口到服务器 | 写任何逻辑 👀 |

这些依赖规则由 `pkg/archtest.CheckLayers` 在每个服务的 `TestLayers` 中检查，违反即测试失败。

**接口定义在使用方**：`OrderRepo`、`BuyerGateway`、`Transaction`、`EventPublisher` 都定义在 `biz`（使用方），由 `data` 实现，这是 Go 中实现依赖倒置的惯用法。

## 3.5 API 与 Protobuf 规范

| # | 规则 |
| --- | --- |
| 1 | 所有接口先写 proto，再生成代码；生成代码与 proto 一起提交，CI 检查二者一致 🔒 |
| 2 | `buf lint`（STANDARD 规则）🔒；`buf breaking` 与 `main` 比较，禁止删除字段、修改字段编号或类型 🔒 |
| 3 | 字段名在 proto 中用 `snake_case`，JSON 输出为 `lowerCamelCase`（protojson 默认）|
| 4 | 对外接口用 `google.api.http` 注解，前缀 `/api/v<n>/`，资源复数名词；集群内接口只定义为 `*InternalService`，不加 HTTP 注解，因此网关不可达 🔒 |
| 5 | int64 在 JSON 中输出为字符串（protojson 默认，避免 JS 精度丢失）；金额用字符串表示的十进制数（服务内用 `shopspring/decimal`） |
| 6 | 时间用 `google.protobuf.Timestamp`（JSON 为 RFC 3339） |
| 7 | 分页请求 `page_no`（从 1 开始）、`page_size`（≤100）；响应 `total`、`page_no`、`page_size`、`records` |
| 8 | 统一响应体 `{code, message, data, traceId}` 与错误码规则同 Spring Cloud 版，由 `pkg/result` 编码 🔒；成功统一 HTTP 200（Kratos 约定），失败使用与错误码一致的 HTTP 状态 |
| 9 | 业务失败返回 `biz.ErrXxx.New(args...)` / `errorx.BadRequest.Msg("...")`，不要返回 `fmt.Errorf` 字符串错误给调用方 👀；未知错误统一 `B0001`，原始错误只记录在日志中 🔒 |
| 10 | 字段只增不删；废弃字段标注 `reserved`，不复用编号 |

## 3.6 Go 编码规范

| # | 规则 |
| --- | --- |
| 1 | `gofmt` / `goimports` 格式化 🔒；`go vet` 与 `golangci-lint`（`.golangci.yml`）通过 🔒（CI） |
| 2 | `context.Context` 作为第一个参数并逐层传递，不保存到结构体，不传 `nil` |
| 3 | 错误必须处理；包装时用 `fmt.Errorf("...: %w", err)` 保留错误链，判断用 `errors.Is / As` |
| 4 | 不在库代码中 `panic`；服务入口与中间件有 `recovery` 兜底 |
| 5 | 不使用全局可变状态与 `init()` 副作用；依赖通过构造函数显式传入（组装集中在 `internal/server`） |
| 6 | 导出的标识符写注释；注释说明“为什么”，而不是复述代码 |
| 7 | 接口小而专注（1~3 个方法），定义在使用方；返回具体类型，接收接口 |
| 8 | 日志用 `log.NewHelper(logger).WithContext(ctx)`，自动带 `trace_id`；键值对形式 `Infow("msg", "...", "order_id", id)` |
| 9 | 不打印密码、令牌、完整手机号 / 身份证 / 银行卡 👀 |
| 10 | 资源（`rows`、`resp.Body`、文件）用 `defer` 关闭 🔒（`bodyclose`、`sqlclosecheck`） |

## 3.7 并发规范

| # | 规则 |
| --- | --- |
| 1 | 每个 goroutine 都要有明确的退出条件（`ctx.Done()` 或通道关闭），不允许“发射后不管” 👀 |
| 2 | 长期运行的后台任务（发件箱投递、消费者、定时任务）实现 `transport.Server`，交给 Kratos App 管理启停（参考 `mq.Relay`、`mq.Router`）|
| 3 | 并发处理一批任务用 `errgroup` 并设置并发上限（`SetLimit`），禁止无界创建 goroutine |
| 4 | 共享数据优先用通道或 `sync/atomic`（如网关的 `atomic.Pointer` 配置热更新）；必须加锁时锁的范围尽量小，不在持锁时做 I/O |
| 5 | 测试统一带 `-race` 运行 🔒（CI） |

## 3.8 服务间调用规范

| # | 规则 |
| --- | --- |
| 1 | 只依赖对方 `api/` 中的 gRPC 客户端，禁止复制对方的结构体或手写 HTTP 调用 👀 |
| 2 | 连接通过 `pkg/client.Dial` 创建（发现、灰度、超时、熔断、元数据透传、错误还原）🔒 |
| 3 | 下游调用放在数据库事务之外 👀（参考 `OrderUsecase.Create`） |
| 4 | 在 `data` 层做防腐适配，把下游错误翻译为本服务语义（参考 `buyerGateway`）👀 |
| 5 | 不在循环中逐条调用，提供批量接口；同步调用链深度 ≤ 3，更长的流程改为事件驱动 👀 |
| 6 | 一个服务进程内对同一下游只建一个 `*grpc.ClientConn`，在启动时创建、退出时关闭 |

## 3.9 数据库规范

数据库设计规范（一服务一库、公共字段、雪花 / Sonyflake 主键、索引、禁止外键与跨库 JOIN、大表 Online DDL、双数据库兼容写法）同 Spring Cloud 版。Go 特有部分：

| # | 规则 |
| --- | --- |
| 1 | PO 结构体内嵌 `database.Model`（`id`、`created_at`、`updated_at`、`deleted` 逻辑删除、`version` 乐观锁）🔒 |
| 2 | 仓储通过 `database.DB(ctx, r.db)` 取连接，自动加入当前事务；事务只在用例层用 `Transaction.InTx` 开启 👀 |
| 3 | **禁止在生产使用 GORM `AutoMigrate`**；表结构只由 goose 脚本管理 🔒（工程中没有调用） |
| 4 | 迁移脚本放在 `migrations/{mysql,postgres,sqlite}/`，文件名 `00001_<描述>.sql`，含 `-- +goose Up / Down`；已在共享环境执行过的脚本禁止修改 |
| 5 | `sqlite/` 目录只用于测试，结构须与 mysql / postgres 保持一致 👀 |
| 6 | 主键 ID 在应用侧生成（`database.NextID()`），便于在事务前拿到 ID、写入事件 |
| 7 | 唯一约束冲突用 `errors.Is(err, gorm.ErrDuplicatedKey)` 判断（已开启 `TranslateError`） |
| 8 | 查询必须带条件与分页上限；原生 SQL 只用参数绑定，禁止拼接用户输入 |

## 3.10 消息规范

与 Spring Cloud 版相同的规则：发布必须走发件箱（`mq.Outbox.Publish` 在事务外调用会直接返回错误 🔒）、消费必须幂等（`mq.Idempotent`）、事件是过去式事实、契约只做向后兼容修改、消息头使用 `acme_event_*`（不要用 `ce_` 前缀）、重试耗尽进入死信并告警、Topic 由平台创建。

Go 特有：事件契约用 proto 定义在 `api/<领域>/v1/events.proto`，载荷用 protojson 编码；消费方用 `protojson.UnmarshalOptions{DiscardUnknown: true}` 容忍新增字段。

## 3.11 配置规范

| 配置 | 位置 |
| --- | --- |
| 平台默认值 | `pkg/conf/defaults.yaml`（平台组维护） |
| 服务固定配置 | `app/<领域>/configs/config.yaml`，不出现任何环境相关值 👀 |
| 部署形态 | `deploy/gitops/envs/<env>/<服务>.yaml` |
| 动态业务配置 | Nacos `<服务名>.yaml`；需要热更新的段用 `cfg.Watch` 订阅 |
| 凭据 | Vault / KMS → ExternalSecret → Secret → 环境变量（如 `DB_DSN`）|
| 基础设施地址 | platform-env ConfigMap（`NACOS_SERVER_ADDR`、`DB_DRIVER`、`MQ_DRIVER`、`KAFKA_BROKERS`……） |

配置项通过 `${VAR:默认值}` 占位符引用环境变量；不按环境拆分配置文件，镜像在所有环境中一致。

## 3.12 测试规范

| 层次 | 做法 |
| --- | --- |
| 单元测试 | 领域对象、纯函数；表驱动测试（`[]struct{...}` + 循环） |
| 服务级集成测试 | `server.New(Deps{...})` 构造完整服务，HTTP 请求直接调用 `ServeHTTP`；数据库用 `testkit.DB`（内存 SQLite + 真实迁移脚本），消息用 gochannel，下游 gRPC 用实现了客户端接口的假对象，JWT 用 `testkit.Signer` 签发 |
| 分层规则 | 每个服务一个 `TestLayers` 🔒 |
| 真实中间件 | 需要验证 MySQL / PostgreSQL 方言、`SKIP LOCKED`、Kafka 语义时使用 Testcontainers（CI 提供 Docker） |
| 运行方式 | `go test -race -count=1 ./...` 🔒 |

示例：`order_test.go` 中 `TestDownstreamFailures` 用假的 `UserInternalServiceClient` 模拟“用户不存在”和“下游不可用”，断言分别得到 `422 A2002` 与 `503 C0503`，且没有写入发件箱。

## 3.13 部署与运行规范

| # | 规则 |
| --- | --- |
| 1 | 所有服务使用 `deploy/helm/acme-service` 通用 Chart，只在 `envs/<env>/<服务>.yaml` 覆盖差异 🔒 |
| 2 | 镜像为静态二进制 + distroless 非 root，只读根文件系统 🔒 |
| 3 | 内存 request = limit；`GOMEMLIMIT` 由 Chart 按上限的 90% 自动设置 🔒；CPU 只设 request |
| 4 | 探针：startup / liveness 用 `/healthz`，readiness 用 `/readyz`，均在管理端口 8081 🔒 |
| 5 | 优雅停机：`preStop` 10s + 应用停止 ≤30s < `terminationGracePeriodSeconds` 60s 🔒 |
| 6 | 生产 ≥3 副本、HPA + PDB、跨可用区打散，按 digest 部署 |

## 3.14 工具链

| 工具 | 用途 | 命令 |
| --- | --- | --- |
| buf | proto 检查、生成、兼容性 | `make api`、`buf breaking` |
| golangci-lint | 静态检查（vet、gosec、errorlint、bodyclose……） | `make lint` |
| go test -race | 测试与竞态检测 | `make test` |
| Trivy | 镜像漏洞扫描（CI 阻断严重漏洞）🔒 | 见 `deploy/ci/gitlab-ci.yml` |
| govulncheck | 依赖漏洞扫描（建议加入 CI） | `govulncheck ./...` |

## 3.15 代码评审清单

- [ ] 分层正确：service 无业务规则，biz 无技术框架依赖，data 实现 biz 接口
- [ ] 事务边界在用例；事务内没有 gRPC 调用与耗时操作
- [ ] 跨服务只经 `api/` 与 `pkg/client`；有防腐适配；没有循环依赖
- [ ] 发布事件走 `mq.Outbox`；消费幂等；proto 契约向后兼容（`buf breaking` 通过）
- [ ] 错误使用错误码；`%w` 保留错误链；没有吞掉错误
- [ ] 用户身份来自 `auth.UID(ctx)`；管理接口有 `RequireRole`；敏感字段脱敏
- [ ] 每个 goroutine 有退出路径；共享状态无数据竞争（`-race` 通过）
- [ ] 迁移脚本三种方言同步，向前兼容
- [ ] 日志带上下文、级别恰当、无敏感信息
- [ ] 有对应测试；`make lint test` 通过

## 3.16 新服务接入流程

流程同 Spring Cloud 版（立项评审 → 创建代码 → 申请资源 → 配置部署 → 可观测 → 上线检查），创建代码的具体步骤见第 4 章 4.7。
