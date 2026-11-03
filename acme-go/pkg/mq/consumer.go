package mq

import (
	"context"
	"errors"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"

	"github.com/acme/acme-go/pkg/database"
)

// ConsumedEvent 消费幂等表（所有消费事件的服务都需要），主键 (consumer, event_id)。
type ConsumedEvent struct {
	Consumer   string `gorm:"primaryKey"`
	EventID    string `gorm:"primaryKey"`
	ConsumedAt time.Time
}

func (ConsumedEvent) TableName() string { return "consumed_event" }

// Idempotent 消费幂等模板：在同一个本地事务中“登记事件 ID + 执行业务”，
// 重复投递的消息因主键冲突被识别并跳过。
type Idempotent struct {
	tx       *database.Transaction
	db       *gorm.DB
	consumer string
	log      *log.Helper
}

func NewIdempotent(db *gorm.DB, consumer string, logger log.Logger) *Idempotent {
	return &Idempotent{tx: database.NewTransaction(db), db: db, consumer: consumer, log: log.NewHelper(logger)}
}

// Handler 包装业务处理函数；fn 收到的 ctx 已处于事务中，仓储写入与登记一起提交。
func (i *Idempotent) Handler(fn func(ctx context.Context, msg *message.Message) error) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		id := msg.Metadata.Get(HeaderID)
		if id == "" {
			return errors.New("消息缺少 " + HeaderID + " 头，无法保证幂等")
		}
		err := i.tx.InTx(msg.Context(), func(ctx context.Context) error {
			record := ConsumedEvent{Consumer: i.consumer, EventID: id, ConsumedAt: time.Now().UTC()}
			if err := database.DB(ctx, i.db).Create(&record).Error; err != nil {
				return err
			}
			return fn(ctx, msg)
		})
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			i.log.Infow("msg", "重复消息已跳过", "consumer", i.consumer, "event_id", id)
			return nil
		}
		return err
	}
}

// RouterConfig 消费路由配置。
type RouterConfig struct {
	// DLQTopic 重试耗尽后进入的死信 topic：<topic>.<消费组>.dlq
	DLQTopic string
	// Retries 重试次数（指数退避）。
	Retries         int
	InitialInterval time.Duration
}

// Router 消费路由，实现 transport.Server，交给 Kratos App 管理生命周期。
type Router struct {
	*message.Router
}

// NewRouter 创建带“重试 + 死信 + panic 恢复”的消费路由。
func NewRouter(dlqPublisher message.Publisher, cfg RouterConfig) (*Router, error) {
	r, err := message.NewRouter(message.RouterConfig{}, Logger())
	if err != nil {
		return nil, err
	}
	poison, err := middleware.PoisonQueue(dlqPublisher, cfg.DLQTopic)
	if err != nil {
		return nil, err
	}
	if cfg.Retries == 0 {
		cfg.Retries = 3
	}
	if cfg.InitialInterval == 0 {
		cfg.InitialInterval = time.Second
	}
	retry := middleware.Retry{
		MaxRetries:      cfg.Retries,
		InitialInterval: cfg.InitialInterval,
		Multiplier:      2,
		Logger:          Logger(),
	}
	// 先添加的在外层：死信 → 重试 → 恢复 → 业务
	r.AddMiddleware(poison, retry.Middleware, middleware.Recoverer)
	return &Router{Router: r}, nil
}

func (r *Router) Start(ctx context.Context) error { return r.Run(ctx) }

func (r *Router) Stop(context.Context) error { return r.Close() }
