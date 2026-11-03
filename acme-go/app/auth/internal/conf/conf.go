// Package conf 认证中心配置。
package conf

import pconf "github.com/acme/acme-go/pkg/conf"

type Config struct {
	pconf.Bootstrap
	OAuth OAuth `json:"oauth"`
}

type OAuth struct {
	// Issuer 对外签发的 iss，必须与资源服务配置的 AUTH_ISSUER_URI 完全一致（如 https://auth.acme.com）
	Issuer  string   `json:"issuer"`
	JWK     JWK      `json:"jwk"`
	Clients []Client `json:"clients"`
	// DemoUsers 演示账号，仅用于本地与测试环境；生产接入 user-service 或企业 IdP
	DemoUsers      []DemoUser     `json:"demoUsers"`
	AccessTokenTTL pconf.Duration `json:"accessTokenTTL"`
}

// JWK 签名密钥：多副本部署必须使用同一把密钥，PEM 文件由 K8s Secret（来自 Vault / KMS）挂载。
type JWK struct {
	KeyID          string `json:"keyId"`
	PrivateKeyFile string `json:"privateKeyFile"`
}

type Client struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
	// Public 公共客户端（SPA / App）：无密钥，必须使用 PKCE
	Public bool `json:"public"`
	// RedirectURIs 允许的回调地址（精确匹配）
	RedirectURIs []string `json:"redirectUris"`
	GrantTypes   []string `json:"grantTypes"`
	Scopes       []string `json:"scopes"`
}

type DemoUser struct {
	UID      int64    `json:"uid"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}
