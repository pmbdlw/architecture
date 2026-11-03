# syntax=docker/dockerfile:1
# =====================================================================
# 所有 Go 服务共用的镜像构建文件（在 acme-go 根目录执行）：
#   docker build -f deploy/docker/Dockerfile --build-arg APP=order \
#     -t registry.acme.com/acme-go/order-service:$(git rev-parse --short HEAD) .
#
# 产物是静态链接的单个二进制 + 配置文件，运行在 distroless 基础镜像上：
# 无 shell、无包管理器、非 root，镜像约 20MB，冷启动通常 < 1 秒。
# =====================================================================
ARG GO_IMAGE=golang:1.26
ARG RUNTIME_IMAGE=gcr.io/distroless/static-debian12:nonroot

FROM ${GO_IMAGE} AS build
ARG APP
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/app ./app/${APP}/cmd/${APP}

FROM ${RUNTIME_IMAGE}
ARG APP
WORKDIR /app
COPY --from=build /out/app /app/app
COPY app/${APP}/configs /app/configs
USER 65532:65532
EXPOSE 8080 9090 8081
ENTRYPOINT ["/app/app", "-conf", "/app/configs/config.yaml"]
