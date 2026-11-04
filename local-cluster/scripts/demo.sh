#!/usr/bin/env bash
# 演示场景：每个场景都会打印“做了什么、看到了什么、去哪里进一步分析”。
#   scripts/demo.sh <spring|go> <场景>
#   scripts/demo.sh go help
source "$(dirname "$0")/lib.sh"

STACK=$(stack_arg "${1:-}")
SCENARIO=${2:-help}
shift 2 || true
need curl

PROM=http://localhost:30090
JAEGER=http://localhost:30686

# ---- 小工具 -------------------------------------------------------------
token() { "$LC_DIR/scripts/token.sh" "$STACK" "$1"; }

# api <令牌> <方法> <路径> [请求体] [额外 curl 参数…] → 输出 “状态码<TAB>响应头 X-Served-By<TAB>响应体”
api() {
  local tok=$1 method=$2 path=$3 body=${4:-}
  shift 4 2>/dev/null || shift $#
  local hdr
  hdr=$(mktemp)
  local args=(-s --noproxy '*' -X "$method" -D "$hdr" -o - -w '\n%{http_code}' -H "Authorization: Bearer $tok")
  [[ -n "$body" ]] && args+=(-H 'Content-Type: application/json' -d "$body")
  local out code served
  out=$(curl "${args[@]}" "$@" "$GATEWAY_URL$path")
  code=${out##*$'\n'}
  served=$(tr -d '\r' <"$hdr" | sed -n 's/^[Xx]-[Ss]erved-[Bb]y: //p')
  rm -f "$hdr"
  printf '%s\t%s\t%s\n' "$code" "$served" "$(tr -d '\r\n' <<<"${out%$'\n'*}")"
}

field() { sed -n "s/.*\"$1\" *: *\"\{0,1\}\([^\",}]*\)\"\{0,1\}.*/\1/p" | head -n1; }

prom() { # 即时查询，输出第一个结果的值
  curl -s --noproxy '*' -G "$PROM/api/v1/query" --data-urlencode "query=$1" |
    sed -n 's/.*"value":\[[^,]*,"\([^"]*\)"\].*/\1/p' | head -n1
}

# 金额用字符串传递（两套实现都接受），避免浮点误差
order_json() { printf '{"productCode":"SKU-%s","quantity":%s,"amount":"%s"}' "$1" "${2:-1}" "${3:-99.90}"; }

place_orders() { # place_orders <令牌> <数量> → 统计状态码
  local tok=$1 n=$2 i
  for ((i = 1; i <= n; i++)); do
    api "$tok" POST /api/v1/orders "$(order_json "$i")" | cut -f1
  done | sort | uniq -c | sed 's/^/      /'
}

nacos_publish() { # nacos_publish <dataId> （内容来自 stdin）
  local data_id=$1
  k -n "$NS_MW" exec -i nacos-0 -- sh -c '
    content=$(cat)
    T=$(curl -s -X POST localhost:8848/nacos/v3/auth/user/login -d username=nacos -d "password=$NACOS_ADMIN_PASSWORD" | sed -n "s/.*\"accessToken\":\"\([^\"]*\)\".*/\1/p")
    curl -sf -X POST localhost:8848/nacos/v3/admin/cs/config -H "accessToken: $T" \
      --data-urlencode "dataId='"$data_id"'" -d groupName=DEFAULT_GROUP -d namespaceId=local -d type=yaml \
      --data-urlencode "content=$content" >/dev/null' && ok "Nacos 配置已发布：local/DEFAULT_GROUP/$data_id"
}

restore_nacos() { nacos_publish "$1" <"$LC_DIR/stacks/$STACK/nacos/$1"; }

# 通过 Nacos 修改网关限流（每用户每秒令牌数 / 桶容量），网关实时生效
set_rate_limit() {
  local rate=$1 burst=$2
  if [[ $STACK == go ]]; then
    sed -e "s/^    perSecond: 50/    perSecond: $rate/" -e "s/^    burst: 100/    burst: $burst/" \
      "$LC_DIR/stacks/go/nacos/acme-gateway.yaml" | nacos_publish acme-gateway.yaml
  else
    { cat "$LC_DIR/stacks/spring/nacos/acme-gateway.yaml"; cat <<EOF2
spring:
  cloud:
    gateway:
      server:
        webflux:
          default-filters:
            - name: RequestRateLimiter
              args:
                key-resolver: "#{@principalOrIpKeyResolver}"
                redis-rate-limiter.replenishRate: $rate
                redis-rate-limiter.burstCapacity: $burst
            - DedupeResponseHeader=Access-Control-Allow-Origin Access-Control-Allow-Credentials
EOF2
    } | nacos_publish acme-gateway.yaml
  fi
}

jaeger_link() { echo "      链路：$JAEGER/trace/$1" >&2; }

# 打印一条链路中的所有 Span：耗时、服务、处理实例（Pod / 灰度标签）、操作名。
#   trace_summary <traceId> <至少包含的服务数>    需要 python3，否则只打印链接
trace_summary() {
  local id=$1 i
  jaeger_link "$id"
  command -v python3 >/dev/null || return 0
  sleep 3 # 入口 Span 最后结束、最后上报
  for ((i = 0; i < 10; i++)); do # Span 批量上报，稍等片刻
    sleep 2
    curl -s --noproxy '*' "$JAEGER/api/traces/$id" | python3 -c '
import json, sys
try:
    trace = json.load(sys.stdin)["data"][0]
except Exception:
    sys.exit(1)
procs = {}
for pid, p in trace["processes"].items():
    tags = {t["key"]: t["value"] for t in p.get("tags", [])}
    gray = tags.get("acme.gray_tag", "")
    procs[pid] = (p["serviceName"], tags.get("k8s.pod.name", "") + ("" if gray in ("", "stable") else " gray=" + gray))
spans = sorted(trace["spans"], key=lambda s: s["startTime"])
if len({procs[s["processID"]][0] for s in spans}) < int(sys.argv[1]):
    sys.exit(1)  # 各服务的 Span 分批上报，等齐了再打印
t0 = spans[0]["startTime"]
for s in spans:
    svc, pod = procs[s["processID"]]
    print("      %7.1fms +%6.1fms  %-21s %-44s %s" % ((s["startTime"] - t0) / 1000, s["duration"] / 1000, svc, pod, s["operationName"]))
' "${2:-1}" >&2 && return 0
  done
}

# ---- 场景 ---------------------------------------------------------------
scenario_smoke() {
  step "冒烟：登录 → 下单 → 查询 → 异步通知"
  local tok res code body order_id trace
  tok=$(token alice)
  ok "alice 通过授权码 + PKCE 登录，获得 JWT"
  res=$(api "$tok" POST /api/v1/orders "$(order_json 1 2 199.80)")
  code=$(cut -f1 <<<"$res"); body=$(cut -f3- <<<"$res")
  [[ $code == 201 || $code == 200 ]] || die "下单失败 HTTP $code：$body"
  order_id=$(field id <<<"$body"); trace=$(field traceId <<<"$body")
  ok "下单成功 HTTP $code，订单 $order_id（$(cut -f2 <<<"$res")）"
  res=$(api "$tok" GET "/api/v1/orders/$order_id")
  [[ $(cut -f1 <<<"$res") == 200 ]] || die "查询订单失败：$res"
  ok "查询订单 HTTP 200"
  local i
  for ((i = 0; i < 30; i++)); do
    res=$(api "$tok" GET "/api/v1/notifications")
    grep -q "$(field orderNo <<<"$body")" <<<"$res" && break
    sleep 1
  done
  grep -q "$(field orderNo <<<"$body")" <<<"$res" || die "30 秒内未收到下单通知（检查 MQ 与 notification-service 日志）"
  ok "notification-service 已消费 order.created 并生成通知（发件箱 → $MQ → 幂等消费）"
  log "整条链路（同步调用 + 发件箱 + MQ 是同一个 Trace）"
  trace_summary "$trace" 4
}

scenario_lb() {
  step "负载均衡：同一个请求连续发 10 次，观察由哪些实例处理"
  local tok
  tok=$(token alice)
  for _ in $(seq 10); do api "$tok" GET /api/v1/users/1001 | cut -f2; done | sort | uniq -c | sed 's/^/      /'
  echo "      user-service 有多个副本时请求被轮询分配；实例来自 Nacos 注册表，而非 K8s Service" >&2
}

scenario_gray() {
  step "灰度发布：部署 user-service v2 → Nacos 下发灰度规则（alice 进灰度）→ 验证 → 回滚"
  local tag
  tag=$(cat "$LC_DIR/.image-tag-$STACK")
  log "部署灰度版本 user-service-v2（grayTag=v2，复用稳定版 Service）"
  helm --kube-context "$KUBE_CONTEXT" upgrade --install user-service-v2 \
    "$REPO_DIR/acme-$([[ $STACK == spring ]] && echo cloud || echo go)/deploy/helm/acme-service" -n "$NS_APP" \
    -f "$LC_DIR/stacks/$STACK/values/_common.yaml" -f "$LC_DIR/stacks/$STACK/values/user-service-v2.yaml" \
    --set image.tag="$tag" >/dev/null
  wait_rollout "$NS_APP" deployment user-service-v2 420s
  sleep 5 # 等待注册到 Nacos、网关刷新实例列表

  local alice bob
  alice=$(token alice); bob=$(token bob)
  log "灰度规则关闭时：alice 的请求"
  for _ in $(seq 6); do api "$alice" GET /api/v1/users/1001 | cut -f2; done | sort | uniq -c | sed 's/^/      /'

  log "Nacos 下发灰度规则：uids=[1001]（alice）"
  if [[ $STACK == go ]]; then
    sed -e 's/^    enabled: false/    enabled: true/' -e 's/^    uids: \[\]/    uids: ["1001"]/' \
      "$LC_DIR/stacks/go/nacos/acme-gateway.yaml" | nacos_publish acme-gateway.yaml
  else
    sed -e 's/^      enabled: false/      enabled: true/' -e 's/^      uids: \[\]/      uids: ["1001"]/' \
      "$LC_DIR/stacks/spring/nacos/acme-gateway.yaml" | nacos_publish acme-gateway.yaml
  fi
  sleep 5
  log "alice（白名单用户）→ 应全部落到 gray=v2 实例"
  for _ in $(seq 6); do api "$alice" GET /api/v1/users/1001 | cut -f2; done | sort | uniq -c | sed 's/^/      /'
  log "bob（非白名单）→ 应全部落到稳定版实例"
  for _ in $(seq 6); do api "$bob" GET /api/v1/users/1002 | cut -f2; done | sort | uniq -c | sed 's/^/      /'
  log "alice 下单：order-service 没有灰度实例 → 走稳定版；灰度标签随调用链透传，order-service 调 user-service 时选中 v2"
  local res
  res=$(api "$alice" POST /api/v1/orders "$(order_json gray)")
  echo "      alice 下单 HTTP $(cut -f1 <<<"$res")" >&2
  trace_summary "$(cut -f3- <<<"$res" | field traceId)" 3
  log "测试同学可用请求头 X-Gray-Tag: v2 指定版本（allowClientHeader=true，仅 test/local）"
  for _ in $(seq 3); do api "$bob" GET /api/v1/users/1002 "" -H 'X-Gray-Tag: v2' | cut -f2; done | sort | uniq -c | sed 's/^/      /'

  if [[ "${KEEP:-}" == 1 ]]; then
    warn "KEEP=1：保留灰度版本与规则"
  else
    log "回滚：恢复 Nacos 规则、卸载灰度版本"
    restore_nacos acme-gateway.yaml
    helm --kube-context "$KUBE_CONTEXT" -n "$NS_APP" uninstall user-service-v2 --wait >/dev/null
    ok "已回滚"
  fi
}

scenario_breaker() {
  step "熔断与降级：user-service 全部下线时，下单快速失败，不拖垮 order-service"
  local tok
  tok=$(token alice)
  log "停止 user-service（scale 0）"
  kapp scale deployment user-service --replicas=0 >/dev/null
  kapp delete deployment user-service-v2 --ignore-not-found >/dev/null
  kapp wait --for=delete pod -l app.kubernetes.io/name=user-service --timeout=120s >/dev/null 2>&1 || true
  sleep 5
  log "连续下单 30 次（状态码统计）"
  local start end
  start=$(date +%s)
  place_orders "$tok" 30
  end=$(date +%s)
  echo "      30 次请求耗时 $((end - start)) 秒：依赖不可用时快速失败（错误码 C0503），而不是等待超时" >&2
  if [[ $STACK == spring ]]; then
    echo "      熔断器状态：$(prom 'max(resilience4j_circuitbreaker_state{state="open",service="order-service"})')（1 = OPEN），Prometheus Alerts 页可看到 CircuitBreakerOpen" >&2
  fi
  log "恢复 user-service"
  kapp scale deployment user-service --replicas=2 >/dev/null
  wait_rollout "$NS_APP" deployment user-service 420s
  sleep 10
  log "恢复后下单 5 次"
  place_orders "$tok" 5
}

scenario_outbox() {
  step "事务发件箱：消息中间件宕机期间下单不受影响，恢复后事件自动补发"
  local tok mq_sts
  tok=$(token alice)
  mq_sts=$([[ $MQ == kafka ]] && echo kafka || echo rabbitmq)
  log "停止 $mq_sts"
  k -n "$NS_MW" scale statefulset "$mq_sts" --replicas=0 >/dev/null
  k -n "$NS_MW" wait --for=delete pod "$mq_sts-0" --timeout=120s >/dev/null 2>&1 || true
  log "MQ 不可用期间下单 5 次 —— 订单与事件在同一本地事务中写库，下单照常成功"
  place_orders "$tok" 5
  sleep 15
  echo "      发件箱积压 acme_outbox_pending = $(prom 'max(acme_outbox_pending)')（Grafana“发件箱积压”面板 / Prometheus Alerts: OutboxBacklog）" >&2
  log "恢复 $mq_sts"
  k -n "$NS_MW" scale statefulset "$mq_sts" --replicas=1 >/dev/null
  wait_rollout "$NS_MW" statefulset "$mq_sts" 300s
  local i pending=""
  for ((i = 0; i < 40; i++)); do
    sleep 5
    pending=$(prom 'max(acme_outbox_pending)')
    [[ "$pending" == 0 ]] && break
  done
  if [[ "$pending" == 0 ]]; then
    ok "积压已清零：Relay 自动补发，notification-service 幂等消费（重复投递会被 consumed_event 表拦截）"
  else
    warn "积压尚未清零（当前 $pending），Relay 会继续重试；可稍后在 Grafana 中观察"
  fi
}

scenario_ratelimit() {
  step "动态配置：通过 Nacos 把网关限流调低，立即生效（无需重启）"
  local tok
  tok=$(token alice)
  set_rate_limit 2 2
  sleep 5
  log "并发 20 个请求（状态码统计，429 = 被限流）"
  seq 20 | xargs -P 10 -I{} curl -s --noproxy '*' -o /dev/null -w '%{http_code}\n' \
    -H "Authorization: Bearer $tok" "$GATEWAY_URL/api/v1/users/1001" | sort | uniq -c | sed 's/^/      /'
  log "恢复限流配置"
  restore_nacos acme-gateway.yaml
}

scenario_rollout() {
  step "滚动发布零中断：持续请求的同时重启 user-service"
  local tok
  tok=$(token alice)
  local out
  out=$(mktemp)
  ( end=$((SECONDS + 60))
    while ((SECONDS < end)); do
      curl -s --noproxy '*' -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $tok" \
        "$GATEWAY_URL/api/v1/users/1001" >>"$out"
      sleep 0.1
    done ) &
  local bg=$!
  sleep 3
  log "kubectl rollout restart deployment/user-service"
  kapp rollout restart deployment/user-service >/dev/null
  wait_rollout "$NS_APP" deployment user-service 420s
  wait "$bg" || true
  log "60 秒内的请求结果"
  sort "$out" | uniq -c | sed 's/^/      /'
  rm -f "$out"
  echo "      preStop 等待 + 注册中心摘除 + 优雅停机：发布期间没有 5xx 即为零中断" >&2
}

scenario_load() {
  local seconds=${1:-120} concurrency=${2:-4}
  step "压测：$concurrency 并发持续 $seconds 秒（下单 + 查询），用于观察 Grafana 看板与 HPA"
  local alice bob
  alice=$(token alice); bob=$(token bob)
  log "压测期间通过 Nacos 临时放宽网关限流（结束后恢复）"
  set_rate_limit 2000 4000
  sleep 5
  worker() {
    local tok=$1 end=$((SECONDS + seconds))
    while ((SECONDS < end)); do
      curl -s --noproxy '*' -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $tok" \
        -H 'Content-Type: application/json' -d "$(order_json load)" "$GATEWAY_URL/api/v1/orders"
      curl -s --noproxy '*' -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $tok" "$GATEWAY_URL/api/v1/users/1001"
    done
  }
  local out pids=() i
  out=$(mktemp)
  for ((i = 0; i < concurrency; i++)); do
    if ((i % 2)); then worker "$bob" >>"$out" & else worker "$alice" >>"$out" & fi
    pids+=($!)
  done
  for ((i = 0; i < seconds; i += 15)); do
    sleep 15
    local hpa
    hpa=$(kapp get hpa order-service -o jsonpath='{.status.currentReplicas}/{.status.desiredReplicas} 副本，CPU {.status.currentMetrics[0].resource.current.averageUtilization}%' 2>/dev/null)
    [[ "$hpa" == *"CPU %"* ]] && hpa="${hpa%%，*}（CPU 指标不可用：检查 kubectl top pods / metrics-server）"
    echo "      $(date +%T)  已发送 $(wc -l <"$out") 个请求  order-service HPA：$hpa" >&2
  done
  wait "${pids[@]}" 2>/dev/null || true
  sort "$out" | uniq -c | sed 's/^/      /'
  rm -f "$out"
  restore_nacos acme-gateway.yaml
}

scenario_all() {
  scenario_smoke
  scenario_lb
  scenario_gray
  scenario_ratelimit
  scenario_outbox
  scenario_breaker
  scenario_rollout
}

case "$SCENARIO" in
  smoke | lb | gray | breaker | outbox | ratelimit | rollout | load | all) "scenario_$SCENARIO" "$@" ;;
  *)
    cat >&2 <<EOF
用法：scripts/demo.sh <spring|go> <场景>

  smoke      登录（授权码 + PKCE）→ 下单 → 查询 → 异步通知，输出 Jaeger 链路地址
  lb         负载均衡：观察请求在多个实例间的分配（响应头 X-Served-By）
  gray       灰度发布：部署 v2 → Nacos 下发灰度规则 → 按用户 / 请求头路由 → 回滚（KEEP=1 保留）
  ratelimit  动态配置：Nacos 调低网关限流，立即出现 429，随后恢复
  outbox     事务发件箱：停掉 MQ 照常下单，积压告警，恢复后自动补发
  breaker    熔断降级：停掉 user-service，下单快速失败，恢复后自动恢复
  rollout    滚动发布零中断：持续请求时重启 user-service
  load [秒] [并发]  压测，观察 Grafana 看板与 order-service 的 HPA 扩容
  all        依次运行除 load 以外的所有场景
EOF
    ;;
esac
