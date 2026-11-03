package service

import (
	"context"

	userv1 "github.com/acme/acme-go/api/user/v1"
	"github.com/acme/acme-go/app/user/internal/biz"
)

// UserInternalService 集群内 gRPC 接口，供 order-service 等调用。
type UserInternalService struct {
	userv1.UnimplementedUserInternalServiceServer
	uc *biz.UserUsecase
}

func NewUserInternalService(uc *biz.UserUsecase) *UserInternalService {
	return &UserInternalService{uc: uc}
}

func (s *UserInternalService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.UserInfo, error) {
	u, err := s.uc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &userv1.UserInfo{Id: u.ID, Username: u.Username, Status: string(u.Status)}, nil
}
