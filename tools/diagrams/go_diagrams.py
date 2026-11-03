"""Go 技术栈专属的图：组件运行时分布、服务内分层（Kratos 布局）、代码模块依赖。"""
from svgkit import Svg, C, wrap
from springcloud_diagrams import badge, stack

W = 1440


def landscape():
    s = Svg(W, 960, "Go 技术栈组件运行时分布")
    s.title_block("Go 技术栈各组件在运行时的位置（以“下单”请求为例）",
                  "①请求进入  ②网关鉴权·限流·灰度  ③发现并转发到服务  ④gRPC 同步调用  ⑤本地事务（订单+发件箱）  "
                  "⑥Watermill 投递事件  ⑦异步消费")

    s.box(300, 92, 1120, 62, "Nacos（自建集群 / 阿里云 MSE）· nacos-sdk-go v2",
          ["Kratos registry：注册 <服务>.http / <服务>.grpc（元数据含 gray-tag / version）   ·   配置源：acme-common.yaml + "
           "<服务>.yaml，修改后 Watch 热更新"], kind="cloud", icon="compass", title_size=15)
    for xx in (470, 870, 1265):
        s.arrow([(xx, 154), (xx, 192)], color=C["cloud"][0], dash="5 4", both=True)

    s.box(20, 300, 120, 90, "客户端", ["Web / App", "携带 JWT"], kind="user")
    s.box(165, 300, 120, 90, "Ingress", ["nginx / ALB", "TLS 终止"], kind="access")
    s.arrow([(140, 345), (163, 345)], color=C["access"][0])
    badge(s, 152, 322, 1)
    s.arrow([(285, 345), (308, 345)], color=C["access"][0])

    s.group(310, 200, 320, 360, "acme-gateway（Go）", kind="cloud", dash=None)
    stack(s, 326, 222, 288, [
        ("net/http + httputil.ReverseProxy", "路由（前缀）· 配置热更新"),
        ("golang-jwt + keyfunc", "JWKS 验签 · 校验 iss"),
        ("redis_rate（GCRA）", "Redis 令牌桶 · 故障放行"),
        ("gobreaker", "路由级熔断 · 503 降级"),
        ("灰度选择器 pkg/selector/gray", "按 gray-tag 选实例"),
        ("otelhttp", "生成链路 · 透传 traceparent"),
    ], kind="cloud")
    badge(s, 540, 210, 2, C["cloud"][0])

    s.group(680, 200, 380, 360, "order-service（Kratos）", kind="svc", dash=None)
    stack(s, 696, 222, 348, [
        ("Kratos HTTP + protojson", "统一 Result · 错误码 · int64→字符串"),
        ("JWT 中间件 pkg/auth", "uid / roles · RequireRole"),
        ("gRPC 客户端 pkg/client", "发现 · 灰度 · 超时 · 熔断 · 错误还原"),
        ("GORM + goose", "MySQL / PostgreSQL · 版本化迁移"),
        ("Watermill + Outbox Relay", "事务内写事件 · 异步投递"),
        ("OpenTelemetry + slog", "指标 · 链路 · JSON 日志"),
    ], kind="svc")
    s.arrow([(630, 300), (678, 300)], color=C["cloud"][0])
    badge(s, 654, 282, 3, C["cloud"][0])

    s.group(1110, 200, 310, 170, "user-service（Kratos）", kind="svc", dash=None)
    stack(s, 1126, 222, 278, [("gRPC UserInternalService", "仅注册在 gRPC 服务器"),
                              ("GORM", "user_account 表")], kind="svc")
    s.arrow([(1060, 330), (1108, 330)], color=C["svc"][0], label="gRPC", label_pos=(1084, 314))
    badge(s, 1084, 346, 4, C["svc"][0])

    s.box(1110, 400, 310, 70, "MySQL / PostgreSQL", ["每个服务独立库，禁止跨库访问"], kind="data", icon="db")
    s.arrow([(1060, 440), (1108, 440)], color=C["data"][0])
    badge(s, 1084, 424, 5, C["data"][0])

    s.box(680, 618, 380, 74, "Kafka / RabbitMQ", ["topic / exchange：acme.order.events",
                                                    "Kafka key = 订单 ID（保证同订单有序）"], kind="mq", icon="mq")
    s.arrow([(870, 560), (870, 616)], color=C["mq"][0], label="Relay → Publisher", label_pos=(870, 588))
    badge(s, 990, 588, 6, C["mq"][0])

    s.group(1110, 600, 310, 120, "notification-service", kind="svc", dash=None)
    s.box(1126, 622, 278, 84, "Watermill Router 消费", ["消费组 = 服务名 · 重试 3 次", "死信 DLQ · consumed_event 幂等"],
          kind="svc", title_size=13.5, line_size=12)
    s.arrow([(1060, 655), (1108, 655)], color=C["mq"][0])
    badge(s, 1084, 638, 7, C["mq"][0])

    s.box(310, 618, 320, 74, "Redis / Tair", ["网关限流令牌桶（多副本共享）"], kind="mq", icon="bolt")
    s.arrow([(470, 560), (470, 616)], color=C["mq"][0], both=True)

    s.box(20, 600, 265, 120, "acme-auth（Go）", ["go-oauth2 · 授权码 + PKCE", "签发 RS256 JWT（uid / roles）",
                                                 "公开 /oauth2/jwks"], kind="cloud", icon="lock")
    s.arrow([(152, 600), (152, 520), (308, 520)], color=C["cloud"][0], dash="5 4", label="拉取 JWKS 验签",
            label_pos=(222, 520))

    s.group(20, 768, 1400, 172, "可观测：OpenTelemetry + Prometheus + slog（所有服务统一由 pkg/observability、pkg/server 提供）",
            kind="obs")
    obs = [("指标 Metrics", ["OTel Metrics → Prometheus 导出", "管理端口 /metrics → Prometheus / ARMS"], "chart"),
           ("链路 Tracing", ["Kratos tracing + otelhttp", "OTLP → Collector → Tempo / ARMS"], "chart"),
           ("日志 Logging", ["log/slog JSON，含 trace_id / span_id", "Fluent Bit → Loki / LoongCollector → SLS"], "doc"),
           ("健康 Health", ["/healthz 存活 · /readyz 就绪", "SIGTERM → 摘流量 → 优雅停机"], "gear")]
    bw = (1400 - 40 - 3 * 16) / 4
    for i, (t, lines, ic) in enumerate(obs):
        s.box(40 + i * (bw + 16), 796, bw, 120, t, lines, kind="obs", icon=ic, title_size=15, line_size=12.5)
    return s


def layers():
    s = Svg(W, 760, "Go 服务内部分层")
    s.title_block("服务内部分层（Kratos 布局，以 order-service 为例）",
                  "biz 层不依赖任何技术框架；仓储、外部调用、事务、事件发布以接口定义在 biz，由 data 层实现；server 只做组装")
    bands = [
        ("internal/service", "接口层", "svc", "协议适配：HTTP / gRPC / 消息 → 用例", [
            ("OrderService", "实现 proto 生成的接口"),
            ("参数校验 · uid 解析", "pkg/auth.UID(ctx)"),
            ("DTO ↔ 领域对象", "toReply()"),
            ("消息消费入口", "见 notification"),
        ]),
        ("internal/biz", "业务层", "data", "领域模型 + 用例 + 接口（纯 Go）", [
            ("Order / OrderStatus", "聚合根 · 状态规则"),
            ("OrderUsecase", "Create · GetForBuyer"),
            ("OrderRepo · BuyerGateway", "仓储 / 防腐层接口"),
            ("Transaction · Events", "事务 / 事件接口"),
        ]),
        ("internal/data", "数据层", "ops", "技术实现：数据库 · gRPC · 发件箱", [
            ("orderRepo", "OrderPO + GORM"),
            ("buyerGateway", "UserInternalServiceClient"),
            ("eventPublisher", "mq.Outbox（同一事务）"),
        ]),
        ("internal/server", "组装", "gray", "创建依赖并注册到服务器", [
            ("server.New(Deps)", "data → biz → service → HTTP / gRPC 服务器"),
            ("cmd/order/main.go", "加载配置 · 连接中间件 · 启动 App"),
        ]),
    ]
    s.group(20, 100, 940, 640, "app/order（一个可部署单元）", kind="gray", dash=None)
    y = 124
    for i, (pkg, name, kind, desc, comps) in enumerate(bands):
        main, light, dark = C[kind]
        s.rect(36, y, 908, 134, fill=light, stroke=main, sw=1.4, rx=10)
        s.text(56, y + 30, name, size=17, weight=700, fill=dark)
        s.text(56, y + 54, pkg, size=12.5, fill=dark, mono=True)
        for j, ln in enumerate(wrap(desc, 120, 12)):
            s.text(56, y + 80 + j * 18, ln, size=12, fill="#475569")
        bw = (908 - 190 - (len(comps) - 1) * 10) / len(comps)
        for j, (t, ln) in enumerate(comps):
            s.box(36 + 180 + j * (bw + 10), y + 22, bw, 90, t, [ln], kind=kind, title_size=13, line_size=12)
        if i == 0:
            s.arrow([(120, y + 134), (120, y + 154)], color="#0f172a", sw=2)
        y += 154
    s.arrow([(120, 124 + 2 * 154), (120, 124 + 154 + 136)], color=C["ops"][0], dash="6 4")
    s.note(130, 124 + 2 * 154 - 9, "实现 biz 中的接口（依赖倒置）", size=12, fill=C["ops"][0])

    s.group(990, 100, 430, 270, "契约（api/，protobuf + 生成代码）", kind="cloud", label_size=14)
    s.box(1010, 130, 390, 100, "api/user/v1", ["UserService（对外 HTTP，google.api.http）", "UserInternalService（仅 gRPC）",
                                                "buf breaking 检查兼容性"], kind="cloud", icon="fn")
    s.box(1010, 250, 390, 100, "api/order/v1", ["OrderService（对外 HTTP）", "OrderCreated 事件 · EventsTopic 常量",
                                                 "消费方 notification 依赖"], kind="cloud", icon="fn")
    s.arrow([(944, 124 + 2 * 154 + 67), (975, 124 + 2 * 154 + 67), (975, 180), (1008, 180)], color=C["cloud"][0],
            label="依赖", label_pos=(975, 300))

    s.rect(990, 400, 430, 340, fill="#fef2f2", stroke=C["sec"][0], sw=1.4, rx=12)
    s.text(1010, 428, "分层规则（■ archtest 自动校验  □ 代码评审）", size=15, weight=700, fill=C["sec"][2])
    rules = [(1, "biz 不导入 data / service / server"), (1, "biz 不导入 gorm、net/http、grpc、watermill"),
             (1, "service 不直接访问 data 与 gorm"), (1, "data 不导入 service / server"),
             (0, "远程调用放在事务之外"), (0, "对外只返回 proto 消息，不暴露 PO"),
             (0, "跨服务只能依赖对方的 api/ 契约"), (0, "发布事件必须经 mq.Outbox")]
    for i, (auto, r) in enumerate(rules):
        fill = C["sec"][0] if auto else "#ffffff"
        s.rect(1013, 456 + i * 34, 11, 11, fill=fill, stroke=C["sec"][0], sw=1.4, rx=2)
        s.text(1034, 462 + i * 34, r, size=13.5, fill="#1f2937")
    return s


def modules():
    s = Svg(W, 740, "Go 代码模块与依赖")
    s.title_block("代码模块结构与依赖关系（acme-go，单一 Go module）",
                  "依赖只能自上而下；服务只依赖 pkg/ 平台库与其它服务的 api/ 契约，三方依赖版本由 go.mod 统一锁定")
    rows = [
        ("可部署单元", "app/*/cmd", "svc", [("gateway", "网关"), ("auth", "认证中心"), ("user", "用户服务"),
                                          ("order", "订单服务"), ("notification", "通知服务")]),
        ("服务契约", "api/*/v1", "cloud", [("user/v1", "HTTP + gRPC"), ("order/v1", "HTTP + 事件"),
                                         ("notification/v1", "HTTP")]),
        ("平台库", "pkg/", "k8s", [("server", "中间件 · 管理端口"), ("client", "gRPC 调用"), ("auth", "JWT"),
                                  ("database", "GORM · goose"), ("mq", "Watermill · Outbox"), ("testkit", "测试")]),
        ("基础", "pkg/", "k8s", [("conf", "分层配置 · Nacos"), ("registry", "Nacos 注册"), ("selector/gray", "灰度"),
                                ("errorx · result", "错误码 · 响应体"), ("observability", "日志 · 链路 · 指标")]),
        ("版本基线", "go.mod", "ops", [("Go 1.26", "Kratos 2.9.2"), ("gRPC · protobuf", "buf 生成"),
                                      ("nacos-sdk-go 2.3.5", "GORM 1.31"), ("Watermill 1.5", "OTel 1.47")]),
    ]
    y = 100
    for name, sub, kind, items in rows:
        main, light, dark = C[kind]
        s.rect(20, y, 860, 108, fill=light, stroke=main, sw=1.4, rx=10)
        s.text(38, y + 44, name, size=16, weight=700, fill=dark)
        s.text(38, y + 68, sub, size=12.5, fill="#475569", mono=True)
        bw = (860 - 170 - (len(items) - 1) * 10) / len(items)
        for j, (t, ln) in enumerate(items):
            s.box(20 + 160 + j * (bw + 10), y + 18, bw, 72, t, [ln], kind=kind, title_size=13.5 if bw > 120 else 12.5,
                  line_size=11.5)
        if y < 100 + 4 * 128:
            s.arrow([(100, y + 108), (100, y + 126)], color="#0f172a", sw=2)
        y += 128

    cols = ["server", "client", "auth", "database", "mq", "conf", "registry", "user/v1", "order/v1"]
    deploy = [("gateway", {"server": 1, "auth": 2, "conf": 1, "registry": 1}),
              ("auth", {"server": 1, "auth": 2, "conf": 1}),
              ("user", {"server": 1, "auth": 1, "database": 1, "conf": 1, "registry": 1, "user/v1": 1}),
              ("order", {"server": 1, "client": 1, "auth": 1, "database": 1, "mq": 1, "conf": 1, "registry": 1,
                         "user/v1": 1, "order/v1": 1}),
              ("notification", {"server": 1, "auth": 1, "database": 1, "mq": 1, "conf": 1, "registry": 1,
                                "order/v1": 1})]
    mx, my = 1066, 260
    cw, rh = 36, 48
    s.text(910, 112, "依赖矩阵", size=16, weight=700, fill="#0f172a")
    s.text(910, 136, "● 直接依赖   ○ 仅复用 Claims 类型", size=12.5, fill="#475569")
    for j, c in enumerate(cols):
        cx = mx + j * cw + cw / 2
        s.raw(f'<text x="{cx}" y="{my - 14}" font-family="\'JetBrains Mono\',Consolas,monospace" font-size="12.5" '
              f'fill="#334155" transform="rotate(-55 {cx} {my - 14})">{c}</text>')
    for i, (name, deps) in enumerate(deploy):
        yy = my + i * rh
        s.rect(900, yy, mx - 900 + len(cols) * cw + 8, rh - 6, fill="#f8fafc" if i % 2 == 0 else "#ffffff", rx=6)
        s.text(912, yy + rh / 2 - 3, name, size=13, weight=600, fill=C["svc"][2], mono=True)
        for j, c in enumerate(cols):
            cx = mx + j * cw + cw / 2
            v = deps.get(c)
            if v == 1:
                s.raw(f'<circle cx="{cx}" cy="{yy + rh / 2 - 3}" r="7" fill="{C["svc"][0]}"/>')
            elif v == 2:
                s.raw(f'<circle cx="{cx}" cy="{yy + rh / 2 - 3}" r="6.5" fill="#fff" stroke="{C["svc"][0]}" '
                      f'stroke-width="2"/>')
    ny = my + len(deploy) * rh + 30
    notes = ["单一 Go module：跨服务重构与统一升级最方便", "服务之间只通过 api/ 契约交互，禁止导入对方 internal/",
             "消息中间件由 MQ_DRIVER 运行时选择，二进制同时支持 Kafka / RabbitMQ", "新增服务：复制 app/user 并改名"]
    for i, n in enumerate(notes):
        s.raw(f'<circle cx="{912}" cy="{ny + i * 28}" r="3.5" fill="#475569"/>')
        s.text(924, ny + i * 28, n, size=13, fill="#334155")
    return s
