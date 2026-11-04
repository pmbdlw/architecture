#!/usr/bin/env bash
# 一键在本地集群中部署一套完整的微服务体系。
#   scripts/up.sh go                     # Go 版：kind 集群 + 中间件 + 可观测 + 5 个服务
#   scripts/up.sh spring                 # Spring Cloud 版
#   DB=postgres MQ=rabbitmq scripts/up.sh go
#   scripts/up.sh go apps                # 只重新构建并发布业务服务（改代码后）
# 切换技术栈时会卸载另一套服务并重建业务库（两套实现的迁移工具不同：Flyway / goose）。
source "$(dirname "$0")/lib.sh"

STACK=$(stack_arg "${1:-}")
PHASE=${2:-all}
CHART=$REPO_DIR/acme-$([[ $STACK == spring ]] && echo cloud || echo go)/deploy/helm/acme-service
STACK_DIR=$LC_DIR/stacks/$STACK
M=$LC_DIR/manifests

# ---------------------------------------------------------------------------
cluster_up() {
  step "1/6 集群"
  if [[ "$CLUSTER_PROVIDER" == "kind" ]]; then
    "$LC_DIR/scripts/cluster.sh" create
  else
    log "使用当前上下文 $KUBE_CONTEXT"
  fi
  "$LC_DIR/scripts/images.sh"
  k apply -f "$M/00-namespaces.yaml" >/dev/null
}

infra_up() {
  step "2/6 中间件（DB=$DB MQ=$MQ）"
  k apply -f "$M/middleware/$DB.yaml" -f "$M/middleware/redis.yaml" -f "$M/middleware/nacos.yaml" >/dev/null
  if [[ $MQ == kafka ]]; then
    k apply -f "$M/middleware/kafka.yaml" >/dev/null
  else
    k apply -f "$M/middleware/rabbitmq.yaml" >/dev/null
  fi

  step "3/6 可观测（Prometheus / Grafana / Jaeger / metrics-server）"
  local changed
  changed=$(k -n "$NS_OBS" create configmap grafana-dashboards --from-file="$LC_DIR/dashboards/acme-overview.json" \
    --dry-run=client -o yaml | k apply -f -; k apply -f "$M/observability/")
  # ConfigMap 变更不会自动生效：配置或看板有改动时滚动重启
  if grep -q "configmap/prometheus-config configured" <<<"$changed"; then k -n "$NS_OBS" rollout restart deployment/prometheus >/dev/null; fi
  if grep -q "configmap/grafana-.* configured" <<<"$changed"; then k -n "$NS_OBS" rollout restart deployment/grafana >/dev/null; fi

  wait_rollout "$NS_MW" statefulset "$DB"
  wait_rollout "$NS_MW" deployment redis
  if [[ $MQ == kafka ]]; then
    wait_rollout "$NS_MW" statefulset kafka
    wait_job "$NS_MW" kafka-topics
  else
    wait_rollout "$NS_MW" statefulset rabbitmq 300s
  fi
  wait_rollout "$NS_MW" statefulset nacos 300s
  for d in jaeger prometheus grafana; do wait_rollout "$NS_OBS" deployment $d; done
}

# ---------------------------------------------------------------------------
# 切换技术栈：卸载另一套服务、重建业务库（保留中间件与可观测数据）
switch_stack_if_needed() {
  local current
  current=$(k -n "$NS_APP" get configmap acme-stack -o jsonpath='{.data.stack}' 2>/dev/null || true)
  if [[ -n "$current" && "$current" != "$STACK" ]]; then
    warn "集群中当前部署的是 $current 版，切换到 $STACK 版：卸载服务并重建业务库"
    "$LC_DIR/scripts/down.sh" apps
    reset_databases
  fi
}

reset_databases() {
  log "重建业务库（$DB）"
  if [[ $DB == mysql ]]; then
    k -n "$NS_MW" exec mysql-0 -- sh -c \
      'mysql -uroot -proot -e "DROP DATABASE IF EXISTS acme_user; DROP DATABASE IF EXISTS acme_order; DROP DATABASE IF EXISTS acme_notification;" && mysql -uroot -proot < /docker-entrypoint-initdb.d/init.sql' 2>/dev/null
  else
    k -n "$NS_MW" exec postgres-0 -- sh -c \
      'for d in acme_user acme_order acme_notification; do psql -U postgres -c "DROP DATABASE IF EXISTS $d WITH (FORCE)"; done; for u in user_svc order_svc notification_svc; do psql -U postgres -c "DROP ROLE IF EXISTS $u"; done; psql -U postgres -f /docker-entrypoint-initdb.d/init.sql' >/dev/null
  fi
}

# 平台公共环境变量（ConfigMap acme-platform-env）与各服务 Secret。
# 生产中前者由平台团队维护（deploy/k8s/platform），后者由 ExternalSecret 从密钥系统同步；这里直接生成。
platform_config() {
  step "4/6 平台配置（Nacos 配置发布、ConfigMap、Secret）"
  k -n "$NS_MW" create configmap nacos-configs --from-file="$STACK_DIR/nacos" \
    --dry-run=client -o yaml | k apply -f - >/dev/null
  k -n "$NS_MW" delete job nacos-init --ignore-not-found >/dev/null
  k apply -f "$M/jobs/nacos-init.yaml" >/dev/null
  wait_job "$NS_MW" nacos-init

  local mw=$NS_MW.svc.cluster.local
  local common=(
    --from-literal=NACOS_SERVER_ADDR=nacos.$mw:8848
    --from-literal=OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://jaeger.$NS_OBS.svc.cluster.local:4318/v1/traces
    # iss 声明：从宿主机获取令牌，因此与宿主机访问认证中心的地址一致
    --from-literal=AUTH_ISSUER_URI=$AUTH_URL
    --from-literal=AUTH_JWK_SET_URI=http://acme-auth.$NS_APP.svc.cluster.local/oauth2/jwks
    # 本地开启：响应头 X-Served-By 显示处理实例，观察负载均衡与灰度
    --from-literal=SERVED_BY_HEADER=true
    --from-literal=GRAY_ALLOW_CLIENT_HEADER=true
  )
  if [[ $STACK == go ]]; then
    common+=(
      --from-literal=DB_DRIVER=$DB
      --from-literal=MQ_DRIVER=$MQ
      --from-literal=KAFKA_BROKERS=kafka.$mw:9092
      --from-literal=REDIS_ADDR=redis.$mw:6379
    )
  else
    common+=(
      --from-literal=KAFKA_BROKERS=kafka.$mw:9092
      --from-literal=RABBITMQ_ADDRESSES=rabbitmq.$mw:5672
      --from-literal=REDIS_HOST=redis.$mw
      --from-literal=REDIS_PORT=6379
    )
  fi
  k -n "$NS_APP" create configmap acme-platform-env "${common[@]}" --dry-run=client -o yaml | k apply -f - >/dev/null

  # 认证中心签名密钥：只在首次生成，之后复用（生产由 Vault / KMS 管理并定期轮换）
  if ! k -n "$NS_APP" get secret acme-auth-jwk >/dev/null 2>&1; then
    need openssl
    local keydir
    keydir=$(mktemp -d)
    openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$keydir/private.pem" 2>/dev/null
    openssl pkey -in "$keydir/private.pem" -pubout -out "$keydir/public.pem"
    k -n "$NS_APP" create secret generic acme-auth-jwk \
      --from-file="$keydir/private.pem" --from-file="$keydir/public.pem" >/dev/null
    rm -rf "$keydir"
    ok "已生成认证中心签名密钥 Secret acme-auth-jwk"
  fi

  local nacos_pwd=local-only
  secret acme-gateway-secrets NACOS_PASSWORD=$nacos_pwd REDIS_PASSWORD=
  secret acme-auth-secrets NACOS_PASSWORD=$nacos_pwd AUTH_INTERNAL_CLIENT_SECRET=local-only-secret AUTH_DEMO_PASSWORD=password
  for svc in user order notification; do
    local db=acme_$svc user=${svc}_svc
    if [[ $STACK == go ]]; then
      local dsn
      if [[ $DB == mysql ]]; then
        dsn="$user:local-only@tcp(mysql.$mw:3306)/$db?parseTime=true&loc=UTC&charset=utf8mb4"
      else
        dsn="postgres://$user:local-only@postgres.$mw:5432/$db?sslmode=disable"
      fi
      secret "$svc-service-secrets" NACOS_PASSWORD=$nacos_pwd "DB_DSN=$dsn" \
        "RABBITMQ_URL=amqp://acme:local-only@rabbitmq.$mw:5672/"
    else
      local url
      if [[ $DB == mysql ]]; then
        url="jdbc:mysql://mysql.$mw:3306/$db?useUnicode=true&characterEncoding=utf8&serverTimezone=Asia/Shanghai"
      else
        url="jdbc:postgresql://postgres.$mw:5432/$db"
      fi
      secret "$svc-service-secrets" NACOS_PASSWORD=$nacos_pwd "DB_URL=$url" DB_USERNAME=$user DB_PASSWORD=local-only \
        RABBITMQ_USERNAME=acme RABBITMQ_PASSWORD=local-only
    fi
  done
  k -n "$NS_APP" create configmap acme-stack --from-literal=stack="$STACK" --from-literal=db="$DB" --from-literal=mq="$MQ" \
    --dry-run=client -o yaml | k apply -f - >/dev/null
  ok "acme-platform-env 与各服务 Secret 已生成"
}

secret() {
  local name=$1; shift
  local args=()
  for kv in "$@"; do args+=(--from-literal="$kv"); done
  k -n "$NS_APP" create secret generic "$name" "${args[@]}" --dry-run=client -o yaml | k apply -f - >/dev/null
}

# ---------------------------------------------------------------------------
apps_up() {
  step "5/6 构建镜像并发布服务（Helm）"
  "$LC_DIR/scripts/build.sh" "$STACK"
  local tag
  tag=$(cat "$LC_DIR/.image-tag-$STACK")
  # 发布顺序与生产一致：认证中心 → 被依赖的服务 → 依赖方 → 网关
  for svc in acme-auth user-service order-service notification-service acme-gateway; do
    helm_release "$svc" "$tag"
  done
  for svc in acme-auth user-service order-service notification-service acme-gateway; do
    wait_rollout "$NS_APP" deployment "$svc" 420s
  done
  # 灰度版本若已部署，同步升级到新镜像
  if helm --kube-context "$KUBE_CONTEXT" -n "$NS_APP" status user-service-v2 >/dev/null 2>&1; then
    helm_release user-service-v2 "$tag"
  fi
}

helm_release() {
  local release=$1 tag=$2
  log "helm upgrade --install $release（$tag）"
  helm --kube-context "$KUBE_CONTEXT" upgrade --install "$release" "$CHART" -n "$NS_APP" \
    -f "$STACK_DIR/values/_common.yaml" -f "$STACK_DIR/values/$release.yaml" \
    --set image.tag="$tag" >/dev/null
}

seed_and_check() {
  step "6/6 演示数据与冒烟验证"
  k -n "$NS_MW" delete job seed-users --ignore-not-found >/dev/null
  k apply -f "$M/jobs/seed-users-$DB.yaml" >/dev/null
  wait_job "$NS_MW" seed-users
  "$LC_DIR/scripts/demo.sh" "$STACK" smoke
}

summary() {
  cat >&2 <<EOF

$(printf '\033[1;32m')本地集群已就绪（$STACK 版，DB=$DB，MQ=$MQ）$(printf '\033[0m')
  API 网关      $GATEWAY_URL
  认证中心      $AUTH_URL/.well-known/openid-configuration
  Grafana       http://localhost:30300   （看板：Acme / $([[ $STACK == spring ]] && echo "Spring Cloud" || echo Go) 服务概览）
  Jaeger        http://localhost:30686
  Prometheus    http://localhost:30090/alerts
  Nacos 控制台  http://localhost:30848   （nacos / local-only，命名空间 local）$([[ $MQ == rabbitmq ]] && printf '\n  RabbitMQ      http://localhost:30672   （acme / local-only）')

  演示场景      scripts/demo.sh $STACK help
  获取令牌      scripts/token.sh $STACK alice
EOF
}

need kubectl helm docker
case "$PHASE" in
  all)
    cluster_up
    infra_up
    switch_stack_if_needed
    platform_config
    apps_up
    seed_and_check
    summary
    ;;
  infra)
    cluster_up
    infra_up
    ;;
  apps)
    switch_stack_if_needed
    platform_config
    apps_up
    seed_and_check
    summary
    ;;
  *)
    die "未知阶段 $PHASE（all | infra | apps）"
    ;;
esac
