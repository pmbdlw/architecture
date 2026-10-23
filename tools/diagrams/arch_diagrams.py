"""生产运行架构图：方案 A 全自建 / 方案 B 阿里云。"""
from svgkit import Svg, C

W = 1440


def _users(s):
    s.group(430, 112, 580, 82, "用户端", kind="user", dash=None)
    for i, (t, ln) in enumerate([("Web / H5", "浏览器"), ("App / 小程序", "移动端"), ("合作方", "开放 API")]):
        s.box(450 + i * 186, 128, 170, 54, t, [ln], kind="user", icon="user")


def _business_namespace(s, x, y, w, extra_note):
    s.group(x, y, w, 250, "命名空间 acme（业务）", kind="svc", sub="Pod IP 注册到 Nacos，服务间直连 Pod")
    s.box(x + 20, y + 24, 440, 66, "acme-gateway ×3", ["Spring Cloud Gateway · JWT 校验 · Redis 限流 · 熔断 · 灰度打标"],
          kind="cloud", icon="lb")
    s.box(x + w - 460, y + 24, 440, 66, "acme-auth ×2", ["Spring Authorization Server · OAuth2.1 / OIDC · 签发 JWT"],
          kind="cloud", icon="lock")
    names = [("user-service ×3", "MVC · MyBatis-Plus"), ("order-service ×3", "Feign · Stream(Outbox)"),
             ("notification-service ×2", "Stream 消费 · 幂等"), ("…其它领域服务", "同一套 starter")]
    bw = (w - 40 - 3 * 12) / 4
    for i, (t, ln) in enumerate(names):
        s.box(x + 20 + i * (bw + 12), y + 120, bw, 62, t, [ln], kind="svc", title_size=14.5)
    s.arrow([(x + 240, y + 90), (x + 240, y + 120)], color=C["cloud"][0], label="lb://服务名（灰度 LB）",
            label_pos=(x + 345, y + 105))
    s.arrow([(x + 20 + bw + 12 + bw / 2, y + 182), (x + 20 + bw + 12 + bw / 2, y + 200),
             (x + 20 + bw / 2, y + 200), (x + 20 + bw / 2, y + 184)], color=C["svc"][0], dash="5 3")
    s.note(x + 20 + bw + 6, y + 213, "OpenFeign（内部 /internal/**）", size=12, fill=C["svc"][2], anchor="middle")
    s.note(x + 20, y + 236, extra_note, size=12.5, fill="#334155")


def selfhosted():
    s = Svg(W, 1080, "方案A 全自建生产环境运行架构")
    s.title_block("方案 A：全自建（IDC / 私有云）— 生产环境运行架构",
                  "测试、预发布环境使用同一套拓扑的缩小版（见“环境规划”）；所有组件均为开源或自有软件")
    s.legend(870, 34, [("access", "接入"), ("cloud", "Spring Cloud 基础设施"), ("svc", "业务服务")])
    s.legend(870, 62, [("data", "数据"), ("mq", "消息/缓存"), ("obs", "可观测"), ("ops", "研发运维")])
    _users(s)
    s.arrow([(720, 194), (720, 232)], color=C["access"][0])

    # 接入层
    s.group(20, 242, 1400, 86, "接入层（IDC 出口 · 双线 BGP）", kind="access")
    items = [("智能 DNS / GSLB", "就近解析 · 故障切换", "globe", 220),
             ("CDN", "静态资源 · 图片", "globe", 180),
             ("防火墙 + WAF / 抗 D", "ModSecurity 或商用设备", "shield", 260),
             ("L4 负载均衡（VIP）", "F5 或 LVS + Keepalived", "lb", 270),
             ("专线 / VPN", "办公网 · 合作方 · 灾备机房", "lock", 300)]
    x = 40
    for t, ln, ic, w in items:
        s.box(x, 258, w, 56, t, [ln], kind="access", icon=ic)
        x += w + 15
    s.arrow([(40 + 220 + 15 + 180 + 15 + 260 + 15 + 135, 314), (40 + 220 + 15 + 180 + 15 + 260 + 15 + 135, 386)],
            color=C["access"][0], label="TCP 443", label_pos=(905, 345))

    # K8s 集群
    s.group(20, 362, 990, 562, "Kubernetes 生产集群", kind="k8s",
            sub="RKE2 / kubeadm · 3 Master + N Worker · 跨机柜/机房打散")
    s.box(40, 390, 950, 46, "ingress-nginx（边缘节点 DaemonSet · TLS 终止 · api.acme.com → 网关，auth.acme.com → 认证中心）",
          kind="access", title_size=14)
    s.arrow([(280, 436), (280, 482)], color=C["access"][0])
    s.arrow([(760, 436), (760, 482)], color=C["access"][0])
    _business_namespace(s, 40, 460, 950, "每个服务：Deployment + HPA + PDB + NetworkPolicy · 跨节点打散 · 镜像按 digest 部署 · 探针走 8081 管理端口")

    s.group(40, 738, 460, 168, "命名空间 middleware（专用节点池 · 本地 SSD）", kind="mq", label_size=13.5)
    s.box(56, 762, 428, 62, "Kafka 3 Broker（Strimzi · KRaft）", ["或 RabbitMQ 3 节点（Cluster Operator · 仲裁队列）"],
          kind="mq", icon="mq")
    s.box(56, 834, 428, 58, "Redis Sentinel（1 主 2 从 + 3 哨兵）", ["网关限流 · 缓存 · 分布式锁"], kind="mq", icon="bolt")

    s.group(520, 738, 470, 168, "命名空间 observability", kind="obs", label_size=13.5)
    obs = [("Prometheus", "指标 · Alertmanager"), ("Grafana", "大盘"), ("OTel Collector", "链路接收/转发"),
           ("Tempo / SkyWalking", "链路存储"), ("Loki（或 ELK）", "日志"), ("Fluent Bit", "日志采集")]
    for i, (t, ln) in enumerate(obs):
        cx, cy = 536 + (i % 3) * 150, 762 + (i // 3) * 72
        s.box(cx, cy, 142, 62, t, [ln], kind="obs", title_size=13, line_size=12)

    s.arrow([(455, 710), (455, 762)], color=C["mq"][0], label="Stream 发布/订阅", label_pos=(455, 724))

    # K8s 外部：数据与注册配置
    s.group(1030, 362, 390, 562, "K8s 外：有状态核心（物理机 / 虚机）", kind="data", label_size=14)
    s.box(1048, 392, 354, 82, "MySQL 8.4 InnoDB Cluster", ["MGR 单主 3 节点 + MySQL Router", "一服务一库 · 半同步 · 读写分离可选"],
          kind="data", icon="db")
    s.note(1225, 488, "— 或 —", anchor="middle", size=12.5, fill=C["data"][2])
    s.box(1048, 502, 354, 70, "PostgreSQL 17 + Patroni", ["1 主 2 从 · etcd 选主 · PgBouncer 连接池"], kind="data",
          icon="db")
    s.box(1048, 592, 354, 82, "Nacos 3.x 集群 ×3", ["注册中心 + 配置中心", "命名空间 = 环境 · 外置 MySQL 存储"], kind="cloud",
          icon="compass")
    s.box(1048, 694, 354, 62, "MinIO · Elasticsearch", ["对象存储 · 全文检索"], kind="data", icon="box")
    s.box(1048, 776, 354, 70, "备份与容灾", ["XtraBackup / pgBackRest → 异地机房", "Velero 备份集群资源 · 定期恢复演练"],
          kind="ops", icon="doc")
    s.note(1048, 870, "原则：数据库不进 K8s；中间件由 Operator 管理", size=12.5, fill=C["data"][2])
    s.note(1048, 892, "或放虚机，生产与非生产物理隔离", size=12.5, fill=C["data"][2])

    s.arrow([(990, 625), (1020, 625), (1020, 432), (1046, 432)], color=C["data"][0], label="JDBC",
            label_pos=(1020, 528))
    s.arrow([(990, 690), (1030, 690), (1030, 632), (1046, 632)], color=C["cloud"][0], dash="6 4",
            label="注册/配置", label_pos=(1010, 712))

    # 研发运维
    s.group(20, 958, 1400, 106, "研发与运维支撑（独立管理集群 / 虚机，所有环境共用）", kind="ops")
    ops = [("GitLab + Runner", "代码 · CI", "git"), ("Harbor", "镜像 · Trivy 扫描", "box"),
           ("Nexus", "Maven 私服", "box"), ("SonarQube", "质量门禁", "chart"), ("Argo CD", "GitOps 部署", "k8s"),
           ("Vault", "密钥 · 动态凭证", "lock"), ("Keycloak / LDAP", "员工 SSO", "user"),
           ("JumpServer", "堡垒机 · 审计", "shield")]
    bw = (1400 - 40 - 7 * 10) / 8
    for i, (t, ln, ic) in enumerate(ops):
        s.box(40 + i * (bw + 10), 980, bw, 64, t, [ln], kind="ops", icon=ic, title_size=13.5, line_size=12)
    return s


def aliyun():
    s = Svg(W, 1080, "方案B 阿里云生产环境运行架构")
    s.title_block("方案 B：阿里云 — 生产环境运行架构",
                  "有状态组件全部使用云产品（多可用区高可用），K8s 只运行无状态服务；测试/预发布为同构的独立账号与 VPC")
    s.legend(870, 34, [("aliyun", "阿里云产品"), ("cloud", "Spring Cloud 基础设施"), ("svc", "业务服务")])
    s.legend(870, 62, [("data", "数据"), ("mq", "消息/缓存"), ("obs", "可观测"), ("ops", "研发治理")])
    _users(s)
    s.arrow([(720, 194), (720, 232)], color=C["aliyun"][0])

    s.group(20, 242, 1400, 86, "接入层（全部托管）", kind="aliyun")
    items = [("云解析 DNS", "智能解析 · 健康检查", "globe", 220),
             ("CDN / DCDN", "静态资源 · 全站加速", "globe", 200),
             ("WAF 3.0 + DDoS 高防", "Web 防护 · 抗 D", "shield", 250),
             ("ALB 应用型负载均衡", "多可用区 · 托管证书", "lb", 260),
             ("CEN 云企业网 / VPN 网关", "办公网 · IDC · 多 VPC 互通", "lock", 315)]
    x = 40
    for t, ln, ic, w in items:
        s.box(x, 258, w, 56, t, [ln], kind="aliyun", icon=ic)
        x += w + 15
    alb_cx = 40 + 220 + 15 + 200 + 15 + 250 + 15 + 130
    s.arrow([(alb_cx, 314), (alb_cx, 386)], color=C["aliyun"][0], label="HTTPS", label_pos=(alb_cx, 345))

    s.group(20, 362, 990, 562, "ACK Pro 托管集群", kind="aliyun",
            sub="生产账号 · 专有网络 VPC · 3 可用区交换机 · ECS 节点池 + ECI 弹性")
    s.box(40, 390, 950, 46, "ALB Ingress Controller（api.acme.com → 网关，auth.acme.com → 认证中心；可选 MSE 云原生网关）",
          kind="aliyun", title_size=14)
    s.arrow([(280, 436), (280, 482)], color=C["aliyun"][0])
    s.arrow([(760, 436), (760, 482)], color=C["aliyun"][0])
    _business_namespace(s, 40, 460, 950, "Pod 通过 RRSA 获取 RAM 角色访问云产品（免 AccessKey） · Terway 网络 · 安全组 + NetworkPolicy")

    s.group(40, 738, 460, 168, "集群组件（ACK 应用市场 / 组件中心）", kind="k8s", label_size=13.5)
    comps = [("ARMS 探针 / Prometheus", "指标 · 链路采集"), ("LoongCollector", "日志采集 → SLS"),
             ("ESO / ack-secret-manager", "同步 KMS 凭据"), ("cluster-autoscaler", "节点自动伸缩")]
    for i, (t, ln) in enumerate(comps):
        s.box(56 + (i % 2) * 218, 762 + (i // 2) * 72, 208, 62, t, [ln], kind="k8s", title_size=13, line_size=12)

    s.group(520, 738, 470, 168, "可观测（云产品）", kind="obs", label_size=13.5)
    obs = [("ARMS 应用监控", "JVM · 接口 · 拓扑"), ("可观测链路 OTel 版", "接收 OTLP"), ("Prometheus 监控", "托管 + Grafana"),
           ("SLS 日志服务", "检索 · 告警 · 投递"), ("云监控", "资源 · 站点探测"), ("ARMS 告警", "钉钉 · 电话")]
    for i, (t, ln) in enumerate(obs):
        s.box(536 + (i % 3) * 150, 762 + (i // 3) * 72, 142, 62, t, [ln], kind="obs", title_size=13, line_size=12)

    s.group(1030, 362, 390, 562, "云数据库与中间件（VPC 内网 · 多可用区）", kind="data", label_size=13.5)
    s.box(1048, 392, 354, 76, "RDS MySQL 高可用版 / PolarDB", ["主备跨可用区 · 自动备份 · 只读实例", "一服务一库（或一服务一实例）"],
          kind="data", icon="db", title_size=14.5)
    s.box(1048, 480, 354, 62, "RDS PostgreSQL / PolarDB PG", ["与 MySQL 二选一"], kind="data", icon="db", title_size=14.5)
    s.box(1048, 554, 354, 70, "MSE Nacos 专业版", ["注册中心 + 配置中心 · 3 节点多可用区", "可叠加 MSE 服务治理（全链路灰度）"],
          kind="cloud", icon="compass", title_size=14.5)
    s.box(1048, 636, 354, 70, "云消息队列 Kafka 版 / RabbitMQ 版", ["Serverless 或专业版 · 多可用区"], kind="mq",
          icon="mq", title_size=14.5)
    s.box(1048, 718, 354, 62, "Tair（Redis 开源版兼容）", ["高可用版 · 限流 · 缓存"], kind="mq", icon="bolt", title_size=14.5)
    s.box(1048, 792, 354, 62, "OSS · 检索分析服务 Elasticsearch", ["对象存储 · 全文检索"], kind="data", icon="box",
          title_size=14.5)
    s.note(1048, 880, "容灾：RDS 跨地域备份 · OSS 跨区域复制", size=12.5, fill=C["data"][2])
    s.note(1048, 902, "两地三中心可选 PolarDB 全球数据库", size=12.5, fill=C["data"][2])

    s.arrow([(990, 625), (1020, 625), (1020, 430), (1046, 430)], color=C["data"][0], label="VPC 内网",
            label_pos=(1020, 528))
    s.arrow([(990, 690), (1032, 690), (1032, 590), (1046, 590)], color=C["cloud"][0], dash="6 4",
            label="注册/配置", label_pos=(1010, 712))

    s.group(20, 958, 1400, 106, "研发、安全与治理", kind="ops")
    ops = [("云效 / 自建 GitLab", "代码 · 流水线", "git"), ("ACR 企业版", "镜像 · 扫描 · 加签", "box"),
           ("ACK One GitOps", "托管 Argo CD", "k8s"), ("KMS 凭据管家", "密钥 · 轮转", "lock"),
           ("RAM + 资源目录", "多账号 · 最小权限", "user"), ("云防火墙", "南北/东西向", "shield"),
           ("云安全中心", "主机 · 容器安全", "shield"), ("堡垒机 · 操作审计", "ActionTrail", "doc")]
    bw = (1400 - 40 - 7 * 10) / 8
    for i, (t, ln, ic) in enumerate(ops):
        s.box(40 + i * (bw + 10), 980, bw, 64, t, [ln], kind="ops", icon=ic, title_size=13.5, line_size=12)
    return s
