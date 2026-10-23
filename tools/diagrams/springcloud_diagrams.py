"""Spring Cloud 组件运行时分布图、服务内分层图、代码模块依赖图。"""
from svgkit import Svg, C, wrap

W = 1440


def badge(s, x, y, n, color="#0f172a"):
    s.raw(f'<circle cx="{x}" cy="{y}" r="11" fill="{color}"/>')
    s.text(x, y + 0.5, str(n), size=12.5, weight=700, fill="#fff", anchor="middle")


def stack(s, x, y, w, items, kind, h=50, gap=8, title_size=13.5, line_size=12):
    for i, (t, ln) in enumerate(items):
        s.box(x, y + i * (h + gap), w, h, t, [ln], kind=kind, title_size=title_size, line_size=line_size)


def landscape():
    s = Svg(W, 960, "Spring Cloud 组件运行时分布")
    s.title_block("Spring Cloud 各项技术在运行时的位置（以“下单”请求为例）",
                  "①请求进入  ②网关鉴权·限流·灰度  ③负载均衡到服务  ④Feign 同步调用  ⑤本地事务（订单+发件箱）  "
                  "⑥Stream 投递事件  ⑦异步消费")

    s.box(300, 92, 1120, 62, "Nacos（自建集群 / 阿里云 MSE）· Spring Cloud Alibaba",
          ["Nacos Discovery：服务注册与发现（实例元数据含 gray-tag / version）   ·   Nacos Config：spring.config.import 加载、"
           "修改后动态刷新"], kind="cloud", icon="compass", title_size=15)
    for xx in (470, 870, 1265):
        s.arrow([(xx, 154), (xx, 192)], color=C["cloud"][0], dash="5 4", both=True)

    s.box(20, 300, 120, 90, "客户端", ["Web / App", "携带 JWT"], kind="user")
    s.box(165, 300, 120, 90, "Ingress", ["nginx / ALB", "TLS 终止"], kind="access")
    s.arrow([(140, 345), (163, 345)], color=C["access"][0])
    badge(s, 152, 322, 1)
    s.arrow([(285, 345), (308, 345)], color=C["access"][0])

    s.group(310, 200, 320, 360, "acme-gateway", kind="cloud", dash=None)
    stack(s, 326, 222, 288, [
        ("Spring Cloud Gateway", "WebFlux · 路由 · 过滤器链"),
        ("Spring Security", "OAuth2 Resource Server · 校验 JWT"),
        ("RequestRateLimiter", "Redis 令牌桶 · 按用户 / IP"),
        ("Spring Cloud CircuitBreaker", "Resilience4j · 超时 · fallback"),
        ("Spring Cloud LoadBalancer", "lb:// + 灰度实例筛选"),
        ("GrayTagFilter（自研）", "按规则打 X-Gray-Tag"),
    ], kind="cloud")
    badge(s, 470, 210, 2, C["cloud"][0])

    s.group(680, 200, 380, 360, "order-service", kind="svc", dash=None)
    stack(s, 696, 222, 348, [
        ("Spring MVC + Validation", "REST · 统一 Result · 全局异常"),
        ("Spring Security Resource Server", "JWT → uid / ROLE_* · @PreAuthorize"),
        ("OpenFeign + LoadBalancer + CircuitBreaker", "调用 user-service · 超时 · 熔断"),
        ("MyBatis-Plus + Flyway", "MySQL / PostgreSQL · 版本化迁移"),
        ("Spring Cloud Stream + Outbox", "事务内写事件 · StreamBridge 投递"),
        ("Actuator + Micrometer Tracing", "探针 · 指标 · OTel 链路"),
    ], kind="svc")
    s.arrow([(630, 300), (678, 300)], color=C["cloud"][0])
    badge(s, 654, 282, 3, C["cloud"][0])

    s.group(1110, 200, 310, 170, "user-service", kind="svc", dash=None)
    stack(s, 1126, 222, 278, [("Spring MVC", "GET /internal/v1/users/{id}"),
                              ("MyBatis-Plus", "user_account 表")], kind="svc")
    s.arrow([(1060, 330), (1108, 330)], color=C["svc"][0], label="Feign", label_pos=(1084, 314))
    badge(s, 1084, 346, 4, C["svc"][0])

    s.box(1110, 400, 310, 70, "MySQL / PostgreSQL", ["每个服务独立库，禁止跨库访问"], kind="data", icon="db")
    s.arrow([(1060, 440), (1108, 440)], color=C["data"][0])
    badge(s, 1084, 424, 5, C["data"][0])

    s.box(680, 618, 380, 74, "Kafka / RabbitMQ", ["destination：acme.order.events", "Kafka key = 订单 ID（保证同订单有序）"],
          kind="mq", icon="mq")
    s.arrow([(870, 560), (870, 616)], color=C["mq"][0], label="OutboxRelay → StreamBridge", label_pos=(870, 588))
    badge(s, 990, 588, 6, C["mq"][0])

    s.group(1110, 600, 310, 120, "notification-service", kind="svc", dash=None)
    s.box(1126, 622, 278, 84, "Stream 函数式 Consumer", ["消费组 = 服务名 · 重试 3 次", "死信 DLQ · consumed_event 幂等"],
          kind="svc", title_size=13.5, line_size=12)
    s.arrow([(1060, 655), (1108, 655)], color=C["mq"][0])
    badge(s, 1084, 638, 7, C["mq"][0])

    s.box(310, 618, 320, 74, "Redis / Tair", ["网关限流令牌桶（多副本共享）"], kind="mq", icon="bolt")
    s.arrow([(470, 560), (470, 616)], color=C["mq"][0], both=True)

    s.box(20, 600, 265, 120, "acme-auth", ["Spring Authorization Server", "OAuth2.1 / OIDC · 签发 JWT", "公开 /oauth2/jwks"],
          kind="cloud", icon="lock")
    s.arrow([(152, 600), (152, 520), (308, 520)], color=C["cloud"][0], dash="5 4", label="拉取 JWKS 验签",
            label_pos=(222, 520))

    s.group(20, 768, 1400, 172, "可观测：Spring Boot Actuator + Micrometer（所有服务统一由 acme-starter-base 提供）", kind="obs")
    obs = [("指标 Metrics", ["Micrometer → /actuator/prometheus", "Prometheus（自建）/ ARMS Prometheus"], "chart"),
           ("链路 Tracing", ["Micrometer Tracing + OpenTelemetry", "OTLP → Collector → Tempo / ARMS"], "chart"),
           ("日志 Logging", ["结构化 JSON（ECS）含 traceId", "Fluent Bit → Loki / LoongCollector → SLS"], "doc"),
           ("健康 Health", ["/actuator/health/liveness · readiness", "K8s 探针 · 优雅停机 · 摘流量"], "gear")]
    bw = (1400 - 40 - 3 * 16) / 4
    for i, (t, lines, ic) in enumerate(obs):
        s.box(40 + i * (bw + 16), 796, bw, 120, t, lines, kind="obs", icon=ic, title_size=15, line_size=12.5)
    return s


def layers():
    s = Svg(W, 760, "服务内部分层")
    s.title_block("服务内部分层（以 order-service 为例）",
                  "四层结构 + 依赖倒置：领域层不依赖任何技术框架；仓储与外部调用以接口定义在领域层，由基础设施层实现")
    bands = [
        ("interfaces", "接口层", "svc", "协议适配：HTTP / 消息 → 应用服务", [
            ("OrderController", "web · /api/v1/orders"),
            ("Request / VO", "web · 入参校验 · 出参脱敏"),
            ("内部接口", "internal · /internal/v1/**"),
            ("消息消费者", "mq · Stream Consumer"),
        ]),
        ("application", "应用层", "cloud", "用例编排 · 事务边界 · 发布事件", [
            ("OrderAppService", "create()：远程校验 → 事务（订单 + 发件箱）"),
            ("CreateOrderCommand", "用例入参（与 HTTP 解耦）"),
        ]),
        ("domain", "领域层", "data", "业务规则（纯 Java）", [
            ("model.Order / OrderStatus", "聚合根 · 状态流转规则"),
            ("repository.OrderRepository", "仓储接口"),
            ("gateway.BuyerGateway", "防腐层接口"),
            ("OrderErrorCode", "错误码 A2xxx"),
        ]),
        ("infrastructure", "基础设施层", "ops", "技术实现：数据库 · 远程调用 · 消息", [
            ("persistence.OrderRepositoryImpl", "OrderDO + OrderMapper（MyBatis-Plus）"),
            ("remote.BuyerGatewayImpl", "UserClient（OpenFeign）"),
            ("OutboxPublisher（starter）", "写 outbox_event 表"),
        ]),
    ]
    s.group(20, 100, 940, 640, "order-server（可部署单元）", kind="gray", dash=None)
    y = 124
    for i, (pkg, name, kind, desc, comps) in enumerate(bands):
        main, light, dark = C[kind]
        s.rect(36, y, 908, 134, fill=light, stroke=main, sw=1.4, rx=10)
        s.text(56, y + 30, name, size=17, weight=700, fill=dark)
        s.text(56, y + 54, pkg, size=13, fill=dark, mono=True)
        for j, ln in enumerate(wrap(desc, 120, 12)):
            s.text(56, y + 80 + j * 18, ln, size=12, fill="#475569")
        bw = (908 - 190 - (len(comps) - 1) * 10) / len(comps)
        for j, (t, ln) in enumerate(comps):
            s.box(36 + 180 + j * (bw + 10), y + 22, bw, 90, t, [ln], kind=kind, title_size=13, line_size=12)
        if i < 2:
            s.arrow([(120, y + 134), (120, y + 154)], color="#0f172a", sw=2)
        y += 154
    s.arrow([(120, 124 + 3 * 154), (120, 124 + 2 * 154 + 136)], color=C["ops"][0], dash="6 4")
    s.note(130, 124 + 3 * 154 - 9, "实现领域层接口（依赖倒置）", size=12, fill=C["ops"][0])

    s.group(990, 100, 430, 270, "契约模块（xxx-api，发布到 Maven 私服）", kind="cloud", label_size=14)
    s.box(1010, 130, 390, 100, "user-api", ["UserClient（@FeignClient，path=/internal/v1/users）", "UserDTO（record）",
                                            "只依赖 common-core，不带实现"], kind="cloud", icon="fn")
    s.box(1010, 250, 390, 100, "order-api", ["OrderCreatedEvent（事件契约 v1）", "OrderEventTypes（类型 / Topic 常量）",
                                             "消费方 notification-server 依赖"], kind="cloud", icon="fn")
    s.arrow([(944, 124 + 3 * 154 + 67), (975, 124 + 3 * 154 + 67), (975, 180), (1008, 180)], color=C["cloud"][0],
            label="依赖", label_pos=(975, 420))

    s.rect(990, 400, 430, 340, fill="#fef2f2", stroke=C["sec"][0], sw=1.4, rx=12)
    s.text(1010, 428, "分层规则（■ ArchUnit 自动校验  □ 代码评审）", size=15, weight=700, fill=C["sec"][2])
    rules = [(1, "interfaces → application → domain，禁止反向依赖"), (1, "domain 不依赖 Spring Web / MyBatis / Feign"),
             (1, "Controller 不得直接访问 Mapper / DO"), (0, "远程调用放在事务之外"),
             (0, "对外只暴露 VO / DTO，不暴露 DO 与领域对象"), (0, "跨服务只能依赖对方的 xxx-api 模块"),
             (0, "禁止共享数据库表，跨域数据走 API / 事件"), (0, "发布事件必须经 OutboxPublisher")]
    for i, (auto, r) in enumerate(rules):
        fill = C["sec"][0] if auto else "#ffffff"
        s.rect(1013, 456 + i * 34, 11, 11, fill=fill, stroke=C["sec"][0], sw=1.4, rx=2)
        s.text(1034, 462 + i * 34, r, size=13.5, fill="#1f2937")
    return s


def modules():
    s = Svg(W, 740, "代码模块与依赖")
    s.title_block("代码模块结构与依赖关系（acme-cloud）",
                  "依赖只能自上而下；业务服务只依赖平台 starter 与其它服务的 api 模块，三方依赖版本全部由根 POM 的 BOM 锁定")
    rows = [
        ("可部署单元", "每个模块打一个镜像", "svc", [("acme-gateway", "网关"), ("acme-auth", "认证中心"),
                                             ("user-server", "用户服务"), ("order-server", "订单服务"),
                                             ("notification-server", "通知服务")]),
        ("服务契约", "xxx-api", "cloud", [("user-api", "Feign 接口 · DTO"), ("order-api", "领域事件")]),
        ("平台 starter", "acme-framework", "k8s", [("starter-web", "异常 · Result"), ("starter-feign", "调用 · 熔断"),
                                                ("starter-mybatis", "持久层 · Flyway"), ("starter-mq", "Stream · Outbox"),
                                                ("starter-security", "JWT 资源服务"), ("starter-test", "测试规范")]),
        ("基座", "所有服务必选", "k8s", [("starter-base", "Nacos · 灰度LB · 可观测 · 默认配置"),
                                   ("common-core", "Result · 错误码 · 常量（纯 Java）")]),
        ("版本基线", "根 POM 统一管理", "ops", [("Spring Boot 4.0.9", "Java 21"), ("Spring Cloud 2025.1.4", "Oakwood"),
                                         ("Spring Cloud Alibaba", "2025.1.0.0"), ("MyBatis-Plus 3.5.17", "springdoc 3.0.3")]),
    ]
    y = 100
    for name, sub, kind, items in rows:
        main, light, dark = C[kind]
        s.rect(20, y, 860, 108, fill=light, stroke=main, sw=1.4, rx=10)
        s.text(38, y + 44, name, size=16, weight=700, fill=dark)
        s.text(38, y + 68, sub, size=12.5, fill="#475569", mono=sub.startswith(("xxx", "acme")))
        bw = (860 - 170 - (len(items) - 1) * 10) / len(items)
        for j, (t, ln) in enumerate(items):
            s.box(20 + 160 + j * (bw + 10), y + 18, bw, 72, t, [ln], kind=kind, title_size=13.5 if bw > 120 else 12.5,
                  line_size=11.5)
        if y < 100 + 4 * 128:
            s.arrow([(100, y + 108), (100, y + 126)], color="#0f172a", sw=2)
        y += 128

    # 依赖矩阵
    cols = ["starter-base", "starter-web", "starter-feign", "starter-mybatis", "starter-mq", "starter-security",
            "starter-test", "user-api", "order-api"]
    deploy = [("acme-gateway", {"starter-base": 1}),
              ("acme-auth", {"starter-base": 1}),
              ("user-server", {"starter-base": 2, "starter-web": 1, "starter-mybatis": 1, "starter-security": 1,
                               "starter-test": 1, "user-api": 1}),
              ("order-server", {"starter-base": 2, "starter-web": 1, "starter-feign": 1, "starter-mybatis": 1,
                                "starter-mq": 1, "starter-security": 1, "starter-test": 1, "user-api": 1,
                                "order-api": 1}),
              ("notification-server", {"starter-base": 2, "starter-web": 1, "starter-mybatis": 1, "starter-mq": 1,
                                       "starter-security": 1, "starter-test": 1, "order-api": 1})]
    mx, my = 1066, 260
    cw, rh = 36, 48
    s.text(910, 112, "依赖矩阵", size=16, weight=700, fill="#0f172a")
    s.text(910, 136, "● 直接依赖   ○ 传递依赖", size=12.5, fill="#475569")
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
    notes = ["网关是 WebFlux 应用，不依赖 starter-web（Servlet）", "starter-feign / mq 只在需要时引入，不做“全家桶”",
             "消息 Binder（Kafka / RabbitMQ）由服务 pom 的 profile 选择", "新增服务：复制 user-service 目录并改名"]
    for i, n in enumerate(notes):
        s.raw(f'<circle cx="{912}" cy="{ny + i * 28}" r="3.5" fill="#475569"/>')
        s.text(924, ny + i * 28, n, size=13, fill="#334155")
    return s
