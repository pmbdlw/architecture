// Package server 组装根。
package server

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	orderv1 "github.com/acme/acme-go/api/order/v1"
	userv1 "github.com/acme/acme-go/api/user/v1"
	"github.com/acme/acme-go/app/order/internal/biz"
	"github.com/acme/acme-go/app/order/internal/conf"
	"github.com/acme/acme-go/app/order/internal/data"
	"github.com/acme/acme-go/app/order/internal/service"
	"github.com/acme/acme-go/pkg/auth"
	"github.com/acme/acme-go/pkg/database"
	"github.com/acme/acme-go/pkg/mq"
	"github.com/acme/acme-go/pkg/observability"
	pserver "github.com/acme/acme-go/pkg/server"
)

type Deps struct {
	Conf      *conf.Config
	Telemetry *observability.Telemetry
	DB        *gorm.DB
	KeyFunc   jwt.Keyfunc
	Users     userv1.UserInternalServiceClient
	Publisher message.Publisher
}

// New 返回对外 HTTP 服务器与发件箱投递器（两者都由 Kratos App 管理生命周期）。
func New(d Deps) (*http.Server, *mq.Relay) {
	uc := biz.NewOrderUsecase(
		data.NewOrderRepo(d.DB),
		data.NewBuyerGateway(d.Users),
		database.NewTransaction(d.DB),
		data.NewEventPublisher(mq.NewOutbox(d.DB, d.Conf.App.Name)),
	)
	mws := pserver.Middlewares(d.Telemetry, auth.Server(d.KeyFunc, d.Conf.Auth.Issuer))
	hs := pserver.NewHTTPServer(d.Conf.Server.HTTP, d.Telemetry.Logger, mws)
	orderv1.RegisterOrderServiceHTTPServer(hs, service.NewOrderService(uc))
	relay := mq.NewRelay(d.DB, d.Publisher, d.Conf.MQ.Outbox, d.Telemetry.Logger)
	return hs, relay
}
