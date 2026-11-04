// Package authserver 认证中心：OAuth 2.1 授权码 + PKCE、客户端凭证，
// 签发 RS256 JWT（含 uid / roles 声明），并提供 JWKS 与 OIDC 发现端点。
//
// 这是一个精简实现，便于本地与测试环境自包含运行；生产环境推荐使用成熟的 IdP：
// Casdoor（Go）、Ory Hydra（Go）、Keycloak、阿里云 IDaaS、Amazon Cognito、Microsoft Entra ID。
// 只要签发的 JWT 带有 uid / roles 声明，资源服务端无需改动。
package authserver

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"html/template"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
	"github.com/golang-jwt/jwt/v5"

	"github.com/acme/acme-go/app/auth/internal/conf"
	"github.com/acme/acme-go/pkg/auth"
)

// Server 认证中心的 HTTP 处理器。
type Server struct {
	cfg   conf.OAuth
	key   *rsa.PrivateKey
	oauth *server.Server
	users map[string]conf.DemoUser
	mux   *http.ServeMux
}

func New(c conf.OAuth, logger log.Logger) (*Server, error) {
	key, generated, err := loadKey(c.JWK.PrivateKeyFile)
	if err != nil {
		return nil, err
	}
	if generated {
		log.NewHelper(logger).Warn("未配置 oauth.jwk.privateKeyFile，使用临时生成的 RSA 密钥——仅限本地开发，多副本部署会导致令牌互不认可")
	}
	s := &Server{cfg: c, key: key, users: map[string]conf.DemoUser{}, mux: http.NewServeMux()}
	for _, u := range c.DemoUsers {
		s.users[u.Username] = u
	}

	ttl := c.AccessTokenTTL.Duration
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	manager := manage.NewDefaultManager()
	manager.SetAuthorizeCodeTokenCfg(&manage.Config{AccessTokenExp: ttl, RefreshTokenExp: 7 * 24 * time.Hour, IsGenerateRefresh: true})
	manager.SetClientTokenCfg(&manage.Config{AccessTokenExp: 10 * time.Minute})
	manager.MustTokenStorage(store.NewMemoryTokenStore())
	manager.MapAccessGenerate(&jwtGenerator{s: s})
	clients := store.NewClientStore()
	for _, cl := range c.Clients {
		_ = clients.Set(cl.ID, &models.Client{ID: cl.ID, Secret: cl.Secret, Public: cl.Public,
			Domain: strings.Join(cl.RedirectURIs, ",")})
	}
	manager.MapClientStorage(clients)
	// 回调地址精确匹配（不做前缀 / 子域名匹配）
	manager.SetValidateURIHandler(func(registered, redirect string) error {
		if slices.Contains(strings.Split(registered, ","), redirect) {
			return nil
		}
		return errors.ErrInvalidRedirectURI
	})

	srv := server.NewServer(&server.Config{
		TokenType:                   "Bearer",
		AllowedResponseTypes:        []oauth2.ResponseType{oauth2.Code},
		AllowedGrantTypes:           []oauth2.GrantType{oauth2.AuthorizationCode, oauth2.ClientCredentials, oauth2.Refreshing},
		AllowedCodeChallengeMethods: []oauth2.CodeChallengeMethod{oauth2.CodeChallengeS256},
		ForcePKCE:                   true,
	}, manager)
	srv.SetClientInfoHandler(clientInfo)
	srv.SetClientAuthorizedHandler(s.grantAllowed)
	srv.SetClientScopeHandler(s.scopeAllowed)
	srv.SetUserAuthorizationHandler(s.login)
	s.oauth = srv

	s.mux.HandleFunc("GET /.well-known/openid-configuration", s.discovery)
	s.mux.HandleFunc("GET /oauth2/jwks", s.jwks)
	s.mux.HandleFunc("/oauth2/authorize", func(w http.ResponseWriter, r *http.Request) {
		if err := s.oauth.HandleAuthorizeRequest(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})
	s.mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		_ = s.oauth.HandleTokenRequest(w, r)
	})
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) discovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"issuer":                                s.cfg.Issuer,
		"authorization_endpoint":                s.cfg.Issuer + "/oauth2/authorize",
		"token_endpoint":                        s.cfg.Issuer + "/oauth2/token",
		"jwks_uri":                              s.cfg.Issuer + "/oauth2/jwks",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "client_credentials", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "none"},
	})
}

func (s *Server) jwks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"keys": []any{jwk(&s.key.PublicKey, s.cfg.JWK.KeyID)}})
}

// clientInfo 机密客户端用 HTTP Basic，公共客户端只在表单中提供 client_id。
func clientInfo(r *http.Request) (string, string, error) {
	if id, secret, ok := r.BasicAuth(); ok {
		return id, secret, nil
	}
	if id := r.FormValue("client_id"); id != "" {
		return id, "", nil
	}
	return "", "", errors.ErrInvalidClient
}

func (s *Server) client(id string) (conf.Client, bool) {
	for _, c := range s.cfg.Clients {
		if c.ID == id {
			return c, true
		}
	}
	return conf.Client{}, false
}

func (s *Server) grantAllowed(clientID string, grant oauth2.GrantType) (bool, error) {
	c, ok := s.client(clientID)
	return ok && slices.Contains(c.GrantTypes, string(grant)), nil
}

func (s *Server) scopeAllowed(tgr *oauth2.TokenGenerateRequest) (bool, error) {
	c, ok := s.client(tgr.ClientID)
	if !ok {
		return false, nil
	}
	for _, sc := range strings.Fields(tgr.Scope) {
		if !slices.Contains(c.Scopes, sc) {
			return false, nil
		}
	}
	return true, nil
}

var loginPage = template.Must(template.New("login").Parse(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">
<title>登录 · acme</title><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="font-family:system-ui,sans-serif;max-width:360px;margin:12vh auto;padding:0 16px">
<h1 style="font-size:22px">登录 acme</h1>
{{if .Error}}<p style="color:#b91c1c">{{.Error}}</p>{{end}}
<form method="post" action="/oauth2/authorize">
{{range $k, $v := .Params}}<input type="hidden" name="{{$k}}" value="{{$v}}">{{end}}
<p><label>用户名<br><input name="username" autocomplete="username" required></label></p>
<p><label>密码<br><input name="password" type="password" autocomplete="current-password" required></label></p>
<p><button type="submit">登录</button></p>
</form></body></html>`))

// login 授权端点的用户认证：GET 显示登录页，POST 校验账号密码并返回用户 ID。
func (s *Server) login(w http.ResponseWriter, r *http.Request) (string, error) {
	if r.Method == http.MethodPost {
		if u, ok := s.users[r.FormValue("username")]; ok && u.Password == r.FormValue("password") {
			return u.Username, nil
		}
	}
	params := map[string]string{}
	for _, k := range []string{"response_type", "client_id", "redirect_uri", "scope", "state",
		"code_challenge", "code_challenge_method"} {
		params[k] = r.FormValue(k)
	}
	data := map[string]any{"Params": params}
	if r.Method == http.MethodPost {
		data["Error"] = "用户名或密码错误"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return "", loginPage.Execute(w, data)
}

// jwtGenerator 签发 RS256 JWT：sub = 用户名或客户端 ID，uid / roles 来自用户目录。
type jwtGenerator struct {
	s *Server
}

func (g *jwtGenerator) Token(_ context.Context, data *oauth2.GenerateBasic, isGenRefresh bool) (string, string, error) {
	now := data.CreateAt
	sub := data.UserID
	if sub == "" {
		sub = data.Client.GetID()
	}
	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    g.s.cfg.Issuer,
			Subject:   sub,
			Audience:  jwt.ClaimStrings{data.Client.GetID()},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(data.TokenInfo.GetAccessExpiresIn())),
		},
		Scope: data.TokenInfo.GetScope(),
	}
	if u, ok := g.s.users[data.UserID]; ok {
		claims.UID = strconv.FormatInt(u.UID, 10)
		claims.Roles = u.Roles
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = g.s.cfg.JWK.KeyID
	access, err := token.SignedString(g.s.key)
	if err != nil {
		return "", "", err
	}
	refresh := ""
	if isGenRefresh {
		refresh = randomToken()
	}
	return access, refresh, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
