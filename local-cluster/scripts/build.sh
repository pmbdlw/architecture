#!/usr/bin/env bash
# 构建一套技术栈的 5 个服务镜像并加载进集群节点。
#   scripts/build.sh go
#   scripts/build.sh spring
# 构建产物（jar / 二进制）在宿主机编译，复用本机 Maven / Go 缓存；镜像使用各工程 deploy/docker/Dockerfile。
source "$(dirname "$0")/lib.sh"

STACK=$(stack_arg "${1:-}")
TAG=$(image_tag)

build_go() {
  need go docker
  local dir=$REPO_DIR/acme-go arch
  # 镜像运行在 Linux 节点上：按 Docker 引擎的架构交叉编译（macOS / Apple Silicon 同样适用）
  arch=$(docker version -f '{{.Server.Arch}}')
  pull "${RUNTIME_IMAGE:-gcr.io/distroless/static-debian12:nonroot}"
  log "编译 Go 服务（linux/$arch）"
  local apps=(gateway auth user order notification) images=()
  for app in "${apps[@]}"; do
    (cd "$dir" && CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags "-s -w" -o "bin/$app" "./app/$app/cmd/$app")
  done
  for app in "${apps[@]}"; do
    local name=$app-service
    [[ $app == gateway ]] && name=acme-gateway
    [[ $app == auth ]] && name=acme-auth
    local img=acme-go/$name:$TAG
    log "docker build $img"
    DOCKER_BUILDKIT=1 docker build -q -f "$dir/deploy/docker/Dockerfile" \
      --build-arg APP="$app" --build-arg BUILD_MODE=prebuilt \
      ${RUNTIME_IMAGE:+--build-arg RUNTIME_IMAGE="$RUNTIME_IMAGE"} \
      -t "$img" "$dir" >/dev/null
    images+=("$img")
  done
  load_images "${images[@]}"
}

build_spring() {
  need mvn java docker
  local dir=$REPO_DIR/acme-cloud profile=kafka
  pull "${JRE_IMAGE:-eclipse-temurin:21-jre-noble}"
  [[ "$MQ" == rabbitmq ]] && profile=rabbit
  # Binder 在构建期决定：Kafka / RabbitMQ 镜像各不相同
  log "Maven 打包（-P$profile，跳过测试）"
  (cd "$dir" && mvn -B -q package -DskipTests -P"$profile")
  local images=()
  local modules=(
    "acme-gateway:acme-gateway"
    "acme-auth:acme-auth"
    "acme-services/user-service/user-server:user-service"
    "acme-services/order-service/order-server:order-service"
    "acme-services/notification-service/notification-server:notification-service"
  )
  for m in "${modules[@]}"; do
    local module=${m%%:*} name=${m##*:}
    local img=acme-cloud/$name:$TAG
    log "docker build $img"
    DOCKER_BUILDKIT=1 docker build -q -f "$dir/deploy/docker/Dockerfile" \
      --build-arg MODULE="$module" \
      ${JRE_IMAGE:+--build-arg JRE_IMAGE="$JRE_IMAGE"} \
      -t "$img" "$dir" >/dev/null
    images+=("$img")
  done
  load_images "${images[@]}"
}

"build_$STACK"
echo "$TAG" > "$LC_DIR/.image-tag-$STACK"
ok "镜像 tag：$TAG"
