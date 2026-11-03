// Package service 通知接口层：HTTP 查询接口 + 消息消费入口（二者同属接口层）。
package service

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	notificationv1 "github.com/acme/acme-go/api/notification/v1"
	orderv1 "github.com/acme/acme-go/api/order/v1"
	"github.com/acme/acme-go/app/notification/internal/biz"
	"github.com/acme/acme-go/pkg/auth"
	"github.com/acme/acme-go/pkg/errorx"
	"github.com/acme/acme-go/pkg/mq"
)

type NotificationService struct {
	notificationv1.UnimplementedNotificationServiceServer
	uc *biz.NotificationUsecase
}

func NewNotificationService(uc *biz.NotificationUsecase) *NotificationService {
	return &NotificationService{uc: uc}
}

func (s *NotificationService) ListMyNotifications(ctx context.Context, req *notificationv1.ListMyNotificationsRequest) (*notificationv1.ListMyNotificationsReply, error) {
	uid, err := auth.UID(ctx)
	if err != nil {
		return nil, err
	}
	pageNo, pageSize := req.PageNo, req.PageSize
	if pageNo == 0 {
		pageNo = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if pageNo < 1 || pageSize < 1 || pageSize > 100 {
		return nil, errorx.BadRequest.Msg("pageNo 从 1 开始，pageSize 需在 1-100 之间")
	}
	list, total, err := s.uc.PageMine(ctx, uid, pageNo, pageSize)
	if err != nil {
		return nil, err
	}
	reply := &notificationv1.ListMyNotificationsReply{Total: total, PageNo: pageNo, PageSize: pageSize}
	for _, n := range list {
		reply.Records = append(reply.Records, &notificationv1.Notification{Id: n.ID, Channel: n.Channel,
			Title: n.Title, Content: n.Content, BizKey: n.BizKey, CreatedAt: timestamppb.New(n.CreatedAt)})
	}
	return reply, nil
}

// OrderEventsConsumer 订单事件消费者。
type OrderEventsConsumer struct {
	uc         *biz.NotificationUsecase
	idempotent *mq.Idempotent
}

func NewOrderEventsConsumer(uc *biz.NotificationUsecase, idempotent *mq.Idempotent) *OrderEventsConsumer {
	return &OrderEventsConsumer{uc: uc, idempotent: idempotent}
}

// Handle 只处理 order.created；同一 topic 上的其它订单事件直接确认、不登记幂等记录。
func (c *OrderEventsConsumer) Handle(msg *message.Message) error {
	if msg.Metadata.Get(mq.HeaderType) != orderv1.EventOrderCreated {
		return nil
	}
	return c.idempotent.Handler(c.onOrderCreated)(msg)
}

func (c *OrderEventsConsumer) onOrderCreated(ctx context.Context, msg *message.Message) error {
	var e orderv1.OrderCreated
	// DiscardUnknown：兼容生产方新增字段
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(msg.Payload, &e); err != nil {
		return err
	}
	return c.uc.OnOrderCreated(ctx, biz.OrderCreated{OrderNo: e.OrderNo, UserID: e.UserId, Amount: e.Amount})
}
