// Package data 订单域数据层：订单仓储、调用 user-service 的防腐层实现、发件箱事件发布。
package data

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	orderv1 "github.com/acme/acme-go/api/order/v1"
	userv1 "github.com/acme/acme-go/api/user/v1"
	"github.com/acme/acme-go/app/order/internal/biz"
	"github.com/acme/acme-go/pkg/database"
	"github.com/acme/acme-go/pkg/errorx"
	"github.com/acme/acme-go/pkg/mq"
)

// OrderPO order_info 表映射。
type OrderPO struct {
	database.Model
	OrderNo     string
	UserID      int64
	ProductCode string
	Quantity    int32
	Amount      decimal.Decimal `gorm:"type:decimal(18,2)"`
	Status      string
}

func (OrderPO) TableName() string { return "order_info" }

type orderRepo struct {
	db *gorm.DB
}

func NewOrderRepo(db *gorm.DB) biz.OrderRepo { return &orderRepo{db: db} }

func (r *orderRepo) NextID() int64 { return database.NextID() }

func (r *orderRepo) FindByID(ctx context.Context, id int64) (*biz.Order, error) {
	var po OrderPO
	err := database.DB(ctx, r.db).First(&po, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &biz.Order{ID: po.ID, OrderNo: po.OrderNo, UserID: po.UserID, ProductCode: po.ProductCode,
		Quantity: po.Quantity, Amount: po.Amount, Status: biz.OrderStatus(po.Status), CreatedAt: po.CreatedAt}, nil
}

func (r *orderRepo) Add(ctx context.Context, o *biz.Order) error {
	po := OrderPO{Model: database.Model{ID: o.ID, CreatedAt: o.CreatedAt}, OrderNo: o.OrderNo, UserID: o.UserID,
		ProductCode: o.ProductCode, Quantity: o.Quantity, Amount: o.Amount, Status: string(o.Status)}
	return database.DB(ctx, r.db).Create(&po).Error
}

// buyerGateway 通过 gRPC 调用 user-service：
//   - 下游业务错误（如 A1001 用户不存在）→ 不可下单；
//   - 下游故障（C0001 / C0503，已由 pkg/client 中间件还原）→ 原样返回。
type buyerGateway struct {
	users userv1.UserInternalServiceClient
}

func NewBuyerGateway(users userv1.UserInternalServiceClient) biz.BuyerGateway {
	return &buyerGateway{users: users}
}

func (g *buyerGateway) CanPlaceOrder(ctx context.Context, userID int64) (bool, error) {
	u, err := g.users.GetUser(ctx, &userv1.GetUserRequest{Id: userID})
	if err != nil {
		se := errorx.Normalize(err)
		if se.Code < 500 {
			return false, nil
		}
		return false, se
	}
	return u.Status == "ACTIVE", nil
}

// eventPublisher 把领域事件写入事务发件箱。
type eventPublisher struct {
	outbox *mq.Outbox
}

func NewEventPublisher(outbox *mq.Outbox) biz.EventPublisher { return &eventPublisher{outbox: outbox} }

func (p *eventPublisher) OrderCreated(ctx context.Context, o *biz.Order) error {
	_, err := p.outbox.Publish(ctx, orderv1.EventsTopic, orderv1.EventOrderCreated, strconv.FormatInt(o.ID, 10),
		&orderv1.OrderCreated{
			OrderId:     o.ID,
			OrderNo:     o.OrderNo,
			UserId:      o.UserID,
			ProductCode: o.ProductCode,
			Quantity:    o.Quantity,
			Amount:      o.Amount.StringFixed(2),
			CreatedAt:   timestamppb.New(o.CreatedAt.In(time.UTC)),
		})
	return err
}
