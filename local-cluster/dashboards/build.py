#!/usr/bin/env python3
"""生成 Grafana 看板 acme-overview.json（两套技术栈共用）。

  python3 local-cluster/dashboards/build.py

每个面板的查询用 `or` 同时覆盖 Go 版（Kratos / OTel 指标）与 Spring Cloud 版（Micrometer 指标）；
CPU / 内存来自 kubelet cAdvisor，两套实现口径一致。修改后重新运行 scripts/up.sh <stack> infra 生效。
"""
import json
import pathlib

PROM = {"type": "prometheus", "uid": "prometheus"}

# ---- 指标口径（左：Go 版，右：Spring Cloud 版）---------------------------------
GATEWAY_RATE = (
    'sum by (code) (label_replace(rate(http_server_request_duration_seconds_count{service="acme-gateway"}[1m]), "code", "$1", "http_response_status_code", "(.*)"))'
    ' or sum by (code) (label_replace(rate(http_server_requests_seconds_count{service="acme-gateway", uri!~"/actuator.*"}[1m]), "code", "$1", "status", "(.*)"))'
)
GATEWAY_P95 = (
    'histogram_quantile(0.95, sum by (le) (rate(http_server_request_duration_seconds_bucket{service="acme-gateway"}[1m])))'
    ' or histogram_quantile(0.95, sum by (le) (rate(http_server_requests_seconds_bucket{service="acme-gateway", uri!~"/actuator.*"}[1m])))'
)
SERVICE_RATE = (
    'sum by (service) (rate(server_requests_code_total[1m]))'
    ' or sum by (service) (rate(http_server_requests_seconds_count{service!="acme-gateway", uri!~"/actuator.*"}[1m]))'
)
SERVICE_ERRORS = (
    'sum by (service) (rate(server_requests_code_total{code=~"5.."}[1m]))'
    ' or sum by (service) (rate(http_server_requests_seconds_count{service!="acme-gateway", status=~"5.."}[1m]))'
)
SERVICE_P95 = (
    'histogram_quantile(0.95, sum by (le, service) (rate(server_requests_seconds_bucket_bucket[1m])))'
    ' or histogram_quantile(0.95, sum by (le, service) (rate(http_server_requests_seconds_bucket{service!="acme-gateway", uri!~"/actuator.*"}[1m])))'
)
USER_BY_POD = (
    'sum by (pod, gray_tag) (rate(server_requests_code_total{service="user-service"}[1m]))'
    ' or sum by (pod, gray_tag) (rate(http_server_requests_seconds_count{service="user-service", uri!~"/actuator.*"}[1m]))'
)
HEAP = 'sum by (pod) (go_memstats_heap_inuse_bytes{job="acme-services"}) or sum by (pod) (jvm_memory_used_bytes{area="heap"})'


def ts(title, expr, legend, unit="short", desc="", stack=False, w=8, h=8):
    return {
        "type": "timeseries",
        "title": title,
        "description": desc,
        "datasource": PROM,
        "gridPos": {"w": w, "h": h},
        "targets": [{"refId": "A", "expr": expr, "legendFormat": legend, "datasource": PROM}],
        "fieldConfig": {
            "defaults": {
                "unit": unit,
                "custom": {
                    "lineWidth": 2,
                    "fillOpacity": 12,
                    "showPoints": "never",
                    "stacking": {"mode": "normal" if stack else "none"},
                },
            },
            "overrides": [],
        },
        "options": {
            "legend": {"displayMode": "list", "placement": "bottom"},
            "tooltip": {"mode": "multi", "sort": "desc"},
        },
    }


def stat(title, expr, legend, unit="short", desc="", w=4, h=4, thresholds=None):
    return {
        "type": "stat",
        "title": title,
        "description": desc,
        "datasource": PROM,
        "gridPos": {"w": w, "h": h},
        "targets": [{"refId": "A", "expr": expr, "legendFormat": legend, "instant": True, "datasource": PROM}],
        "fieldConfig": {
            "defaults": {
                "unit": unit,
                "thresholds": {
                    "mode": "absolute",
                    "steps": thresholds or [{"color": "green", "value": None}],
                },
            },
            "overrides": [],
        },
        "options": {"colorMode": "value", "graphMode": "area", "reduceOptions": {"calcs": ["lastNotNull"]}},
    }


def row(title):
    return {"type": "row", "title": title, "collapsed": False, "gridPos": {"w": 24, "h": 1}, "panels": []}


panels = [
    row("概况"),
    stat("运行中的实例", 'count(up{job="acme-services"} == 1)', "实例", desc="Prometheus 按 Pod 注解发现并抓取成功的业务实例数"),
    stat("网关 QPS", (
        'sum(rate(http_server_request_duration_seconds_count{service="acme-gateway"}[1m]))'
        ' or sum(rate(http_server_requests_seconds_count{service="acme-gateway", uri!~"/actuator.*"}[1m]))'
    ), "QPS", unit="reqps"),
    stat("网关 5xx 比例", (
        'sum(rate(http_server_request_duration_seconds_count{service="acme-gateway", http_response_status_code=~"5.."}[1m])) / sum(rate(http_server_request_duration_seconds_count{service="acme-gateway"}[1m]))'
        ' or sum(rate(http_server_requests_seconds_count{service="acme-gateway", status=~"5.."}[1m])) / sum(rate(http_server_requests_seconds_count{service="acme-gateway", uri!~"/actuator.*"}[1m]))'
    ), "5xx", unit="percentunit", thresholds=[{"color": "green", "value": None}, {"color": "red", "value": 0.01}]),
    stat("网关 P95", GATEWAY_P95, "P95", unit="s", thresholds=[{"color": "green", "value": None}, {"color": "orange", "value": 0.5}]),
    stat("发件箱积压", "max(acme_outbox_pending) or vector(0)", "待投递", desc="order-service 的 outbox_event 中 status=0 的事件数；持续上涨说明 MQ 不可用（demo.sh outbox）",
         thresholds=[{"color": "green", "value": None}, {"color": "orange", "value": 1}, {"color": "red", "value": 10}]),
    stat("触发中的告警", 'count(ALERTS{alertstate="firing"}) or vector(0)', "告警", desc="Prometheus 告警规则（manifests/observability/prometheus.yaml），详情见 http://localhost:30090/alerts",
         thresholds=[{"color": "green", "value": None}, {"color": "red", "value": 1}]),

    row("入口网关"),
    ts("网关请求速率（按状态码）", GATEWAY_RATE, "{{code}}", unit="reqps", stack=True, w=12,
       desc="429 = 限流（demo.sh ratelimit），503 = 下游不可用 / 熔断（demo.sh breaker）"),
    ts("网关 P95 延迟", GATEWAY_P95, "P95", unit="s", w=12),

    row("业务服务"),
    ts("各服务请求速率", SERVICE_RATE, "{{service}}", unit="reqps", desc="含服务间内部调用（Go：gRPC，Spring：/internal HTTP）"),
    ts("各服务 5xx", SERVICE_ERRORS, "{{service}}", unit="reqps"),
    ts("各服务 P95 延迟", SERVICE_P95, "{{service}}", unit="s"),
    ts("user-service 各实例请求量（负载均衡 / 灰度）", USER_BY_POD, "{{pod}} {{gray_tag}}", unit="reqps", stack=True, w=24,
       desc="稳定版多副本均分流量；部署灰度版（demo.sh gray）后可以看到 gray_tag=v2 的实例只接收白名单用户的流量"),

    row("异步消息（事务发件箱 + MQ）"),
    ts("发件箱积压", "max by (service) (acme_outbox_pending)", "{{service}}", desc="MQ 宕机期间下单照常成功，事件堆积在发件箱；恢复后 Relay 自动补发"),
    ts("事件投递", "sum by (type, result) (rate(acme_outbox_published_total[1m]))", "{{type}} {{result}}", unit="ops"),
    ts("事件消费", "sum by (consumer, result) (rate(acme_mq_consumed_total[1m]))", "{{consumer}} {{result}}", unit="ops",
       desc="duplicate = 重复投递被幂等表拦截；failed = 处理失败（重试后进入死信）"),

    row("资源与运行时"),
    ts("CPU（按 Pod）", 'sum by (pod) (rate(container_cpu_usage_seconds_total{namespace="acme", container="app"}[1m]))', "{{pod}}", unit="cores",
       desc="来自 kubelet cAdvisor；order-service 开启了 HPA（demo.sh load）"),
    ts("内存工作集（按 Pod）", 'sum by (pod) (container_memory_working_set_bytes{namespace="acme", container="app"})', "{{pod}}", unit="bytes",
       desc="对比两套实现：JVM 每副本数百 MiB，Go 每副本数十 MiB"),
    ts("堆内存（Go heap / JVM heap）", HEAP, "{{pod}}", unit="bytes"),
    ts("熔断器状态（Spring：Resilience4j）", 'sum by (service, name, state) (resilience4j_circuitbreaker_state{state=~"open|half_open"}) > 0', "{{service}} {{name}} {{state}}",
       desc="Spring Cloud CircuitBreaker 的状态；Go 版使用 Kratos SRE 自适应熔断，无独立状态指标，可从“各服务 5xx”与网关 503 观察", w=12),
    ts("Pod 重启 / 存活", 'up{job="acme-services"}', "{{pod}}", w=12),
]

# 自动排版：按行从左到右排列
x = y = 0
line_h = 0
for p in panels:
    g = p["gridPos"]
    if p["type"] == "row" or x + g["w"] > 24:
        y += line_h
        x, line_h = 0, 0
    g["x"], g["y"] = x, y
    x += g["w"]
    line_h = max(line_h, g["h"])
    if p["type"] == "row":
        y += 1
        x, line_h = 0, 0

for i, p in enumerate(panels, start=1):
    p["id"] = i

dashboard = {
    "uid": "acme-overview",
    "title": "Acme 微服务总览",
    "tags": ["acme", "local-cluster"],
    "timezone": "browser",
    "schemaVersion": 41,
    "version": 1,
    "refresh": "10s",
    "time": {"from": "now-15m", "to": "now"},
    "links": [
        {"title": "Jaeger 链路", "type": "link", "url": "http://localhost:30686", "targetBlank": True, "icon": "external link"},
        {"title": "Prometheus 告警", "type": "link", "url": "http://localhost:30090/alerts", "targetBlank": True, "icon": "external link"},
        {"title": "Nacos 控制台", "type": "link", "url": "http://localhost:30848", "targetBlank": True, "icon": "external link"},
    ],
    "panels": panels,
}

out = pathlib.Path(__file__).with_name("acme-overview.json")
out.write_text(json.dumps(dashboard, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
print(f"wrote {out}")
