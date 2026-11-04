package mq_test

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/database"
	"github.com/acme/acme-go/pkg/mq"
	"github.com/acme/acme-go/pkg/testkit"
)

//go:embed testdata/sqlite/*.sql
var migrations embed.FS

func fsSub() (fs.FS, error) { return fs.Sub(migrations, "testdata") }

func TestOutboxRelayAndIdempotentConsumer(t *testing.T) {
	sub, err := fsSub()
	if err != nil {
		t.Fatal(err)
	}
	db := testkit.DB(t, sub)
	ctx := context.Background()
	bus := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, mq.Logger())
	outbox := mq.NewOutbox(db, "order-service")

	if _, err := outbox.Publish(ctx, "acme.order.events", "order.created", "1", wrapperspb.String("x")); !errors.Is(err, mq.ErrNoTransaction) {
		t.Fatalf("publish outside tx should fail, got %v", err)
	}
	// 模拟入口请求的链路：traceparent 应随事件写入发件箱，并由 Relay 续接到消息头
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	reqCtx, reqSpan := otel.Tracer("test").Start(ctx, "POST /api/v1/orders")
	traceID := reqSpan.SpanContext().TraceID().String()
	tx := database.NewTransaction(db)
	var eventID string
	if err := tx.InTx(reqCtx, func(ctx context.Context) error {
		eventID, err = outbox.Publish(ctx, "acme.order.events", "order.created", "42", wrapperspb.String("hello"))
		return err
	}); err != nil {
		t.Fatal(err)
	}

	msgs, err := bus.Subscribe(ctx, "acme.order.events")
	if err != nil {
		t.Fatal(err)
	}
	relay := mq.NewRelay(db, bus, conf.Outbox{BatchSize: 10, MaxRetries: 3}, log.DefaultLogger)
	if n, err := relay.RelayAll(ctx); err != nil || n != 1 {
		t.Fatalf("relay sent %d, err %v", n, err)
	}
	var msg *message.Message
	select {
	case msg = <-msgs:
		defer msg.Ack()
	case <-time.After(time.Second):
		t.Fatal("no message relayed")
	}
	if msg.Metadata.Get(mq.HeaderID) != eventID || msg.Metadata.Get(mq.HeaderKey) != "42" ||
		msg.Metadata.Get(mq.HeaderType) != "order.created" || msg.Metadata.Get(mq.HeaderSource) != "order-service" {
		t.Fatalf("unexpected metadata %v", msg.Metadata)
	}
	reqSpan.End()
	if tp := msg.Metadata.Get(mq.HeaderTraceParent); !strings.HasPrefix(tp, "00-"+traceID+"-") {
		t.Fatalf("traceparent %q should continue trace %s", tp, traceID)
	}
	if n, _ := relay.RelayAll(ctx); n != 0 {
		t.Fatalf("already-sent events relayed again: %d", n)
	}

	calls := 0
	handler := mq.NewIdempotent(db, "notification-service", log.DefaultLogger).
		Handler(func(context.Context, *message.Message) error { calls++; return nil })
	for range 2 {
		if err := handler(msg); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("duplicate message handled %d times", calls)
	}
}

func TestKafkaProducerConfigIsValid(t *testing.T) {
	// 幂等生产者对 sarama 配置有约束（acks=all、MaxOpenRequests=1、版本 ≥ 0.11），在单元测试中提前校验
	if err := mq.KafkaProducerConfig().Validate(); err != nil {
		t.Fatalf("invalid kafka producer config: %v", err)
	}
}
