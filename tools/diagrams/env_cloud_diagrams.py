"""环境规划图、AWS/Azure 对应架构图、多云组件对照表。"""
from svgkit import Svg, C, text_width, wrap

W = 1440


def environments():
    s = Svg(W, 900, "环境规划与发布晋级")
    s.title_block("环境规划与发布晋级：一次构建，逐级晋级",
                  "同一个镜像（同一 digest）从测试一路晋级到生产；环境差异只存在于 GitOps values、Nacos 命名空间与 K8s Secret 中")

    # 交付流水线
    s.group(20, 100, 1400, 140, "交付流水线", kind="ops")
    steps = [("开发者", ["feature/* 短分支", "本地 docker compose 自测"], "user"),
             ("GitLab / 云效", ["Merge Request", "Code Review ≥1 人"], "git"),
             ("CI 流水线", ["编译 · 单测/集成测试", "ArchUnit · Sonar · Trivy"], "gear"),
             ("镜像仓库", ["Harbor / ACR", "tag = Git 短 SHA"], "box"),
             ("GitOps 仓库", ["envs/<env>/<服务>.yaml", "只改 image.tag/digest"], "doc"),
             ("Argo CD", ["比对期望状态", "同步到各环境集群"], "k8s")]
    bw, gap = 206, 26
    for i, (t, lines, ic) in enumerate(steps):
        x = 40 + i * (bw + gap)
        s.box(x, 124, bw, 96, t, lines, kind="ops", icon=ic, title_size=15)
        if i:
            s.arrow([(x - gap + 2, 172), (x - 2, 172)], color=C["ops"][0])

    envs = [
        ("开发 dev", "gray", [
            ("用途", "编码、自测、调试"),
            ("运行位置", "IDE 本地启动服务"),
            ("中间件", "docker compose 单机版"),
            ("数据", "本地造数，随时清空"),
            ("Nacos", "public 命名空间"),
            ("发布方式", "无（本地运行）"),
            ("访问", "localhost"),
        ]),
        ("测试 test", "svc", [
            ("用途", "功能测试、联调、自动化回归"),
            ("集群", "非生产集群 · 命名空间 acme"),
            ("规模", "1~2 副本 · 中间件小规格"),
            ("数据", "测试数据，可重置"),
            ("Nacos", "命名空间 test"),
            ("发布方式", "合并 main 后自动同步"),
            ("访问", "*.test.acme.com（内网/VPN）"),
        ]),
        ("预发布 staging", "data", [
            ("用途", "上线前回归、压测、发布演练"),
            ("集群", "独立集群，与生产同构（缩容）"),
            ("规模", "≥2 副本 · 中间件高可用小规格"),
            ("数据", "生产脱敏数据，定期同步"),
            ("Nacos", "命名空间 staging（配置项同生产）"),
            ("发布方式", "测试通过后人工晋级同一镜像"),
            ("访问", "*.staging.acme.com（白名单）"),
        ]),
        ("生产 prod", "sec", [
            ("用途", "对外提供服务"),
            ("集群", "独立账号/集群 · 多可用区"),
            ("规模", "≥3 副本 + HPA · 中间件全高可用"),
            ("数据", "真实数据 · 备份 · 审计"),
            ("Nacos", "命名空间 prod（变更需审批）"),
            ("发布方式", "变更单 → 手动 Sync → 灰度 → 全量"),
            ("访问", "api.acme.com / auth.acme.com"),
        ]),
    ]
    cw, cg, top = 335, 20, 330
    for i, (name, kind, rows) in enumerate(envs):
        x = 20 + i * (cw + cg)
        main, light, dark = C[kind]
        s.rect(x, top, cw, 420, fill=light, stroke=main, sw=1.6, rx=12)
        s.rect(x, top, cw, 46, fill=main, rx=12)
        s.rect(x, top + 30, cw, 16, fill=main, rx=0)
        s.text(x + cw / 2, top + 24, name, size=18, weight=700, fill="#fff", anchor="middle")
        for j, (k, v) in enumerate(rows):
            y = top + 72 + j * 50
            s.text(x + 18, y, k, size=13, weight=700, fill=dark)
            s.text(x + 18, y + 21, v, size=13.5, fill="#1f2937")
            if j < len(rows) - 1:
                s.line(x + 14, y + 37, x + cw - 14, y + 37, color=main, sw=0.6, dash="3 3")
    # 晋级路径：同一镜像 digest 从 test → staging → prod
    px = [20 + i * (cw + cg) + cw / 2 for i in range(1, 4)]
    s.arrow([(px[0], top + 420), (px[0], top + 446), (px[2], top + 446), (px[2], top + 422)], color="#0f172a",
            sw=2.2)
    s.line(px[1], top + 420, px[1], top + 446, color="#0f172a", sw=2.2)
    s.arrow([(px[1], top + 446), (px[1], top + 422)], color="#0f172a", sw=2.2)
    s.rect((px[0] + px[2]) / 2 - 250, top + 434, 500, 24, fill="#ffffff", rx=12, stroke="#0f172a", sw=1)
    s.text((px[0] + px[2]) / 2, top + 446.5, "晋级的是同一个镜像 digest，不重新构建、不改代码", size=13, weight=600,
           fill="#0f172a", anchor="middle")

    # Argo CD → 环境
    ax = 40 + 5 * (bw + gap) + bw / 2
    targets = [(20 + 1 * (cw + cg) + cw / 2, "自动同步", C["svc"][0]),
               (20 + 2 * (cw + cg) + cw / 2, "人工晋级后自动同步", C["data"][0]),
               (20 + 3 * (cw + cg) + cw / 2, "审批后手动 Sync", C["sec"][0])]
    s.line(ax, 220, ax, 268, color=C["ops"][0], sw=1.8)
    s.line(targets[0][0], 268, ax, 268, color=C["ops"][0], sw=1.8)
    for tx, label, col in targets:
        s.arrow([(tx, 268), (tx, top - 2)], color=col, sw=2, label=label, label_pos=(tx, 296))

    s.rect(20, 812, 1400, 70, fill="#f8fafc", stroke="#cbd5e1", sw=1, rx=10)
    s.text(40, 835, "隔离维度", size=14, weight=700, fill="#0f172a")
    s.text(40, 861, "网络（VPC / 网段 / 安全组） · 云账号或集群 · Nacos 命名空间（生产独立 Nacos 集群） · MQ 集群 / vhost · "
                    "数据库实例 · 密钥库路径 · 域名与证书 · 镜像仓库项目权限", size=13.5, fill="#334155")
    return s


def aws_azure():
    s = Svg(W, 704, "AWS 与 Azure 对应部署架构")
    s.title_block("AWS / Azure 对应部署架构（简图）",
                  "应用代码、Helm Chart、GitOps 流程完全复用；只替换托管云产品与 platform-env 中的地址、身份绑定方式")

    def panel(x0, kind, title, rows, side):
        main, light, dark = C[kind]
        s.rect(x0, 92, 690, 590, fill=light, stroke=main, sw=1.8, rx=14)
        s.rect(x0, 92, 690, 50, fill=main, rx=14)
        s.rect(x0, 126, 690, 16, fill=main, rx=0)
        s.text(x0 + 24, 118, title, size=19, weight=700, fill="#fff")
        y = 162
        for row in rows:
            if row[0] == "flow":
                _, items = row
                bw = (690 - 40 - (len(items) - 1) * 26) / len(items)
                for i, (t, ln, ic) in enumerate(items):
                    bx = x0 + 20 + i * (bw + 26)
                    s.box(bx, y, bw, 60, t, [ln], kind=kind, icon=ic, title_size=14, line_size=12)
                    if i:
                        s.arrow([(bx - 24, y + 30), (bx - 2, y + 30)], color=main)
                s.arrow([(x0 + 345, y + 60), (x0 + 345, y + 84)], color=main)
                y += 84
            elif row[0] == "cluster":
                _, name, sub = row
                s.group(x0 + 20, y + 12, 650, 214, name, kind="k8s", sub=sub, label_size=14)
                s.box(x0 + 36, y + 38, 618, 40, "Ingress（" + side + "）→ acme-gateway（Spring Cloud Gateway）", kind="access",
                      title_size=13.5)
                s.box(x0 + 36, y + 90, 300, 56, "acme-gateway · acme-auth", ["网关 · 认证中心"], kind="cloud",
                      title_size=13.5, line_size=12)
                s.box(x0 + 354, y + 90, 300, 56, "user / order / notification", ["业务服务（同一镜像）"], kind="svc",
                      title_size=13.5, line_size=12)
                s.box(x0 + 36, y + 158, 618, 52, "Nacos 集群（StatefulSet ×3）",
                      ["托管云无 Nacos：在集群内自建；或改用 Spring Cloud Kubernetes 方案"], kind="cloud", title_size=13.5,
                      line_size=12, icon="compass")
                y += 240
            elif row[0] == "grid":
                _, label, items, k = row
                s.text(x0 + 24, y + 6, label, size=14, weight=700, fill=dark)
                y += 18
                cols = 3
                bw = (690 - 40 - (cols - 1) * 12) / cols
                for i, (t, ln, ic) in enumerate(items):
                    bx = x0 + 20 + (i % cols) * (bw + 12)
                    by = y + (i // cols) * 64
                    s.box(bx, by, bw, 56, t, [ln], kind=k, icon=ic, title_size=13, line_size=11.5)
                y += ((len(items) + cols - 1) // cols) * 64 + 10

    panel(20, "aws", "Amazon Web Services", [
        ("flow", [("Route 53", "DNS", "globe"), ("CloudFront + WAF", "CDN · 防护", "shield"),
                  ("ALB", "LB Controller", "lb")]),
        ("cluster", "Amazon EKS", "多 AZ · 托管节点组 / Karpenter"),
        ("grid", "数据与中间件", [("Aurora / RDS MySQL", "或 RDS PostgreSQL", "db"),
                             ("ElastiCache", "Valkey / Redis", "bolt"),
                             ("Amazon MSK", "或 Amazon MQ RabbitMQ", "mq")], "data"),
        ("grid", "安全 · 可观测 · 交付", [("Secrets Manager", "+ IRSA / Pod Identity", "lock"),
                                   ("AMP + Grafana", "X-Ray(ADOT) · CloudWatch", "chart"),
                                   ("ECR + CodePipeline", "或 GitHub Actions + Argo CD", "box")], "ops"),
    ], "AWS Load Balancer Controller")
    panel(730, "azure", "Microsoft Azure", [
        ("flow", [("Azure DNS", "DNS", "globe"), ("Front Door + WAF", "CDN · 防护", "shield"),
                  ("Application Gateway", "AGIC", "lb")]),
        ("cluster", "Azure Kubernetes Service", "可用性区域 · 节点池 / KEDA"),
        ("grid", "数据与中间件", [("Azure Database", "for MySQL / PostgreSQL", "db"),
                             ("Azure Managed Redis", "缓存 · 限流", "bolt"),
                             ("Event Hubs", "Kafka 协议端点", "mq")], "data"),
        ("grid", "安全 · 可观测 · 交付", [("Key Vault", "+ Workload Identity", "lock"),
                                   ("Azure Monitor", "托管 Prometheus · App Insights", "chart"),
                                   ("ACR + Azure DevOps", "或 GitHub Actions + Flux", "box")], "ops"),
    ], "AGIC / App Routing")
    return s


MAPPING = [
    ("容器编排", "k8s", "Kubernetes（RKE2 / kubeadm）", "ACK Pro 托管版", "Amazon EKS", "Azure Kubernetes Service (AKS)"),
    ("镜像仓库", "box", "Harbor", "ACR 企业版", "Amazon ECR", "Azure Container Registry"),
    ("DNS", "globe", "PowerDNS / Bind（智能 DNS）", "云解析 DNS", "Route 53", "Azure DNS"),
    ("CDN", "globe", "商用 CDN", "CDN / DCDN", "CloudFront", "Azure Front Door"),
    ("WAF / 抗 D", "shield", "ModSecurity / 商用 WAF", "WAF 3.0 + DDoS 高防", "AWS WAF + Shield", "Azure WAF + DDoS Protection"),
    ("负载均衡 / Ingress", "lb", "F5 / LVS + ingress-nginx", "ALB + ALB Ingress", "ALB + Load Balancer Controller",
     "Application Gateway (AGIC)"),
    ("注册 + 配置中心", "compass", "Nacos 集群", "MSE Nacos", "Nacos on EKS", "Nacos on AKS"),
    ("API 网关", "lb", "Spring Cloud Gateway", "SCG（可选 MSE 云原生网关）", "SCG（可选 API Gateway）",
     "SCG（可选 API Management）"),
    ("MySQL", "db", "MySQL InnoDB Cluster", "RDS MySQL / PolarDB", "Aurora MySQL / RDS MySQL",
     "Azure Database for MySQL"),
    ("PostgreSQL", "db", "PostgreSQL + Patroni", "RDS PostgreSQL / PolarDB PG", "Aurora PostgreSQL / RDS PG",
     "Azure Database for PostgreSQL"),
    ("Redis", "bolt", "Redis Sentinel / Cluster", "Tair（Redis 兼容）", "ElastiCache (Valkey/Redis)",
     "Azure Managed Redis"),
    ("Kafka", "mq", "Kafka（Strimzi Operator）", "云消息队列 Kafka 版", "Amazon MSK", "Event Hubs（Kafka 协议）"),
    ("RabbitMQ", "mq", "RabbitMQ Cluster Operator", "云消息队列 RabbitMQ 版", "Amazon MQ for RabbitMQ",
     "RabbitMQ on AKS"),
    ("对象存储", "box", "MinIO", "OSS", "Amazon S3", "Blob Storage"),
    ("密钥管理", "lock", "HashiCorp Vault", "KMS 凭据管家", "Secrets Manager / KMS", "Key Vault"),
    ("工作负载身份", "user", "Vault Kubernetes Auth", "RRSA", "IRSA / EKS Pod Identity", "Workload Identity"),
    ("指标监控", "chart", "Prometheus + Grafana", "ARMS Prometheus", "Managed Prometheus + Grafana",
     "Azure Monitor Prometheus"),
    ("链路追踪", "chart", "OTel Collector + Tempo", "ARMS 可观测链路 OTel 版", "AWS X-Ray（ADOT）",
     "Application Insights"),
    ("日志", "doc", "Loki / ELK", "SLS 日志服务", "CloudWatch Logs", "Log Analytics"),
    ("CI / CD", "git", "GitLab CI + Argo CD", "云效 + ACK One GitOps", "CodePipeline + Argo CD",
     "Azure DevOps + Flux / Argo CD"),
]


def mapping():
    rh, top = 44, 150
    s = Svg(W, top + len(MAPPING) * rh + 70, "自建 阿里云 AWS Azure 组件对照")
    s.title_block("组件选型对照：自建 / 阿里云 / AWS / Azure",
                  "Spring Cloud 代码对四种部署形态完全一致；差异集中在平台层，由 platform-env ConfigMap、Secret 与 Helm values 吸收")
    cols = [("能力", "gray", 200), ("方案 A：全自建", "ops", 300), ("方案 B：阿里云", "aliyun", 300),
            ("AWS", "aws", 300), ("Azure", "azure", 300)]
    x = 20
    xs = []
    for name, kind, w in cols:
        main, light, dark = C[kind]
        xs.append((x, w))
        s.rect(x, 96, w - 6, 44, fill=main, rx=8)
        if kind == "aws":
            s.text(x + (w - 6) / 2, 114, name, size=17, weight=800, fill="#ffffff", anchor="middle")
            s.raw(f'<path d="M{x + (w - 6) / 2 - 24},{124} q24,10 48,0" fill="none" stroke="#ff9900" '
                  f'stroke-width="3" stroke-linecap="round"/>')
        elif kind == "azure":
            s.raw(f'<path d="M{x + (w - 6) / 2 - 46},{127} l9,-20 l9,20 z" fill="#50e6ff"/>')
            s.text(x + (w - 6) / 2 + 6, 118, name, size=17, weight=700, fill="#ffffff", anchor="middle")
        elif kind == "aliyun":
            gx = x + (w - 6) / 2 - text_width(name, 16) / 2 - 16
            s.raw(f'<path d="M{gx - 4},{109} h-6 v18 h6 M{gx + 8},{109} h6 v18 h-6" fill="none" stroke="#fff" '
                  f'stroke-width="2.6"/>')
            s.text(x + (w - 6) / 2 + 12, 118, name, size=16, weight=700, fill="#ffffff", anchor="middle")
        else:
            s.text(x + (w - 6) / 2, 118, name, size=16, weight=700, fill="#ffffff", anchor="middle")
        x += w
    for r, row in enumerate(MAPPING):
        y = top + r * rh
        s.rect(20, y, 1400 - 6, rh - 4, fill="#f8fafc" if r % 2 == 0 else "#ffffff", rx=6)
        cat, ic = row[0], row[1]
        s.icon(ic, xs[0][0] + 22, y + rh / 2 - 2, 20, "#475569")
        s.text(xs[0][0] + 40, y + rh / 2 - 1, cat, size=14, weight=700, fill="#0f172a")
        for c, val in enumerate(row[2:]):
            cx, cw = xs[c + 1]
            kind = cols[c + 1][1]
            lines = wrap(val, cw - 30, 13.5)
            for li, ln in enumerate(lines):
                yy = y + rh / 2 - 2 + (li - (len(lines) - 1) / 2) * 17
                s.text(cx + 14, yy, ln, size=13.5, fill=C[kind][2] if kind != "aws" else "#1f2937")
    s.note(20, top + len(MAPPING) * rh + 28,
           "注：AWS / Azure 无托管 Nacos，可在集群内部署 Nacos（StatefulSet + 托管 MySQL），或切换为 Spring Cloud Kubernetes "
           "（K8s Service 发现 + ConfigMap 配置），详见第 2 章“注册发现方案选择”。", size=13, fill="#475569")
    return s
