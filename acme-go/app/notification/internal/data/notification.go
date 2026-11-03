// Package data 通知域数据层。
package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"

	"github.com/acme/acme-go/app/notification/internal/biz"
	"github.com/acme/acme-go/pkg/database"
)

// NotificationPO notification 表映射。
type NotificationPO struct {
	database.Model
	UserID  int64
	Channel string
	Title   string
	Content string
	BizKey  string
}

func (NotificationPO) TableName() string { return "notification" }

type notificationRepo struct {
	db *gorm.DB
}

func NewNotificationRepo(db *gorm.DB) biz.NotificationRepo { return &notificationRepo{db: db} }

func (r *notificationRepo) NextID() int64 { return database.NextID() }

func (r *notificationRepo) Add(ctx context.Context, n *biz.Notification) error {
	po := NotificationPO{Model: database.Model{ID: n.ID, CreatedAt: n.CreatedAt}, UserID: n.UserID,
		Channel: n.Channel, Title: n.Title, Content: n.Content, BizKey: n.BizKey}
	return database.DB(ctx, r.db).Create(&po).Error
}

func (r *notificationRepo) PageByUser(ctx context.Context, userID, pageNo, pageSize int64) ([]*biz.Notification, int64, error) {
	q := database.DB(ctx, r.db).Model(&NotificationPO{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []NotificationPO
	if err := q.Order("created_at DESC").Offset(int((pageNo - 1) * pageSize)).Limit(int(pageSize)).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*biz.Notification, 0, len(rows))
	for _, p := range rows {
		out = append(out, &biz.Notification{ID: p.ID, UserID: p.UserID, Channel: p.Channel, Title: p.Title,
			Content: p.Content, BizKey: p.BizKey, CreatedAt: p.CreatedAt})
	}
	return out, total, nil
}

// logSender 占位实现：只记录日志。接入真实渠道时新增实现并按渠道路由。
type logSender struct {
	log *log.Helper
}

func NewLogSender(logger log.Logger) biz.Sender { return &logSender{log: log.NewHelper(logger)} }

func (s *logSender) Send(ctx context.Context, n *biz.Notification) error {
	s.log.WithContext(ctx).Infow("msg", "发送通知", "channel", n.Channel, "user_id", n.UserID, "biz_key", n.BizKey)
	return nil
}
