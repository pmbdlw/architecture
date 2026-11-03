// Package service 接口层：实现 proto 定义的接口，负责参数校验、身份解析与 DTO 转换。
package service

import (
	"context"
	"regexp"
	"strings"

	userv1 "github.com/acme/acme-go/api/user/v1"
	"github.com/acme/acme-go/app/user/internal/biz"
	"github.com/acme/acme-go/pkg/auth"
	"github.com/acme/acme-go/pkg/errorx"
)

var mobilePattern = regexp.MustCompile(`^1\d{10}$`)

// UserService 对外接口（经网关 /api/v1/users/**）。
type UserService struct {
	userv1.UnimplementedUserServiceServer
	uc *biz.UserUsecase
}

func NewUserService(uc *biz.UserUsecase) *UserService { return &UserService{uc: uc} }

// RegisterUser 注册用户，仅管理员可调用。
func (s *UserService) RegisterUser(ctx context.Context, req *userv1.RegisterUserRequest) (*userv1.UserReply, error) {
	if err := auth.RequireRole(ctx, "ADMIN"); err != nil {
		return nil, err
	}
	if err := validateRegister(req); err != nil {
		return nil, err
	}
	u, err := s.uc.Register(ctx, biz.RegisterCommand{Username: req.Username, Mobile: req.Mobile, Email: req.Email})
	if err != nil {
		return nil, err
	}
	return toReply(u), nil
}

func (s *UserService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.UserReply, error) {
	u, err := s.uc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return toReply(u), nil
}

func validateRegister(req *userv1.RegisterUserRequest) error {
	var problems []string
	if n := len([]rune(req.Username)); n < 3 || n > 64 {
		problems = append(problems, "username 长度需在 3-64 之间")
	}
	if req.Mobile != "" && !mobilePattern.MatchString(req.Mobile) {
		problems = append(problems, "手机号格式不正确")
	}
	if req.Email != "" && !strings.Contains(req.Email, "@") {
		problems = append(problems, "邮箱格式不正确")
	}
	if len(problems) > 0 {
		return errorx.BadRequest.Msg(strings.Join(problems, "; "))
	}
	return nil
}

// toReply 对外视图：手机号脱敏（日志、接口均不得输出完整个人敏感信息）。
func toReply(u *biz.User) *userv1.UserReply {
	return &userv1.UserReply{Id: u.ID, Username: u.Username, Mobile: maskMobile(u.Mobile), Email: u.Email,
		Status: string(u.Status)}
}

func maskMobile(m string) string {
	if len(m) < 7 {
		return m
	}
	return m[:3] + "****" + m[len(m)-4:]
}
