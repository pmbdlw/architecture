#!/usr/bin/env bash
# 拉取中间件 / 可观测组件镜像并加载进 kind 节点。
# 集群节点不直接访问镜像仓库：先在宿主机拉取（可走 IMAGE_MIRROR 加速），再 kind load，离线也能反复重建集群。
#   scripts/images.sh            按 DB / MQ 选择加载所需镜像
#   scripts/images.sh list       只打印镜像清单
source "$(dirname "$0")/lib.sh"

images() {
  # 版本与 acme-cloud/deploy/local/docker-compose.yml、acme-go/deploy/local/docker-compose.yml 保持一致
  echo "nacos/nacos-server:v3.1.1"
  echo "redis:8.2"
  echo "curlimages/curl:8.16.0"
  case "$DB" in
    mysql) echo "mysql:8.4" ;;
    postgres) echo "postgres:17" ;;
    *) die "不支持的 DB=$DB（mysql | postgres）" ;;
  esac
  case "$MQ" in
    kafka) echo "apache/kafka:4.1.0" ;;
    rabbitmq) echo "rabbitmq:4.1-management" ;;
    *) die "不支持的 MQ=$MQ（kafka | rabbitmq）" ;;
  esac
  echo "jaegertracing/jaeger:2.11.0"
  echo "prom/prometheus:v3.5.0"
  echo "grafana/grafana:12.2.0"
  echo "registry.k8s.io/metrics-server/metrics-server:v0.8.0"
}

if [[ "${1:-}" == "list" ]]; then images; exit 0; fi

need docker
mapfile -t list < <(images)
for img in "${list[@]}"; do pull "$img"; done
log "加载 ${#list[@]} 个中间件镜像到集群节点"
load_images "${list[@]}"
ok "中间件镜像就绪"
