// Package auth 实现资源服务器侧的 JWT 校验（对应 Java 版 acme-starter-security）。
//
// 令牌由认证中心签发（acme-auth / Casdoor / Keycloak / 阿里云 IDaaS 均可），
// 服务通过 JWKS 拉取公钥验签，并校验 iss。约定的声明：
//
//	uid    当前用户 ID（服务以此识别用户，绝不信任请求参数中的 userId）
//	roles  角色列表，如 ["ADMIN","USER"]
//	scope  空格分隔的授权范围
package auth

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/go-kratos/kratos/v2/middleware"
	kjwt "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/golang-jwt/jwt/v5"

	"github.com/acme/acme-go/pkg/errorx"
)

// Claims 平台约定的 JWT 声明。
type Claims struct {
	jwt.RegisteredClaims
	UID   string   `json:"uid,omitempty"`
	Roles []string `json:"roles,omitempty"`
	Scope string   `json:"scope,omitempty"`
}

// RemoteKeyFunc 从认证中心的 JWKS 地址获取公钥（后台定期刷新，支持密钥轮换）。
func RemoteKeyFunc(ctx context.Context, jwksURL string) (jwt.Keyfunc, error) {
	k, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, err
	}
	return k.Keyfunc, nil
}

// Server 返回 JWT 校验中间件；publicOperations 中的操作（如健康检查）跳过校验。
func Server(keyFunc jwt.Keyfunc, issuer string, publicOperations ...string) middleware.Middleware {
	verify := kjwt.Server(keyFunc,
		kjwt.WithSigningMethod(jwt.SigningMethodRS256),
		kjwt.WithClaims(func() jwt.Claims { return &Claims{} }),
	)
	checkIssuer := func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if c, ok := FromContext(ctx); !ok || c.Issuer != issuer {
				return nil, errorx.Unauthorized.New()
			}
			return handler(ctx, req)
		}
	}
	public := map[string]bool{}
	for _, op := range publicOperations {
		public[op] = true
	}
	return selector.Server(verify, checkIssuer).
		Match(func(_ context.Context, operation string) bool { return !public[operation] }).
		Build()
}

// FromContext 读取当前请求的令牌声明。
func FromContext(ctx context.Context) (*Claims, bool) {
	c, ok := kjwt.FromContext(ctx)
	if !ok {
		return nil, false
	}
	claims, ok := c.(*Claims)
	return claims, ok
}

// UID 返回当前用户 ID；令牌中没有 uid（如客户端凭证令牌）时返回 A0401。
func UID(ctx context.Context) (int64, error) {
	c, ok := FromContext(ctx)
	if !ok || c.UID == "" {
		return 0, errorx.Unauthorized.New()
	}
	uid, err := strconv.ParseInt(c.UID, 10, 64)
	if err != nil {
		return 0, errorx.Unauthorized.New()
	}
	return uid, nil
}

// HasRole 判断当前用户是否拥有角色。
func HasRole(ctx context.Context, role string) bool {
	c, ok := FromContext(ctx)
	return ok && slices.Contains(c.Roles, role)
}

// RequireRole 不满足角色时返回 A0403。
func RequireRole(ctx context.Context, role string) error {
	if !HasRole(ctx, role) {
		return errorx.Forbidden.New()
	}
	return nil
}

// HasScope 判断令牌是否包含授权范围。
func HasScope(ctx context.Context, scope string) bool {
	c, ok := FromContext(ctx)
	return ok && slices.Contains(strings.Fields(c.Scope), scope)
}
