package notification_test

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
	"gorm.io/gorm"

	orderv1 "github.com/acme/acme-go/api/order/v1"
	"github.com/acme/acme-go/app/notification/internal/conf"
	"github.com/acme/acme-go/app/notification/internal/server"
	"github.com/acme/acme-go/app/notification/migrations"
	"github.com/acme/acme-go/pkg/archtest"
	pconf "github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/mq"
	"github.com/acme/acme-go/pkg/testkit"
)

func event(id, eventType, payload string) *message.Message {
	msg := message.NewMessage(uuid.NewString(), []byte(payload))
	msg.Metadata.Set(mq.HeaderID, id)
	msg.Metadata.Set(mq.HeaderType, eventType)
	return msg
}

func count(t *testing.T, db *gorm.DB, table, where string, args ...any) int64 {
	var n int64
	if err := db.Table(table).Where(where, args...).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestConsumeOrderCreatedExactlyOnceAndDeadLetter(t *testing.T) {
	bus := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, mq.Logger())
	signer := testkit.NewSigner(t)
	db := testkit.DB(t, migrations.FS)
	c := &conf.Config{Bootstrap: testkit.Bootstrap("notification-service"),
		Consumer: conf.Consumer{Retries: 2, InitialInterval: pconf.Duration{Duration: 10 * time.Millisecond}}}
	h, router, err := server.New(server.Deps{Conf: c, Telemetry: testkit.Telemetry(t, "notification-service"),
		DB: db, KeyFunc: signer.KeyFunc, Subscriber: bus, DLQPublisher: bus})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = router.Start(ctx) }()
	<-router.Running()
	dlq, err := bus.Subscribe(ctx, server.DLQTopic)
	if err != nil {
		t.Fatal(err)
	}

	payload := `{"orderId":"1","orderNo":"O1","userId":"1001","productCode":"SKU-1","quantity":1,"amount":"9.90",
		"createdAt":"2026-10-01T08:00:00Z","fieldFromFutureVersion":"ignored"}`
	eventID := uuid.NewString()
	// MQ 至少一次投递：同一事件重复到达
	for range 2 {
		if err := bus.Publish(orderv1.EventsTopic, event(eventID, orderv1.EventOrderCreated, payload)); err != nil {
			t.Fatal(err)
		}
	}
	other := uuid.NewString()
	if err := bus.Publish(orderv1.EventsTopic, event(other, "order.cancelled", `{}`)); err != nil {
		t.Fatal(err)
	}
	poison := uuid.NewString()
	if err := bus.Publish(orderv1.EventsTopic, event(poison, orderv1.EventOrderCreated, `not-json`)); err != nil {
		t.Fatal(err)
	}

	eventually(t, "notification", func() bool { return count(t, db, "notification", "biz_key = ?", "O1") == 1 })
	select {
	case msg := <-dlq:
		msg.Ack()
		if msg.Metadata.Get(mq.HeaderID) != poison {
			t.Fatalf("unexpected DLQ message %v", msg.Metadata)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("poison message did not reach DLQ")
	}
	if n := count(t, db, "notification", "biz_key = ?", "O1"); n != 1 {
		t.Fatalf("duplicate event handled %d times", n)
	}
	if n := count(t, db, "consumed_event", "event_id = ?", other); n != 0 {
		t.Fatal("other event types must be ignored")
	}

	resp := testkit.Do(t, h, "GET", "/api/v1/notifications?pageNo=1&pageSize=10", signer.Token(t, "1001"), "")
	if resp.Status != 200 || resp.DataField(t, "total") != "1" {
		t.Fatalf("list: %d %s", resp.Status, resp.Data)
	}
	records := resp.DataField(t, "records").([]any)
	if records[0].(map[string]any)["title"] != "下单成功" {
		t.Fatalf("records: %v", records)
	}
	if r := testkit.Do(t, h, "GET", "/api/v1/notifications?pageSize=500", signer.Token(t, "1001"), ""); r.Status != 400 {
		t.Fatalf("page size limit: %d", r.Status)
	}
}

func TestLayers(t *testing.T) {
	archtest.CheckLayers(t, ".")
}
