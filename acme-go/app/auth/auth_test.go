package auth_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang-jwt/jwt/v5"

	"github.com/acme/acme-go/app/auth/internal/authserver"
	"github.com/acme/acme-go/app/auth/internal/conf"
	"github.com/acme/acme-go/pkg/auth"
)

const issuer = "http://auth.test"

func newServer(t *testing.T) *httptest.Server {
	srv, err := authserver.New(conf.OAuth{
		Issuer: issuer,
		JWK:    conf.JWK{KeyID: "test-key"},
		Clients: []conf.Client{
			{ID: "acme-web", Public: true, RedirectURIs: []string{"http://localhost:3000/callback"},
				GrantTypes: []string{"authorization_code", "refresh_token"}, Scopes: []string{"openid", "order.read"}},
			{ID: "acme-internal", Secret: "s3cret", GrantTypes: []string{"client_credentials"}, Scopes: []string{"internal"}},
		},
		DemoUsers: []conf.DemoUser{{UID: 1001, Username: "alice", Password: "pw", Roles: []string{"USER"}}},
	}, log.DefaultLogger)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}

// verify 用 JWKS 端点公布的公钥校验令牌，与资源服务（pkg/auth）的做法一致。
func verify(t *testing.T, ts *httptest.Server, token string) *auth.Claims {
	t.Helper()
	resp, err := http.Get(ts.URL + "/oauth2/jwks")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	k, err := keyfunc.NewJWKSetJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	claims := &auth.Claims{}
	if _, err := jwt.ParseWithClaims(token, claims, k.Keyfunc, jwt.WithIssuer(issuer)); err != nil {
		t.Fatalf("token does not verify: %v", err)
	}
	return claims
}

func tokenFrom(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	tok, _ := body["access_token"].(string)
	if resp.StatusCode != 200 || tok == "" {
		t.Fatalf("token request failed: %d %v", resp.StatusCode, body)
	}
	return tok
}

func TestDiscoveryAndClientCredentials(t *testing.T) {
	ts := newServer(t)
	resp, err := http.Get(ts.URL + "/.well-known/openid-configuration")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("discovery: %v", err)
	}
	var doc map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&doc)
	resp.Body.Close()
	if doc["issuer"] != issuer || doc["jwks_uri"] != issuer+"/oauth2/jwks" {
		t.Fatalf("discovery doc: %v", doc)
	}

	req, _ := http.NewRequest("POST", ts.URL+"/oauth2/token",
		strings.NewReader("grant_type=client_credentials&scope=internal"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("acme-internal", "s3cret")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	claims := verify(t, ts, tokenFrom(t, resp))
	if claims.Subject != "acme-internal" || claims.Scope != "internal" || claims.UID != "" {
		t.Fatalf("client credentials claims: %+v", claims)
	}

	req, _ = http.NewRequest("POST", ts.URL+"/oauth2/token", strings.NewReader("grant_type=client_credentials"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("acme-internal", "wrong")
	if resp, _ := http.DefaultClient.Do(req); resp.StatusCode == 200 {
		t.Fatal("wrong secret must be rejected")
	}
}

func TestAuthorizationCodeWithPKCE(t *testing.T) {
	ts := newServer(t)
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk-acme-test"
	sum := sha256.Sum256([]byte(verifier))
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {"acme-web"},
		"redirect_uri":          {"http://localhost:3000/callback"},
		"scope":                 {"openid order.read"},
		"state":                 {"xyz"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}

	// 未登录：显示登录页
	resp, err := noRedirect.Get(ts.URL + "/oauth2/authorize?" + params.Encode())
	if err != nil || resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("login page: %v %v", resp.StatusCode, err)
	}
	// 登录成功：带授权码重定向回客户端
	authorize := func() string {
		form := url.Values{"username": {"alice"}, "password": {"pw"}}
		for k, v := range params {
			form[k] = v
		}
		resp, err := noRedirect.PostForm(ts.URL+"/oauth2/authorize", form)
		if err != nil || resp.StatusCode != http.StatusFound {
			t.Fatalf("authorize: %v %v", resp.StatusCode, err)
		}
		loc, _ := url.Parse(resp.Header.Get("Location"))
		if loc.Query().Get("code") == "" || loc.Query().Get("state") != "xyz" {
			t.Fatalf("redirect: %s", loc)
		}
		return loc.Query().Get("code")
	}
	exchange := func(code, v string) *http.Response {
		r, err := http.PostForm(ts.URL+"/oauth2/token", url.Values{
			"grant_type": {"authorization_code"}, "client_id": {"acme-web"}, "code": {code},
			"redirect_uri": {"http://localhost:3000/callback"}, "code_verifier": {v}})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	// 授权码一次性有效，验证失败后作废
	if r := exchange(authorize(), "wrong-verifier-wrong-verifier-wrong-verifier"); r.StatusCode == 200 {
		t.Fatal("wrong PKCE verifier must be rejected")
	}
	wrongPwd := url.Values{"username": {"alice"}, "password": {"nope"}}
	for k, v := range params {
		wrongPwd[k] = v
	}
	if r, _ := noRedirect.PostForm(ts.URL+"/oauth2/authorize", wrongPwd); r.StatusCode != 200 {
		t.Fatalf("wrong password must show login page again: %d", r.StatusCode)
	}
	claims := verify(t, ts, tokenFrom(t, exchange(authorize(), verifier)))
	if claims.UID != "1001" || len(claims.Roles) != 1 || claims.Roles[0] != "USER" || claims.Subject != "alice" {
		t.Fatalf("user claims: %+v", claims)
	}
}
