# shellcheck shell=bash disable=SC2034
# （SC2034：这里定义的变量供 source 本文件的脚本使用）
# 公共变量与函数，由其他脚本 source。所有参数都可以用环境变量覆盖。
set -euo pipefail

LC_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
REPO_DIR=$(cd "$LC_DIR/.." && pwd)

# ---- 可调参数 ----------------------------------------------------------
CLUSTER_NAME=${CLUSTER_NAME:-acme}
# kind：创建本地多节点集群（默认）；current：使用当前 kubectl 上下文（Docker Desktop / minikube / 远端测试集群）
CLUSTER_PROVIDER=${CLUSTER_PROVIDER:-kind}
KIND_WORKERS=${KIND_WORKERS:-3}
KIND_NODE_IMAGE=${KIND_NODE_IMAGE:-kindest/node:v1.34.0}
# 镜像加速：docker.io 拉取受限时设置，如 IMAGE_MIRROR=mirror.gcr.io 或企业 Harbor 代理地址
IMAGE_MIRROR=${IMAGE_MIRROR:-}
DB=${DB:-mysql}          # mysql | postgres
REPLICAS=${REPLICAS:-}   # 覆盖每个服务的副本数（默认按 stacks/<stack>/values，多为 2）
MQ=${MQ:-kafka}          # kafka | rabbitmq
NS_APP=acme
NS_MW=middleware
NS_OBS=observability

# 宿主机访问入口（kind extraPortMappings → NodePort）
GATEWAY_URL=${GATEWAY_URL:-http://localhost:30080}
AUTH_URL=${AUTH_URL:-http://localhost:30900}

if [[ "$CLUSTER_PROVIDER" == "kind" ]]; then
  KUBE_CONTEXT=${KUBE_CONTEXT:-kind-$CLUSTER_NAME}
else
  KUBE_CONTEXT=${KUBE_CONTEXT:-$(kubectl config current-context)}
fi

# ---- 输出 --------------------------------------------------------------
log()  { printf '\033[1;34m==>\033[0m %s\n' "$*" >&2; }
ok()   { printf '\033[1;32m ✓\033[0m %s\n' "$*" >&2; }
warn() { printf '\033[1;33m !\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m ✗\033[0m %s\n' "$*" >&2; exit 1; }
step() { printf '\n\033[1;35m━━ %s\033[0m\n' "$*" >&2; }

need() {
  for c in "$@"; do command -v "$c" >/dev/null 2>&1 || die "缺少命令 $c，请先安装（见 local-cluster/README.md 的“前置条件”）"; done
}

k()    { kubectl --context "$KUBE_CONTEXT" "$@"; }
kapp() { k -n "$NS_APP" "$@"; }

# 校验技术栈参数：spring | go
stack_arg() {
  local s=${1:-}
  [[ "$s" == "spring" || "$s" == "go" ]] || die "第一个参数必须是技术栈：spring 或 go"
  echo "$s"
}

# 等待 Deployment / StatefulSet 就绪
wait_rollout() {
  local ns=$1 kind=$2 name=$3 timeout=${4:-300s}
  k -n "$ns" rollout status "$kind/$name" --timeout="$timeout" >/dev/null || {
    warn "$kind/$name 未在 $timeout 内就绪，最近事件："
    k -n "$ns" get events --sort-by=.lastTimestamp | tail -n 15 >&2
    k -n "$ns" logs "$kind/$name" --tail=40 >&2 || true
    return 1
  }
  ok "$ns/$name 就绪"
}

wait_job() {
  local ns=$1 name=$2 timeout=${3:-300s}
  k -n "$ns" wait --for=condition=complete "job/$name" --timeout="$timeout" >/dev/null || {
    k -n "$ns" logs "job/$name" --tail=40 >&2 || true
    die "job/$name 未完成"
  }
  ok "job $ns/$name 完成"
}

# 镜像 tag：Git 短 SHA；工作区有改动时追加时间戳，保证每次构建都触发滚动更新
image_tag() {
  if [[ -n "${TAG:-}" ]]; then echo "$TAG"; return; fi
  local sha
  sha=$(git -C "$REPO_DIR" rev-parse --short HEAD 2>/dev/null || echo dev)
  if [[ -n "$(git -C "$REPO_DIR" status --porcelain 2>/dev/null)" ]]; then
    sha="$sha-$(date +%m%d%H%M%S)"
  fi
  echo "$sha"
}

# Docker Hub 镜像走 IMAGE_MIRROR；registry.k8s.io / gcr.io 等带域名的镜像直接拉取
pull() {
  local img=$1 src=$1
  if docker image inspect "$img" >/dev/null 2>&1; then return; fi
  local first=${img%%/*}
  if [[ -n "$IMAGE_MIRROR" && "$first" != *.* ]]; then
    if [[ "$img" == */* ]]; then src="$IMAGE_MIRROR/$img"; else src="$IMAGE_MIRROR/library/$img"; fi
  fi
  log "docker pull $src"
  docker pull -q "$src" >/dev/null
  [[ "$src" == "$img" ]] || docker tag "$src" "$img"
}

# docker.io 镜像在 containerd 中的完整名称：mysql:8.4 → docker.io/library/mysql:8.4
normalize_image() {
  local img=$1 first=${1%%/*}
  if [[ "$img" != */* ]]; then echo "docker.io/library/$img"
  elif [[ "$first" != *.* && "$first" != localhost* ]]; then echo "docker.io/$img"
  else echo "$img"; fi
}

# 把宿主机镜像加载进集群的工作节点。
# 逐个按当前平台导出再导入：Docker 使用 containerd 镜像存储时，本地只有单一平台的层，
# 直接 kind load docker-image 会因缺少其他平台的内容而失败。
load_images() {
  if [[ "$CLUSTER_PROVIDER" != "kind" ]]; then
    warn "CLUSTER_PROVIDER=$CLUSTER_PROVIDER：跳过镜像加载（Docker Desktop 可直接使用本机镜像；minikube 用 minikube image load）"
    return
  fi
  local platform tar img nodes
  platform=$(docker version -f '{{.Server.Os}}/{{.Server.Arch}}')
  # 只加载到工作节点：控制面有污点不运行业务 Pod，少存一份镜像
  nodes=$(kind get nodes --name "$CLUSTER_NAME" | grep -v control-plane | paste -sd, -)
  tar=$(mktemp "${TMPDIR:-/tmp}/acme-image-XXXXXX.tar")
  local present
  present=$(docker exec "${nodes%%,*}" ctr -n k8s.io images ls -q 2>/dev/null || true)
  for img in "$@"; do
    if grep -Fxq "$(normalize_image "$img")" <<<"$present"; then continue; fi
    log "kind load $img"
    docker save --platform "$platform" -o "$tar" "$img" 2>/dev/null || docker save -o "$tar" "$img"
    kind load image-archive --name "$CLUSTER_NAME" --nodes "$nodes" "$tar" >/dev/null
  done
  rm -f "$tar"
}
