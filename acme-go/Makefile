# acme-go 常用命令
APPS := gateway auth user order notification
VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
REGISTRY ?= registry.acme.com/acme-go

.PHONY: init api lint test build docker

# 安装代码生成与检查工具（版本固定，团队统一）
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/bufbuild/buf/cmd/buf@v1.57.0
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

# 根据 api/**/*.proto 生成 Go 代码（生成结果提交到仓库）
api:
	buf lint
	buf generate

# 静态检查 + 契约兼容性检查（与 main 分支比较，禁止破坏性修改）
lint:
	go vet ./...
	golangci-lint run ./...
	buf breaking --against '.git#branch=main,subdir=acme-go'

# 单元测试 + 服务级集成测试（SQLite + 进程内消息总线，无需外部中间件）
test:
	go test -race -count=1 ./...

build:
	@for app in $(APPS); do \
		CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/$$app ./app/$$app/cmd/$$app || exit 1; \
	done

docker:
	@for app in $(APPS); do \
		name=$$app-service; [ $$app = gateway ] && name=acme-gateway; [ $$app = auth ] && name=acme-auth; \
		docker build -f deploy/docker/Dockerfile --build-arg APP=$$app -t $(REGISTRY)/$$name:$(VERSION) . || exit 1; \
	done
