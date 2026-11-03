// Package biz 业务层：领域模型、用例与仓储接口。只依赖标准库与平台错误码，不依赖任何技术框架。
package biz

import (
	"context"

	"github.com/acme/acme-go/pkg/errorx"
)

// user-service 错误码，号段 A1000-A1999。
var (
	ErrUserNotFound    = errorx.Code{Reason: "A1001", Message: "用户不存在: %v", HTTP: 404}
	ErrUsernameTaken   = errorx.Code{Reason: "A1002", Message: "用户名已被占用: %v", HTTP: 409}
	ErrUserAlreadyGone = errorx.Code{Reason: "A1003", Message: "用户已被禁用", HTTP: 409}
)

type UserStatus string

const (
	UserActive   UserStatus = "ACTIVE"
	UserDisabled UserStatus = "DISABLED"
)

// User 用户聚合根：只包含业务状态与规则。
type User struct {
	ID       int64
	Username string
	Mobile   string
	Email    string
	Status   UserStatus
}

// RegisterUser 创建一个新用户（初始为启用状态）。
func RegisterUser(id int64, username, mobile, email string) *User {
	return &User{ID: id, Username: username, Mobile: mobile, Email: email, Status: UserActive}
}

// Disable 禁用用户。
func (u *User) Disable() error {
	if u.Status == UserDisabled {
		return ErrUserAlreadyGone.New()
	}
	u.Status = UserDisabled
	return nil
}

// UserRepo 仓储接口定义在业务层，实现在 data 层（依赖倒置）。
type UserRepo interface {
	NextID() int64
	// FindByID 不存在时返回 (nil, nil)。
	FindByID(ctx context.Context, id int64) (*User, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	Add(ctx context.Context, u *User) error
}

// RegisterCommand 注册用例入参。
type RegisterCommand struct {
	Username string
	Mobile   string
	Email    string
}

// UserUsecase 用例编排：不写 HTTP 细节，业务规则放在领域对象中。
type UserUsecase struct {
	repo UserRepo
}

func NewUserUsecase(repo UserRepo) *UserUsecase { return &UserUsecase{repo: repo} }

func (uc *UserUsecase) Register(ctx context.Context, cmd RegisterCommand) (*User, error) {
	exists, err := uc.repo.ExistsByUsername(ctx, cmd.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameTaken.New(cmd.Username)
	}
	u := RegisterUser(uc.repo.NextID(), cmd.Username, cmd.Mobile, cmd.Email)
	if err := uc.repo.Add(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (uc *UserUsecase) Get(ctx context.Context, id int64) (*User, error) {
	u, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound.New(id)
	}
	return u, nil
}
