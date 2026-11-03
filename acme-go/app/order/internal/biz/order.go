// Package biz 订单域：领域模型、用例、仓储与外部能力接口。
package biz

import (
	"context"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/acme/acme-go/pkg/errorx"
)

// order-service 错误码，号段 A2000-A2999。
var (
	ErrOrderNotFound      = errorx.Code{Reason: "A2001", Message: "订单不存在: %v", HTTP: 404}
	ErrBuyerNotAvailable  = errorx.Code{Reason: "A2002", Message: "下单用户不存在或已被禁用", HTTP: 422}
	ErrOrderStatusInvalid = errorx.Code{Reason: "A2003", Message: "订单状态不允许该操作: %v", HTTP: 409}
)

type OrderStatus string

const (
	OrderCreated   OrderStatus = "CREATED"
	OrderPaid      OrderStatus = "PAID"
	OrderCancelled OrderStatus = "CANCELLED"
)

// Order 订单聚合根。
type Order struct {
	ID          int64
	OrderNo     string
	UserID      int64
	ProductCode string
	Quantity    int32
	Amount      decimal.Decimal
	Status      OrderStatus
	CreatedAt   time.Time
}

func NewOrder(id, userID int64, productCode string, quantity int32, amount decimal.Decimal, now time.Time) *Order {
	return &Order{ID: id, OrderNo: "O" + strconv.FormatInt(id, 10), UserID: userID, ProductCode: productCode,
		Quantity: quantity, Amount: amount, Status: OrderCreated, CreatedAt: now}
}

func (o *Order) Cancel() error {
	if o.Status != OrderCreated {
		return ErrOrderStatusInvalid.New(o.Status)
	}
	o.Status = OrderCancelled
	return nil
}

func (o *Order) BelongsTo(userID int64) bool { return o.UserID == userID }

// OrderRepo 订单仓储。
type OrderRepo interface {
	NextID() int64
	// FindByID 不存在时返回 (nil, nil)。
	FindByID(ctx context.Context, id int64) (*Order, error)
	Add(ctx context.Context, o *Order) error
}

// BuyerGateway 防腐层：订单域只关心“买家能否下单”，不依赖 user-service 的数据结构。
type BuyerGateway interface {
	CanPlaceOrder(ctx context.Context, userID int64) (bool, error)
}

// Transaction 本地事务边界（data 层用 GORM 实现）。
type Transaction interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// EventPublisher 领域事件发布（data 层通过事务发件箱实现，必须在事务内调用）。
type EventPublisher interface {
	OrderCreated(ctx context.Context, o *Order) error
}

// CreateOrderCommand 下单用例入参。
type CreateOrderCommand struct {
	UserID      int64
	ProductCode string
	Quantity    int32
	Amount      decimal.Decimal
}

type OrderUsecase struct {
	repo   OrderRepo
	buyers BuyerGateway
	tx     Transaction
	events EventPublisher
	now    func() time.Time
}

func NewOrderUsecase(repo OrderRepo, buyers BuyerGateway, tx Transaction, events EventPublisher) *OrderUsecase {
	return &OrderUsecase{repo: repo, buyers: buyers, tx: tx, events: events,
		now: func() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }}
}

// Create 下单：
//  1. 远程校验买家——放在事务之外，避免远程调用期间占用数据库连接；
//  2. 本地事务内：保存订单 + 写入发件箱，二者原子提交；
//  3. Relay 异步把事件投递到 MQ，notification-service 等订阅方消费。
func (uc *OrderUsecase) Create(ctx context.Context, cmd CreateOrderCommand) (*Order, error) {
	ok, err := uc.buyers.CanPlaceOrder(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrBuyerNotAvailable.New()
	}
	o := NewOrder(uc.repo.NextID(), cmd.UserID, cmd.ProductCode, cmd.Quantity, cmd.Amount, uc.now())
	err = uc.tx.InTx(ctx, func(ctx context.Context) error {
		if err := uc.repo.Add(ctx, o); err != nil {
			return err
		}
		return uc.events.OrderCreated(ctx, o)
	})
	if err != nil {
		return nil, err
	}
	return o, nil
}

// GetForBuyer 查询订单；不属于当前用户的订单同样返回“不存在”，避免泄露订单是否存在。
func (uc *OrderUsecase) GetForBuyer(ctx context.Context, orderID, buyerID int64) (*Order, error) {
	o, err := uc.repo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o == nil || !o.BelongsTo(buyerID) {
		return nil, ErrOrderNotFound.New(orderID)
	}
	return o, nil
}
