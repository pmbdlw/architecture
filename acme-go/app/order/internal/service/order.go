// Package service 订单接口层。当前用户 ID 一律取自 JWT 的 uid，绝不信任请求体中的 userId。
package service

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	orderv1 "github.com/acme/acme-go/api/order/v1"
	"github.com/acme/acme-go/app/order/internal/biz"
	"github.com/acme/acme-go/pkg/auth"
	"github.com/acme/acme-go/pkg/errorx"
)

type OrderService struct {
	orderv1.UnimplementedOrderServiceServer
	uc *biz.OrderUsecase
}

func NewOrderService(uc *biz.OrderUsecase) *OrderService { return &OrderService{uc: uc} }

func (s *OrderService) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.OrderReply, error) {
	uid, err := auth.UID(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := validateCreate(req)
	if err != nil {
		return nil, err
	}
	o, err := s.uc.Create(ctx, biz.CreateOrderCommand{UserID: uid, ProductCode: req.ProductCode,
		Quantity: req.Quantity, Amount: amount})
	if err != nil {
		return nil, err
	}
	return toReply(o), nil
}

func (s *OrderService) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.OrderReply, error) {
	uid, err := auth.UID(ctx)
	if err != nil {
		return nil, err
	}
	o, err := s.uc.GetForBuyer(ctx, req.Id, uid)
	if err != nil {
		return nil, err
	}
	return toReply(o), nil
}

func validateCreate(req *orderv1.CreateOrderRequest) (decimal.Decimal, error) {
	var problems []string
	if strings.TrimSpace(req.ProductCode) == "" {
		problems = append(problems, "productCode 不能为空")
	}
	if req.Quantity < 1 || req.Quantity > 999 {
		problems = append(problems, "quantity 需在 1-999 之间")
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil || amount.LessThan(decimal.RequireFromString("0.01")) {
		problems = append(problems, "amount 需为不小于 0.01 的金额")
	}
	if len(problems) > 0 {
		return decimal.Zero, errorx.BadRequest.Msg(strings.Join(problems, "; "))
	}
	return amount, nil
}

func toReply(o *biz.Order) *orderv1.OrderReply {
	return &orderv1.OrderReply{Id: o.ID, OrderNo: o.OrderNo, ProductCode: o.ProductCode, Quantity: o.Quantity,
		Amount: o.Amount.StringFixed(2), Status: string(o.Status), CreatedAt: timestamppb.New(o.CreatedAt)}
}
