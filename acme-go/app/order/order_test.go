package order_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"

	orderv1 "github.com/acme/acme-go/api/order/v1"
	userv1 "github.com/acme/acme-go/api/user/v1"
	"github.com/acme/acme-go/app/order/internal/conf"
	"github.com/acme/acme-go/app/order/internal/server"
	"github.com/acme/acme-go/app/order/migrations"
	"github.com/acme/acme-go/pkg/archtest"
	"github.com/acme/acme-go/pkg/errorx"
	"github.com/acme/acme-go/pkg/mq"
	"github.com/acme/acme-go/pkg/testkit"
)

// fakeUsers 模拟下游 user-service 的 gRPC 客户端，用于构造“用户正常 / 不存在 / 服务不可用”等场景。
type fakeUsers struct {
	info *userv1.UserInfo
	err  error
}

func (f *fakeUsers) GetUser(context.Context, *userv1.GetUserRequest, ...grpc.CallOption) (*userv1.UserInfo, error) {
	return f.info, f.err
}

type env struct {
	h      http.Handler
	db     *gorm.DB
	relay  *mq.Relay
	bus    *gochannel.GoChannel
	signer *testkit.Signer
	users  *fakeUsers
}

func setup(t *testing.T) *env {
	e := &env{signer: testkit.NewSigner(t), users: &fakeUsers{},
		bus: gochannel.NewGoChannel(gochannel.Config{Persistent: true}, mq.Logger())}
	e.db = testkit.DB(t, migrations.FS)
	c := &conf.Config{Bootstrap: testkit.Bootstrap("order-service")}
	hs, relay := server.New(server.Deps{Conf: c, Telemetry: testkit.Telemetry(t, "order-service"), DB: e.db,
		KeyFunc: e.signer.KeyFunc, Users: e.users, Publisher: e.bus})
	e.h, e.relay = hs, relay
	return e
}

func (e *env) outboxCount(t *testing.T, where string, args ...any) int64 {
	var n int64
	if err := e.db.Table("outbox_event").Where(where, args...).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCreateOrderPublishesEventThroughOutbox(t *testing.T) {
	e := setup(t)
	e.users.info = &userv1.UserInfo{Id: 1001, Username: "alice", Status: "ACTIVE"}
	token := e.signer.Token(t, "1001", "USER")

	resp := testkit.Do(t, e.h, "POST", "/api/v1/orders", token, `{"productCode":"SKU-1","quantity":2,"amount":"99.80"}`)
	if resp.Status != 200 || resp.DataField(t, "status") != "CREATED" || resp.DataField(t, "amount") != "99.80" {
		t.Fatalf("create: %d %s %s %s", resp.Status, resp.Code, resp.Message, resp.Data)
	}
	orderID := resp.DataField(t, "id").(string)

	// 订单与事件在同一事务中写入
	if n := e.outboxCount(t, "aggregate_id = ? AND status = 0", orderID); n != 1 {
		t.Fatalf("outbox rows = %d", n)
	}

	msgs, err := e.bus.Subscribe(context.Background(), orderv1.EventsTopic)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := e.relay.RelayAll(context.Background()); err != nil || n != 1 {
		t.Fatalf("relay: %d %v", n, err)
	}
	var msg *message.Message
	select {
	case msg = <-msgs:
		msg.Ack()
	case <-time.After(time.Second):
		t.Fatal("event not published")
	}
	if msg.Metadata.Get(mq.HeaderType) != orderv1.EventOrderCreated || msg.Metadata.Get(mq.HeaderKey) != orderID {
		t.Fatalf("metadata: %v", msg.Metadata)
	}
	var event orderv1.OrderCreated
	if err := protojson.Unmarshal(msg.Payload, &event); err != nil || event.UserId != 1001 || event.Amount != "99.80" {
		t.Fatalf("payload %s: %v", msg.Payload, err)
	}
	if n := e.outboxCount(t, "aggregate_id = ? AND status = 1", orderID); n != 1 {
		t.Fatal("outbox event not marked as sent")
	}

	// 只能看到自己的订单
	if r := testkit.Do(t, e.h, "GET", "/api/v1/orders/"+orderID, token, ""); r.Status != 200 {
		t.Fatalf("owner get: %d", r.Status)
	}
	if r := testkit.Do(t, e.h, "GET", "/api/v1/orders/"+orderID, e.signer.Token(t, "2002"), ""); r.Status != 404 ||
		r.Code != "A2001" {
		t.Fatalf("other user get: %d %s", r.Status, r.Code)
	}
}

func TestDownstreamFailures(t *testing.T) {
	e := setup(t)
	token := e.signer.Token(t, "9", "USER")
	body := `{"productCode":"SKU-1","quantity":1,"amount":"1"}`

	e.users.err = errorx.Code{Reason: "A1001", Message: "用户不存在", HTTP: 404}.New()
	if r := testkit.Do(t, e.h, "POST", "/api/v1/orders", token, body); r.Status != 422 || r.Code != "A2002" {
		t.Fatalf("unknown buyer: %d %s", r.Status, r.Code)
	}
	if n := e.outboxCount(t, "1 = 1"); n != 0 {
		t.Fatalf("no event expected, got %d", n)
	}

	e.users.err = errorx.RemoteUnavailable.New()
	if r := testkit.Do(t, e.h, "POST", "/api/v1/orders", token, body); r.Status != 503 || r.Code != "C0503" {
		t.Fatalf("user-service down: %d %s", r.Status, r.Code)
	}

	e.users.err = nil
	if r := testkit.Do(t, e.h, "POST", "/api/v1/orders", token, `{"productCode":"","quantity":0,"amount":"x"}`); r.Status != 400 ||
		r.Code != "A0400" {
		t.Fatalf("validation: %d %s", r.Status, r.Code)
	}
	if r := testkit.Do(t, e.h, "POST", "/api/v1/orders", e.signer.Token(t, ""), body); r.Status != 401 {
		t.Fatalf("client-credentials token without uid must be rejected: %d", r.Status)
	}
}

func TestLayers(t *testing.T) {
	archtest.CheckLayers(t, ".")
}
