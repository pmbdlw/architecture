// Package biz 通知域：订阅业务事件并通知用户。
package biz

import (
	"context"
	"fmt"
	"time"
)

// Notification 站内通知。Channel：SITE / SMS / EMAIL；BizKey：业务键（如订单号），便于排查。
type Notification struct {
	ID        int64
	UserID    int64
	Channel   string
	Title     string
	Content   string
	BizKey    string
	CreatedAt time.Time
}

// NotificationRepo 通知仓储。
type NotificationRepo interface {
	NextID() int64
	Add(ctx context.Context, n *Notification) error
	PageByUser(ctx context.Context, userID, pageNo, pageSize int64) ([]*Notification, int64, error)
}

// Sender 发送渠道：自建短信网关 / 阿里云短信 / 邮件推送 / AWS SES / Azure Communication Services 等。
type Sender interface {
	Send(ctx context.Context, n *Notification) error
}

// OrderCreated 业务层关心的订单事件字段（与消息契约解耦）。
type OrderCreated struct {
	OrderNo string
	UserID  int64
	Amount  string
}

type NotificationUsecase struct {
	repo   NotificationRepo
	sender Sender
}

func NewNotificationUsecase(repo NotificationRepo, sender Sender) *NotificationUsecase {
	return &NotificationUsecase{repo: repo, sender: sender}
}

// OnOrderCreated 处理“订单已创建”事件。调用方（幂等消费模板）已保证同一事件只会成功处理一次。
func (uc *NotificationUsecase) OnOrderCreated(ctx context.Context, e OrderCreated) error {
	n := &Notification{
		ID:        uc.repo.NextID(),
		UserID:    e.UserID,
		Channel:   "SITE",
		Title:     "下单成功",
		Content:   fmt.Sprintf("您的订单 %s 已创建，金额 %s 元", e.OrderNo, e.Amount),
		BizKey:    e.OrderNo,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	if err := uc.repo.Add(ctx, n); err != nil {
		return err
	}
	return uc.sender.Send(ctx, n)
}

// PageMine 分页查询某用户的通知。
func (uc *NotificationUsecase) PageMine(ctx context.Context, userID, pageNo, pageSize int64) ([]*Notification, int64, error) {
	return uc.repo.PageByUser(ctx, userID, pageNo, pageSize)
}
