// Package server 组装根。
package server

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	notificationv1 "github.com/acme/acme-go/api/notification/v1"
	orderv1 "github.com/acme/acme-go/api/order/v1"
	"github.com/acme/acme-go/app/notification/internal/biz"
	"github.com/acme/acme-go/app/notification/internal/conf"
	"github.com/acme/acme-go/app/notification/internal/data"
	"github.com/acme/acme-go/app/notification/internal/service"
	"github.com/acme/acme-go/pkg/auth"
	"github.com/acme/acme-go/pkg/mq"
	"github.com/acme/acme-go/pkg/observability"
	pserver "github.com/acme/acme-go/pkg/server"
)

// DLQTopic 死信 topic：<topic>.<消费组>.dlq
const DLQTopic = orderv1.EventsTopic + ".notification-service.dlq"

type Deps struct {
	Conf       *conf.Config
	Telemetry  *observability.Telemetry
	DB         *gorm.DB
	KeyFunc    jwt.Keyfunc
	Subscriber message.Subscriber
	// DLQPublisher 重试耗尽的消息投递到死信 topic
	DLQPublisher message.Publisher
}

// New 返回 HTTP 服务器与消费路由（两者都由 Kratos App 管理生命周期）。
func New(d Deps) (*http.Server, *mq.Router, error) {
	uc := biz.NewNotificationUsecase(data.NewNotificationRepo(d.DB), data.NewLogSender(d.Telemetry.Logger))

	mws := pserver.Middlewares(d.Telemetry, auth.Server(d.KeyFunc, d.Conf.Auth.Issuer))
	hs := pserver.NewHTTPServer(d.Conf.Server.HTTP, d.Telemetry.Logger, mws, pserver.ServedBy(d.Conf.App))
	notificationv1.RegisterNotificationServiceHTTPServer(hs, service.NewNotificationService(uc))

	router, err := mq.NewRouter(d.DLQPublisher, mq.RouterConfig{
		DLQTopic:        DLQTopic,
		Retries:         d.Conf.Consumer.Retries,
		InitialInterval: d.Conf.Consumer.InitialInterval.Duration,
	})
	if err != nil {
		return nil, nil, err
	}
	consumer := service.NewOrderEventsConsumer(uc, mq.NewIdempotent(d.DB, d.Conf.App.Name, d.Telemetry.Logger))
	router.AddConsumerHandler("order-created", orderv1.EventsTopic, d.Subscriber, consumer.Handle)
	return hs, router, nil
}
