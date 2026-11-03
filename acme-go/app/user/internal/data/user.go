// Package data 数据层：仓储实现（GORM），把技术细节挡在业务层之外。
package data

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/acme/acme-go/app/user/internal/biz"
	"github.com/acme/acme-go/pkg/database"
)

// UserPO user_account 表映射。
type UserPO struct {
	database.Model
	Username string
	Mobile   string
	Email    string
	Status   string
}

func (UserPO) TableName() string { return "user_account" }

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) biz.UserRepo { return &userRepo{db: db} }

func (r *userRepo) NextID() int64 { return database.NextID() }

func (r *userRepo) FindByID(ctx context.Context, id int64) (*biz.User, error) {
	var po UserPO
	err := database.DB(ctx, r.db).First(&po, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &biz.User{ID: po.ID, Username: po.Username, Mobile: po.Mobile, Email: po.Email,
		Status: biz.UserStatus(po.Status)}, nil
}

func (r *userRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var n int64
	err := database.DB(ctx, r.db).Model(&UserPO{}).Where("username = ?", username).Count(&n).Error
	return n > 0, err
}

func (r *userRepo) Add(ctx context.Context, u *biz.User) error {
	po := UserPO{Model: database.Model{ID: u.ID}, Username: u.Username, Mobile: u.Mobile, Email: u.Email,
		Status: string(u.Status)}
	err := database.DB(ctx, r.db).Create(&po).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		// 并发注册同名用户：由唯一索引兜底
		return biz.ErrUsernameTaken.New(u.Username)
	}
	return err
}
