// Package testkit 服务级集成测试工具：
// 进程内 SQLite 执行真实迁移脚本、本地签发 JWT、内存注册中心、进程内消息总线。
// 需要验证真实 MySQL / PostgreSQL / Kafka 行为时，改用 Testcontainers（CI 提供 Docker）。
package testkit

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io/fs"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"github.com/acme/acme-go/pkg/auth"
	"github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/database"
	"github.com/acme/acme-go/pkg/observability"
)

var dbSeq atomic.Int64

// DB 创建一个独立的内存 SQLite 库，并执行 fsys/sqlite 下的迁移脚本。
func DB(t testing.TB, migrations fs.FS) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:testdb%d?mode=memory&cache=shared&_pragma=foreign_keys(1)", dbSeq.Add(1))
	db, err := database.Open(conf.Database{Driver: "sqlite", DSN: dsn, MaxOpenConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(context.Background(), db, "sqlite", migrations); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// Issuer 测试令牌的签发者。
const Issuer = "http://acme-auth.test"

// Signer 本地 RSA 密钥签发 JWT，并提供对应的 KeyFunc 给 auth.Server 使用。
type Signer struct {
	key *rsa.PrivateKey
}

func NewSigner(t testing.TB) *Signer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return &Signer{key: key}
}

func (s *Signer) KeyFunc(*jwt.Token) (any, error) { return &s.key.PublicKey, nil }

// Token 签发一个用户令牌；uid 为空表示客户端凭证令牌。
func (s *Signer) Token(t testing.TB, uid string, roles ...string) string {
	t.Helper()
	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   uid,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		UID:   uid,
		Roles: roles,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.key)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// Telemetry 测试用可观测组件（不上报链路）。
func Telemetry(t testing.TB, service string) *observability.Telemetry {
	t.Helper()
	b := &conf.Bootstrap{App: conf.App{Name: service, Env: "test"},
		Observability: conf.Observability{Log: conf.Log{Format: "text", Level: "warn"}, Tracing: conf.Tracing{Sampling: 1}}}
	tel, err := observability.Setup(context.Background(), b)
	if err != nil {
		t.Fatal(err)
	}
	return tel
}

// Bootstrap 测试用平台配置：签发者与测试令牌一致，Outbox 不加行锁（SQLite）。
func Bootstrap(service string) conf.Bootstrap {
	return conf.Bootstrap{
		App:    conf.App{Name: service, Env: "test"},
		Auth:   conf.Auth{Issuer: Issuer},
		MQ:     conf.MQ{Driver: "gochannel", Outbox: conf.Outbox{BatchSize: 100, MaxRetries: 10}},
		Client: conf.Client{Timeout: conf.Duration{Duration: time.Second}},
	}
}
