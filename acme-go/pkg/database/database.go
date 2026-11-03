// Package database 持久层规范（对应 Java 版 acme-starter-mybatis）：
// GORM 访问 MySQL / PostgreSQL，goose 按数据库方言管理版本化迁移脚本，事务通过 context 传递。
package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	stdlog "log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/plugin/optimisticlock"
	"gorm.io/plugin/soft_delete"

	"github.com/acme/acme-go/pkg/conf"
)

// Model 所有表的公共字段（数据库规范强制要求）：
//
//	id          BIGINT    主键，Sonyflake 生成，不使用自增
//	created_at  创建时间，GORM 自动填充
//	updated_at  更新时间，GORM 自动填充
//	deleted     逻辑删除标记 0/1
//	version     乐观锁版本号
type Model struct {
	ID        int64                 `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt time.Time             `gorm:"precision:3"`
	UpdatedAt time.Time             `gorm:"precision:3"`
	Deleted   soft_delete.DeletedAt `gorm:"softDelete:flag;default:0"`
	Version   optimisticlock.Version
}

// Open 按驱动打开数据库连接池。驱动只决定方言，代码与镜像不变：
// 切换 MySQL / PostgreSQL 只需修改 DB_DRIVER 与 DB_DSN。
func Open(c conf.Database) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch c.Driver {
	case "mysql":
		dialector = mysql.Open(c.DSN)
	case "postgres":
		dialector = postgres.Open(c.DSN)
	case "sqlite":
		dialector = sqlite.Open(c.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", c.Driver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		// 唯一约束冲突统一翻译为 gorm.ErrDuplicatedKey，便于做幂等
		TranslateError: true,
		NowFunc:        func() time.Time { return time.Now().UTC().Truncate(time.Millisecond) },
		// 慢 SQL（>200ms）与错误记 WARN；“记录不存在”属于正常业务结果，不记录
		Logger: gormlogger.New(stdlog.New(os.Stdout, "", stdlog.LstdFlags), gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
		}),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if c.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(c.MaxOpenConns)
	}
	if c.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	}
	if c.ConnMaxLifetime.Duration > 0 {
		sqlDB.SetConnMaxLifetime(c.ConnMaxLifetime.Duration)
	}
	return db, nil
}

var dialects = map[string]goose.Dialect{
	"mysql":    goose.DialectMySQL,
	"postgres": goose.DialectPostgres,
	"sqlite":   goose.DialectSQLite3,
}

// Migrate 执行 fsys/<driver>/ 目录下的 goose 迁移脚本（按数据库方言分目录）。
func Migrate(ctx context.Context, db *gorm.DB, driver string, fsys fs.FS) error {
	dialect, ok := dialects[driver]
	if !ok {
		return fmt.Errorf("unsupported database driver %q", driver)
	}
	sub, err := fs.Sub(fsys, driver)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(dialect, sqlDB, sub)
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}

// Ping 用作就绪检查。
func Ping(db *gorm.DB) func(context.Context) error {
	return func(ctx context.Context) error {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.PingContext(ctx)
	}
}

type txKey struct{}

// Transaction 在 context 中传递事务：仓储通过 DB(ctx) 自动加入当前事务。
type Transaction struct {
	db *gorm.DB
}

func NewTransaction(db *gorm.DB) *Transaction { return &Transaction{db: db} }

// InTx 在一个本地事务中执行 fn；fn 返回错误则回滚。
func (t *Transaction) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if InTransaction(ctx) {
		return fn(ctx)
	}
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	}, &sql.TxOptions{})
}

// DB 返回当前事务（若有）或基础连接。
func DB(ctx context.Context, base *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return base.WithContext(ctx)
}

// InTransaction 判断 ctx 是否处于事务中。
func InTransaction(ctx context.Context) bool {
	_, ok := ctx.Value(txKey{}).(*gorm.DB)
	return ok
}
