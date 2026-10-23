# Spring Cloud 中大型微服务系统架构说明

基于 **Spring Boot 4 + Spring Cloud 2025.1 + Spring Cloud Alibaba**、运行在 **Kubernetes** 上的中大型微服务系统参考架构，
数据库 **MySQL / PostgreSQL** 二选一，消息 **Kafka / RabbitMQ** 二选一。仓库包含两部分：

1. **架构说明文档**（Markdown + HTML 两个版本，图文并茂）；
2. **可运行的初始工程** [`acme-cloud/`](acme-cloud)：框架层 starter、网关、认证中心、三个示例业务服务、Helm / GitOps / CI 部署清单，`mvn verify` 全部通过。

> 📘 **HTML 版本**：[`html/index.html`](html/index.html)，单文件、可离线打开；内置目录导航，所有图表为矢量图，点击可全屏放大查看。

![Spring Cloud 各项技术在运行时的位置](docs/images/spring-cloud-landscape.svg)

## 文档目录

| 章节 | 内容 |
| --- | --- |
| [第 1 章 运行架构](docs/01-runtime-architecture.md) | 测试 / 预发布 / 生产环境规划与隔离；**方案 A 全自建**、**方案 B 阿里云** 的完整运行架构；AWS / Azure 对应方案与组件对照；高可用、发布、灰度、回滚 |
| [第 2 章 Spring Cloud 技术落地](docs/02-spring-cloud-in-practice.md) | 版本基线；Nacos 注册发现与配置、Gateway、OpenFeign + LoadBalancer、CircuitBreaker、Stream + Outbox、Authorization Server、可观测在整套环境中的位置与各环境差异 |
| [第 3 章 团队开发规范](docs/03-team-conventions.md) | 服务拆分与命名、仓库与分支、分层与包结构、API / 错误码、服务调用、数据库、消息、配置、日志、安全、测试、部署规范与评审清单 |
| [第 4 章 初始代码结构](docs/04-code-structure.md) | 模块全景与目录树、平台 starter、关键代码导读、本地开发、构建部署、新增服务步骤 |

## 架构图一览

| 图 | 说明 |
| --- | --- |
| [环境规划与发布晋级](docs/images/environments.svg) | dev / test / staging / prod 定位、隔离与晋级路径 |
| [方案 A：全自建](docs/images/arch-selfhosted.svg) | IDC / 私有云生产环境运行架构 |
| [方案 B：阿里云](docs/images/arch-aliyun.svg) | ACK + MSE + RDS / PolarDB + 云消息队列 + ARMS / SLS |
| [AWS / Azure](docs/images/arch-aws-azure.svg) | EKS / AKS 对应部署简图 |
| [组件选型对照](docs/images/cloud-mapping.svg) | 自建 / 阿里云 / AWS / Azure 20 项能力对照 |
| [Spring Cloud 组件分布](docs/images/spring-cloud-landscape.svg) | 一次请求经过的全部 Spring Cloud 组件 |
| [服务内部分层](docs/images/service-layers.svg) | interfaces / application / domain / infrastructure |
| [代码模块与依赖](docs/images/code-modules.svg) | 模块分层与依赖矩阵 |

## 技术栈

| 类别 | 选型 |
| --- | --- |
| 语言 / 框架 | Java 21、Spring Boot 4.0.9、Spring Cloud 2025.1.4、Spring Cloud Alibaba 2025.1.0.0 |
| 注册 / 配置 | Nacos 3.x（自建集群 / 阿里云 MSE） |
| 网关 / 调用 / 容错 | Spring Cloud Gateway、OpenFeign、Spring Cloud LoadBalancer（灰度）、Spring Cloud CircuitBreaker（Resilience4j） |
| 消息 | Spring Cloud Stream（Kafka / RabbitMQ Binder）+ 事务发件箱 + 幂等消费 |
| 安全 | Spring Authorization Server（OAuth2.1 / OIDC）+ Resource Server（JWT） |
| 数据 | MySQL 8.4 / PostgreSQL 17、MyBatis-Plus 3.5、Flyway |
| 可观测 | Micrometer、OpenTelemetry（OTLP）、Prometheus / ARMS、Loki / SLS |
| 交付 | Docker 分层镜像、Helm 通用 Chart、Argo CD（GitOps）、GitLab CI / 云效 |

## 快速开始

```bash
cd acme-cloud
mvn verify                                                     # 编译 + 30 个测试（无需外部中间件）
docker compose -f deploy/local/docker-compose.yml up -d        # 本地中间件
```

详见 [第 4 章 4.5 本地开发](docs/04-code-structure.md#45-本地开发)。

## 仓库结构

```text
.
├── README.md
├── docs/                    # Markdown 版文档（GitHub 直接渲染，含 Mermaid 图）
│   └── images/              # 架构图（SVG）
├── html/index.html          # HTML 版文档（由 tools/doc-site 生成）
├── acme-cloud/              # 初始工程（Maven 多模块 + deploy/）
└── tools/
    ├── diagrams/            # 架构图生成脚本（Python，修改后重新生成 SVG）
    └── doc-site/            # HTML 版生成脚本（Markdown → 单文件 HTML，预渲染 Mermaid）
```

## 文档维护

```bash
python3 tools/diagrams/build_diagrams.py        # 重新生成 docs/images/*.svg
cd tools/doc-site && npm ci && npm run build    # 重新生成 html/index.html（需要 Chromium，用于预渲染 Mermaid）
```
