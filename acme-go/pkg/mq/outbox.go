package mq

import (
	"context"
	"errors"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/database"
)

// OutboxEvent 事务发件箱表（所有发布事件的服务都需要，见各服务迁移脚本）。
type OutboxEvent struct {
	EventID     string `gorm:"primaryKey"`
	Topic       string
	EventType   string
	AggregateID string
	Payload     string
	Source      string
	Status      int
	RetryCount  int
	CreatedAt   time.Time `gorm:"precision:3"`
	SentAt      *time.Time
	// 写入时的 W3C traceparent，Relay 投递时续接同一条链路
	TraceParent *string
}

func (OutboxEvent) TableName() string { return "outbox_event" }

// Outbox 事务发件箱写入端。
//
// 业务代码在本地事务内调用 Publish，事件与业务数据同库同事务落盘，
// 由 Relay 异步投递到 Kafka / RabbitMQ —— “改数据”与“发消息”要么都成功、要么都失败。
type Outbox struct {
	db     *gorm.DB
	source string
}

func NewOutbox(db *gorm.DB, source string) *Outbox { return &Outbox{db: db, source: source} }

// ErrNoTransaction 在事务外调用 Publish 时返回。
var ErrNoTransaction = errors.New("outbox: Publish 必须在事务内调用（database.Transaction.InTx）")

// Publish 把事件写入发件箱，返回事件 ID。
func (o *Outbox) Publish(ctx context.Context, topic, eventType, aggregateID string, payload proto.Message) (string, error) {
	if !database.InTransaction(ctx) {
		return "", ErrNoTransaction
	}
	body, err := protojson.Marshal(payload)
	if err != nil {
		return "", err
	}
	e := OutboxEvent{
		EventID:     uuid.NewString(),
		Topic:       topic,
		EventType:   eventType,
		AggregateID: aggregateID,
		Payload:     string(body),
		Source:      o.source,
		CreatedAt:   time.Now().UTC().Truncate(time.Millisecond),
	}
	if tp := currentTraceParent(ctx); tp != "" {
		e.TraceParent = &tp
	}
	return e.EventID, database.DB(ctx, o.db).Create(&e).Error
}

// Relay 发件箱投递器：定时扫描待发送事件，投递成功后标记为已发送。
//
// 多副本部署依靠 SELECT ... FOR UPDATE SKIP LOCKED 互不抢占同一批事件；
// 投递语义为至少一次，消费端必须幂等（见 Idempotent）。
// Relay 实现了 transport.Server，可直接交给 Kratos App 管理生命周期。
type Relay struct {
	db        *gorm.DB
	publisher message.Publisher
	cfg       conf.Outbox
	log       *log.Helper
	stop      chan struct{}
}

func NewRelay(db *gorm.DB, publisher message.Publisher, cfg conf.Outbox, logger log.Logger) *Relay {
	return &Relay{db: db, publisher: publisher, cfg: cfg, log: log.NewHelper(logger), stop: make(chan struct{})}
}

// Start 周期执行投递直到 Stop。
func (r *Relay) Start(ctx context.Context) error {
	interval := r.cfg.PollInterval.Duration
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err := r.RelayAll(ctx); err != nil {
				r.log.Warnw("msg", "发件箱投递失败", "error", err)
			}
		case <-r.stop:
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

func (r *Relay) Stop(context.Context) error {
	close(r.stop)
	return nil
}

// RelayAll 投递所有待发送事件，返回成功条数（测试中可直接调用）。
func (r *Relay) RelayAll(ctx context.Context) (int, error) {
	total := 0
	for {
		n, err := r.relayBatch(ctx)
		total += n
		if err != nil || n < r.batchSize() {
			return total, err
		}
	}
}

func (r *Relay) batchSize() int {
	if r.cfg.BatchSize <= 0 {
		return 100
	}
	return r.cfg.BatchSize
}

func (r *Relay) relayBatch(ctx context.Context) (int, error) {
	sent := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var events []OutboxEvent
		q := `SELECT * FROM outbox_event WHERE status = 0 AND retry_count < ? ORDER BY created_at LIMIT ? ` + r.cfg.LockClause
		if err := tx.Raw(q, r.cfg.MaxRetries, r.batchSize()).Scan(&events).Error; err != nil {
			return err
		}
		for _, e := range events {
			err := r.publish(ctx, e)
			result := "success"
			if err != nil {
				result = "failure"
			}
			metrics().published.Add(ctx, 1, attrs("topic", e.Topic, "type", e.EventType, "result", result))
			if err != nil {
				r.log.Warnw("msg", "投递发件箱事件失败", "event_id", e.EventID, "type", e.EventType, "error", err)
				if err := tx.Model(&OutboxEvent{}).Where("event_id = ?", e.EventID).
					Update("retry_count", gorm.Expr("retry_count + 1")).Error; err != nil {
					return err
				}
				continue
			}
			now := time.Now().UTC()
			if err := tx.Model(&OutboxEvent{}).Where("event_id = ?", e.EventID).
				Updates(map[string]any{"status": 1, "sent_at": now}).Error; err != nil {
				return err
			}
			sent++
		}
		return nil
	})
	return sent, err
}

// publish 以写入时的链路为父节点创建 PRODUCER Span，并把新的 traceparent 写入消息头。
func (r *Relay) publish(ctx context.Context, e OutboxEvent) error {
	parent := ""
	if e.TraceParent != nil {
		parent = *e.TraceParent
	}
	ctx, span := tracer().Start(contextFromTraceParent(ctx, parent), "publish "+e.EventType,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.destination.name", e.Topic),
			attribute.String("messaging.message.id", e.EventID),
		))
	defer span.End()

	msg := message.NewMessage(e.EventID, []byte(e.Payload))
	msg.Metadata.Set(HeaderID, e.EventID)
	msg.Metadata.Set(HeaderType, e.EventType)
	msg.Metadata.Set(HeaderSource, e.Source)
	msg.Metadata.Set(HeaderTime, e.CreatedAt.Format(time.RFC3339Nano))
	msg.Metadata.Set(HeaderKey, e.AggregateID)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(msg.Metadata))
	if err := r.publisher.Publish(e.Topic, msg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

// RegisterMetrics 注册积压量指标 acme_outbox_pending：持续上涨说明 MQ 不可用或投递失败，需要告警。
func (r *Relay) RegisterMetrics() error {
	_, err := otel.Meter(instrumentationName).Int64ObservableGauge("acme.outbox.pending",
		metric.WithDescription("发件箱中待投递的事件数"),
		metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
			var n int64
			if err := r.db.WithContext(ctx).Model(&OutboxEvent{}).Where("status = 0").Count(&n).Error; err != nil {
				return err
			}
			o.Observe(n)
			return nil
		}))
	return err
}
