# 第 4 章 初始代码结构（Go）

> 初始工程位于仓库的 [`acme-go/`](../acme-go) 目录：一个 Go module，包含平台库、网关、认证中心和三个示例业务服务，以及完整的部署清单。覆盖前三章提到的全部技术：
> Nacos 注册 / 配置、Go 网关、gRPC + Protobuf + 灰度选择器、熔断、Watermill（Kafka / RabbitMQ）+ 事务发件箱、OAuth 2.1 认证中心 + JWT 资源服务、GORM + goose（MySQL / PostgreSQL）、OpenTelemetry + Prometheus + slog、Helm + Argo CD。
> `make test`（`go test -race ./...`）包含 19 个测试函数（单元、服务级集成、分层规则），`golangci-lint` 与 `buf lint` 均为 0 问题。

## 4.1 模块全景

![代码模块结构与依赖关系](images/code-modules.svg)

## 4.2 目录树

```text
acme-go/
├── go.mod                                   # 单一 module：github.com/acme/acme-go（Go 1.26）
├── Makefile                                 # init / api / lint / test / build / docker
├── buf.yaml · buf.gen.yaml                  # proto 检查与代码生成（protoc-gen-go / -go-grpc / Kratos -go-http）
├── .golangci.yml                            # 静态检查规则
│
├── api/                                     # ① 服务契约：对外 HTTP、集群内 gRPC、领域事件
│   ├── user/v1/user.proto                   #   UserService（对外 HTTP）+ UserInternalService（仅 gRPC）
│   ├── order/v1/order.proto · events.proto  #   OrderService + OrderCreated 事件；events.go 定义 Topic / 事件类型常量
│   └── notification/v1/notification.proto
│       └── *.pb.go · *_grpc.pb.go · *_http.pb.go   # 生成代码（提交到仓库）
├── third_party/google/api/                  # google.api.http 注解定义
│
├── pkg/                                     # ② 平台库（平台组维护，业务服务只依赖这里）
│   ├── conf/                                #   Bootstrap 配置结构、defaults.yaml、分层加载、Nacos v2 配置源
│   ├── registry/                            #   Nacos 注册中心（<服务>.http / .grpc）与实例元数据
│   ├── selector/gray/                       #   灰度实例筛选（网关与 gRPC 客户端共用）
│   ├── server/                              #   中间件链、HTTP / gRPC 服务器、管理端口、App 生命周期、Cleanup
│   ├── client/                              #   gRPC 客户端：发现 + 灰度 + 超时 + 熔断 + 元数据透传 + 错误还原
│   ├── auth/                                #   JWT 资源服务器：Claims（uid / roles / scope）、UID、RequireRole
│   ├── database/                            #   GORM 打开连接、goose 迁移、事务上下文、Model 公共字段、Sonyflake ID
│   ├── mq/                                  #   Watermill 驱动选择、Outbox / Relay、Idempotent、Router（重试 + 死信）
│   ├── errorx/ · result/                    #   错误码与统一响应体
│   ├── observability/                       #   slog 日志、OTel 链路、Prometheus 指标
│   ├── testkit/                             #   SQLite 测试库、JWT 签名器、内存注册中心、HTTP 断言
│   └── archtest/                            #   分层依赖检查
│
├── app/                                     # ③ 可部署单元（每个目录一个镜像）
│   ├── gateway/                             #   API 网关：internal/gateway/{gateway,balancer}.go
│   ├── auth/                                #   认证中心：internal/authserver（go-oauth2）
│   ├── user/                                #   用户服务
│   │   ├── cmd/user/main.go
│   │   ├── configs/config.yaml
│   │   ├── migrations/{mysql,postgres,sqlite}/00001_init_user.sql
│   │   ├── internal/{conf,service,biz,data,server}/
│   │   └── user_test.go
│   ├── order/                               #   订单服务：gRPC 调用 user → 事务内订单 + 发件箱 → Relay 投递
│   └── notification/                        #   通知服务：Watermill 消费 + 幂等 + 死信；分页查询通知
│
└── deploy/                                  # ④ 交付与运行
    ├── docker/Dockerfile                    #   多阶段构建 → distroless 非 root（ARG APP=order）
    ├── local/docker-compose.yml             #   本地中间件：MySQL、Nacos、Kafka、Redis、Jaeger（+ PostgreSQL / RabbitMQ）
    ├── helm/acme-service/                   #   通用 Chart：gRPC 端口、/healthz /readyz、GOMEMLIMIT
    ├── gitops/{argocd,envs/{test,staging,prod}}/   # ApplicationSet + 各环境 values（含生产灰度示例）
    ├── k8s/platform/                        #   命名空间、platform-env（自建 / 阿里云）、ExternalSecret、KafkaTopic
    ├── nacos/{test,staging,prod}/           #   acme-common.yaml、acme-gateway.yaml、order-service.yaml
    └── ci/gitlab-ci.yml                     #   verify → image → test → staging → prod
```

## 4.3 平台库一览

| 包 | 提供 | 使用方 |
| --- | --- | --- |
| `pkg/conf` | `Bootstrap`、`Load`（默认值 → 服务文件 → Nacos → 环境变量）、`Duration` | 所有服务 |
| `pkg/registry` | `New(conf.Nacos)` 返回 Registrar + Discovery；`Metadata` | 业务服务、网关 |
| `pkg/selector/gray` | `Select`、`Filter()`（Kratos NodeFilter）、`TagFromContext` | 网关、`pkg/client` |
| `pkg/server` | `Middlewares`（恢复 → 链路 → TraceId → 指标 → 访问日志 → 元数据 → 鉴权 → 错误规整）、`NewHTTPServer`、`NewGRPCServer`、`NewManagement`、`NewHandlerServer`、`NewApp`、`Cleanup` | 所有服务 |
| `pkg/client` | `Dial`、`Middlewares`（元数据 → 错误还原 → 熔断） | 调用下游的服务 |
| `pkg/auth` | `Claims`、`Server`（JWT 中间件）、`RemoteKeyFunc`、`UID`、`HasRole`、`RequireRole`、`HasScope` | 业务服务、网关、认证中心 |
| `pkg/database` | `Open`、`Migrate`、`Transaction.InTx`、`DB(ctx)`、`Model`、`NextID`、`Ping` | 有数据库的服务 |
| `pkg/mq` | `NewPublisher`、`NewSubscriber`、`Outbox`、`Relay`、`Idempotent`、`NewRouter`、事件头常量 | 发布 / 消费事件的服务 |
| `pkg/errorx` · `pkg/result` | 错误码定义与规整；Kratos 响应 / 错误编码器 | 所有服务 |
| `pkg/observability` | `Setup`（日志、链路、指标）、`NewLogger` | 所有服务 |
| `pkg/testkit` · `pkg/archtest` | 测试工具与分层检查 | 测试 |

## 4.4 关键代码导读

### 4.4.1 一个服务的 main 只有组装

```go
func run() error {
	var c conf.Config
	cfg, err := pconf.Load(*configPath, &c, func() *pconf.Bootstrap { return &c.Bootstrap })
	if err != nil {
		return err
	}
	var cleanup pserver.Cleanup
	defer cleanup.Run()
	cleanup.Add(cfg.Close)

	tel, _ := observability.Setup(ctx, &c.Bootstrap)                // 日志 · 链路 · 指标
	db, _ := database.Open(c.Data.Database)                         // DB_DRIVER + DB_DSN
	_ = database.Migrate(ctx, db, c.Data.Database.Driver, migrations.FS)
	keyFunc, _ := auth.RemoteKeyFunc(ctx, c.Auth.JWKSURL)           // JWKS
	publisher, _ := mq.NewPublisher(c.MQ)                           // MQ_DRIVER
	reg, _ := preg.New(c.Nacos)
	conn, _ := client.Dial(ctx, reg, c.Client, "user-service")      // gRPC 下游

	hs, relay := server.New(server.Deps{Conf: &c, Telemetry: tel, DB: db, KeyFunc: keyFunc,
		Users: userv1.NewUserInternalServiceClient(conn), Publisher: publisher})
	mgmt := pserver.NewManagement(c.Server.Management.Addr, map[string]pserver.Check{"db": database.Ping(db)})
	return pserver.NewApp(&c.Bootstrap, tel.Logger, reg, hs, relay, mgmt).Run() // 注册 Nacos · 等待 SIGTERM
}
```
（节选自 `app/order/cmd/order/main.go`，省略了错误处理）

### 4.4.2 组装根：data → biz → service

```go
// app/order/internal/server/server.go
func New(d Deps) (*http.Server, *mq.Relay) {
	uc := biz.NewOrderUsecase(
		data.NewOrderRepo(d.DB),
		data.NewBuyerGateway(d.Users),                                // 防腐层：gRPC 调 user-service
		database.NewTransaction(d.DB),
		data.NewEventPublisher(mq.NewOutbox(d.DB, d.Conf.App.Name)),  // 事件写发件箱
	)
	mws := pserver.Middlewares(d.Telemetry, auth.Server(d.KeyFunc, d.Conf.Auth.Issuer))
	hs := pserver.NewHTTPServer(d.Conf.Server.HTTP, d.Telemetry.Logger, mws)
	orderv1.RegisterOrderServiceHTTPServer(hs, service.NewOrderService(uc))
	return hs, mq.NewRelay(d.DB, d.Publisher, d.Conf.MQ.Outbox, d.Telemetry.Logger)
}
```

### 4.4.3 用例：远程调用在事务外，订单与事件同事务

```go
// app/order/internal/biz/order.go
func (uc *OrderUsecase) Create(ctx context.Context, cmd CreateOrderCommand) (*Order, error) {
	ok, err := uc.buyers.CanPlaceOrder(ctx, cmd.UserID)   // ① gRPC，在事务外
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrBuyerNotAvailable.New()
	}
	o := NewOrder(uc.repo.NextID(), cmd.UserID, cmd.ProductCode, cmd.Quantity, cmd.Amount, uc.now())
	err = uc.tx.InTx(ctx, func(ctx context.Context) error { // ② 本地事务
		if err := uc.repo.Add(ctx, o); err != nil {
			return err
		}
		return uc.events.OrderCreated(ctx, o)              // ③ 写发件箱
	})
	if err != nil {
		return nil, err
	}
	return o, nil
}
```

### 4.4.4 防腐层：把下游错误翻译成本域语义

```go
// app/order/internal/data/order.go
func (g *buyerGateway) CanPlaceOrder(ctx context.Context, userID int64) (bool, error) {
	u, err := g.users.GetUser(ctx, &userv1.GetUserRequest{Id: userID})
	if err != nil {
		se := errorx.Normalize(err)
		if se.Code < 500 {
			return false, nil   // 下游业务错误（如 A1001 用户不存在）→ 不可下单
		}
		return false, se        // 下游故障（C0001 / C0503）→ 原样返回
	}
	return u.Status == "ACTIVE", nil
}
```

### 4.4.5 灰度筛选（网关与 gRPC 共用一套规则）

```go
// pkg/selector/gray/gray.go
func Select[T any](nodes []T, tag string, meta func(T) map[string]string) []T {
	var stable, gray []T
	for _, n := range nodes {
		t := meta(n)[MetadataKey]
		switch {
		case t == "":
			stable = append(stable, n)
		case tag != "" && t == tag:
			gray = append(gray, n)
		}
	}
	if len(gray) > 0 {
		return gray       // 带标签的请求 → 同标签实例
	}
	if len(stable) == 0 {
		return nodes      // 全部为灰度实例时不过滤，可用性优先
	}
	return stable         // 普通请求 → 只走稳定实例
}
```

### 4.4.6 幂等消费

```go
// pkg/mq/consumer.go
func (i *Idempotent) Handler(fn func(ctx context.Context, msg *message.Message) error) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		id := msg.Metadata.Get(HeaderID)
		err := i.tx.InTx(msg.Context(), func(ctx context.Context) error {
			record := ConsumedEvent{Consumer: i.consumer, EventID: id, ConsumedAt: time.Now().UTC()}
			if err := database.DB(ctx, i.db).Create(&record).Error; err != nil {
				return err
			}
			return fn(ctx, msg)   // 业务写入与登记在同一事务
		})
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil            // 重复投递：确认并跳过
		}
		return err
	}
}
```

### 4.4.7 测试：用真实服务器 + 进程内依赖

```go
// app/order/order_test.go（节选）
hs, relay := server.New(server.Deps{Conf: c, Telemetry: testkit.Telemetry(t, "order-service"),
	DB: testkit.DB(t, migrations.FS),           // 内存 SQLite，执行真实迁移脚本
	KeyFunc: signer.KeyFunc,                    // 本地签发 JWT
	Users: &fakeUsers{info: &userv1.UserInfo{Id: 1001, Status: "ACTIVE"}}, // 假的 gRPC 客户端
	Publisher: gochannel.NewGoChannel(...)})    // 进程内消息总线

resp := testkit.Do(t, hs, "POST", "/api/v1/orders", signer.Token(t, "1001", "USER"),
	`{"productCode":"SKU-1","quantity":2,"amount":"99.80"}`)
// 断言：200、发件箱有 1 条待发送 → relay.RelayAll → 收到 order.created 消息 → 发件箱标记为已发送
```

| 测试 | 覆盖内容 |
| --- | --- |
| `pkg/conf` · `TestLoadPrecedence` | 默认值 → 服务文件 → 环境变量 → 覆盖 的优先级与类型转换 |
| `pkg/errorx` · `TestNormalize` | 业务错误、JWT 错误、熔断错误、未知错误的规整与信息隐藏 |
| `pkg/selector/gray` · `TestSelect` | 灰度实例选择的 4 种情况 |
| `pkg/mq` · `TestOutboxRelayAndIdempotentConsumer` · `TestKafkaProducerConfigIsValid` | 事务外发布被拒、投递与元数据、不重复投递、重复消息只处理一次；Kafka 幂等生产者配置合法 |
| `app/user`（3 个） | 管理员注册、手机号脱敏、int64 输出为字符串、TraceId、内部 gRPC 接口；409 / 404 / 400 / 403 / 401；分层规则 |
| `app/order`（3 个） | 下单 → 发件箱 → Relay → 消息；越权访问 404；下游不存在 422；下游不可用 503；参数错误 400；无 uid 令牌 401；分层规则 |
| `app/notification`（2 个） | 重复消息只处理一次、忽略其它事件类型、毒消息重试后进入死信、分页查询与上限；分层规则 |
| `app/gateway`（4 个） | `/internal` 屏蔽、401、未知路由、灰度白名单、伪造灰度头无效、令牌与灰度头透传、无实例降级、熔断、限流与 Redis 故障放行、灰度规则 |
| `app/auth`（2 个） | OIDC 发现、客户端凭证令牌（JWKS 验签）、授权码 + PKCE 全流程、错误 verifier、错误密码 |

除测试外，还用真实进程做过冒烟：启动 auth、user、order 三个二进制，走 PKCE 取令牌 → 注册用户 → 跨服务 gRPC 下单（业务错误还原为 `422 A2002`）→ 杀掉 user 进程后下单（`503 C0503`）→ SIGTERM 优雅退出，并确认链路 ID 在 order 与 user 的日志中一致。

## 4.5 本地开发

### 4.5.1 启动中间件

```bash
cd acme-go
docker compose -f deploy/local/docker-compose.yml up -d     # MySQL + Nacos + Kafka + Redis + Jaeger
```

首次访问 Nacos 控制台 <http://localhost:18080> 时设置 `nacos` 用户密码为 `nacos`。

### 4.5.2 运行服务

每个服务在自己的目录下运行（配置文件路径为 `configs/config.yaml`）：

```bash
cd app/auth         && go run ./cmd/auth            # 9000（管理端口 7000）
cd app/user         && go run ./cmd/user            # HTTP 8101 · gRPC 9101 · 管理 7101
cd app/order        && go run ./cmd/order           # HTTP 8102 · gRPC 9102 · 管理 7102
cd app/notification && go run ./cmd/notification    # HTTP 8103 · gRPC 9103 · 管理 7103
cd app/gateway      && go run ./cmd/gateway         # 8080（需要 Nacos 发现服务）
```

不想启动 Nacos 时，可以设置 `NACOS_ENABLED=false`：业务服务不注册，order-service 按 `client.endpoints`（默认 `127.0.0.1:9101`）直连 user-service；此时直接访问各服务端口，而不是网关。

| 地址 | 说明 |
| --- | --- |
| <http://localhost:8080/api/v1/...> | 经网关访问业务接口 |
| <http://localhost:7102/metrics> | 订单服务指标 |
| <http://localhost:7102/readyz> | 订单服务就绪状态 |
| <http://localhost:16686> | Jaeger 链路查询 |

### 4.5.3 获取令牌

- **用户令牌**（授权码 + PKCE）：Postman / Insomnia 中选择 OAuth 2.0 → Authorization Code (With PKCE)，Client ID `acme-web`，授权地址 `http://127.0.0.1:9000/oauth2/authorize`，令牌地址 `http://127.0.0.1:9000/oauth2/token`，回调地址需与 `AUTH_WEB_REDIRECT_URI` 一致；演示账号 `alice / password`（uid 1001）、`admin / password`（ADMIN）。
- **客户端凭证**：

```bash
curl -u acme-internal:local-only-secret -d 'grant_type=client_credentials&scope=internal' \
     http://127.0.0.1:9000/oauth2/token
```

### 4.5.4 切换数据库与消息中间件

只改环境变量，**不需要重新构建**：

| 切换 | 做法 |
| --- | --- |
| MySQL → PostgreSQL | `DB_DRIVER=postgres`、`DB_DSN=postgres://user:pass@127.0.0.1:5432/acme_order?sslmode=disable`；goose 自动选择 `migrations/postgres` |
| Kafka → RabbitMQ | `MQ_DRIVER=rabbitmq`、`RABBITMQ_URL=amqp://guest:guest@127.0.0.1:5672/` |

## 4.6 构建与部署

```bash
make lint test                          # 静态检查 + 测试
make build                              # 5 个二进制到 bin/
docker build -f deploy/docker/Dockerfile --build-arg APP=order \
  -t registry.acme.com/acme-go/order-service:$(git rev-parse --short HEAD) .
helm template order-service deploy/helm/acme-service -f deploy/gitops/envs/prod/order-service.yaml
```

部署流程：CI 在 `main` 上构建镜像并扫描漏洞，然后更新 `deploy/gitops/envs/test/*.yaml` 中的镜像版本，Argo CD（`deploy/gitops/argocd/applicationsets.yaml`）自动同步到测试环境；测试通过后人工把同一镜像晋级到 staging；生产在变更审批后写入镜像 digest，由发布负责人在 Argo CD 中手动 Sync，可先发灰度（`envs/prod/order-service-gray.yaml`）。新集群需要准备的平台资源见 `deploy/k8s/platform/` 与 `deploy/nacos/<env>/`。

## 4.7 新增一个服务（以 `inventory-service` 为例）

```bash
cd acme-go
cp -r app/user app/inventory
mv app/inventory/cmd/user app/inventory/cmd/inventory
grep -rl 'user' app/inventory | xargs sed -i 's#app/user#app/inventory#g; s/user-service/inventory-service/g'
```

然后：

1. 在 `api/inventory/v1/inventory.proto` 定义对外 `InventoryService` 与集群内 `InventoryInternalService`，`make api` 生成代码；
2. 重写 `internal/biz`（领域与用例）、`internal/data`（仓储）、`internal/service`（实现 proto 接口），在 `internal/server` 中组装；
3. 修改 `configs/config.yaml`：服务名、本地端口（如 8104 / 9104 / 7104）、数据库 `acme_inventory`；分配错误码号段（如 `A4xxx`）；
4. 编写 `migrations/{mysql,postgres,sqlite}` 脚本；
5. 网关路由增加 `/api/v1/inventories → inventory-service`（Nacos `acme-gateway.yaml`）；
6. 新增 `deploy/gitops/envs/{test,staging,prod}/inventory-service.yaml` 与 ExternalSecret；`Makefile` 的 `APPS` 增加 `inventory`；
7. `make lint test` 通过（`TestLayers` 会立即约束新服务的分层）。

## 4.8 后续可扩展项

- protovalidate（proto 中声明校验规则）、uber/fx（服务多时的依赖注入）
- Testcontainers（MySQL / PostgreSQL / Kafka 真实环境集成测试）、govulncheck
- 生产级 IdP（Casdoor / Hydra / Keycloak / IDaaS）替换 `app/auth`
- 分布式调度（XXL-JOB Go 执行器 / SchedulerX）、缓存（go-redis + singleflight）
- DTM（Saga / TCC，仅限强一致场景）、MSE 服务治理
- 以 APISIX / Higress 替换或前置 Go 网关
