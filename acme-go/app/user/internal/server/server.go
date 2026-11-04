// Package server 组装根：创建仓储、用例、接口实现，并注册到 HTTP / gRPC 服务器。
package server

import (
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	userv1 "github.com/acme/acme-go/api/user/v1"
	"github.com/acme/acme-go/app/user/internal/biz"
	"github.com/acme/acme-go/app/user/internal/conf"
	"github.com/acme/acme-go/app/user/internal/data"
	"github.com/acme/acme-go/app/user/internal/service"
	"github.com/acme/acme-go/pkg/auth"
	"github.com/acme/acme-go/pkg/observability"
	pserver "github.com/acme/acme-go/pkg/server"
)

// Deps 外部依赖（由 main 或测试提供）。
type Deps struct {
	Conf      *conf.Config
	Telemetry *observability.Telemetry
	DB        *gorm.DB
	KeyFunc   jwt.Keyfunc
}

// New 返回对外 HTTP 服务器与集群内 gRPC 服务器。
func New(d Deps) (*http.Server, *grpc.Server) {
	uc := biz.NewUserUsecase(data.NewUserRepo(d.DB))
	mws := pserver.Middlewares(d.Telemetry, auth.Server(d.KeyFunc, d.Conf.Auth.Issuer))

	hs := pserver.NewHTTPServer(d.Conf.Server.HTTP, d.Telemetry.Logger, mws, pserver.ServedBy(d.Conf.App))
	userv1.RegisterUserServiceHTTPServer(hs, service.NewUserService(uc))

	gs := pserver.NewGRPCServer(d.Conf.Server.GRPC, mws)
	userv1.RegisterUserInternalServiceServer(gs, service.NewUserInternalService(uc))
	return hs, gs
}

// InternalService 单独构造集群内接口（测试中直接调用，不经过 gRPC 网络）。
func InternalService(db *gorm.DB) userv1.UserInternalServiceServer {
	return service.NewUserInternalService(biz.NewUserUsecase(data.NewUserRepo(db)))
}
