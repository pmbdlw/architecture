package user_test

import (
	"context"
	"net/http"
	"testing"

	userv1 "github.com/acme/acme-go/api/user/v1"
	"github.com/acme/acme-go/app/user/internal/conf"
	"github.com/acme/acme-go/app/user/internal/server"
	"github.com/acme/acme-go/app/user/migrations"
	"github.com/acme/acme-go/pkg/archtest"
	"github.com/acme/acme-go/pkg/testkit"
)

func setup(t *testing.T) (http.Handler, *testkit.Signer, userv1.UserInternalServiceServer) {
	signer := testkit.NewSigner(t)
	c := &conf.Config{Bootstrap: testkit.Bootstrap("user-service")}
	db := testkit.DB(t, migrations.FS)
	hs, _ := server.New(server.Deps{Conf: c, Telemetry: testkit.Telemetry(t, "user-service"), DB: db,
		KeyFunc: signer.KeyFunc})
	return hs, signer, server.InternalService(db)
}

func TestAdminRegistersUserAndOtherServicesReadIt(t *testing.T) {
	h, signer, internal := setup(t)
	admin := signer.Token(t, "1", "ADMIN")

	resp := testkit.Do(t, h, "POST", "/api/v1/users", admin,
		`{"username":"alice","mobile":"13800001234","email":"alice@example.com"}`)
	if resp.Status != 200 || resp.Code != "00000" {
		t.Fatalf("register: %d %s %s", resp.Status, resp.Code, resp.Message)
	}
	if got := resp.DataField(t, "mobile"); got != "138****1234" {
		t.Errorf("mobile must be masked, got %v", got)
	}
	id, ok := resp.DataField(t, "id").(string)
	if !ok || id == "" {
		t.Fatalf("int64 id must be serialized as string, got %v", resp.DataField(t, "id"))
	}
	if resp.Header.Get("X-Trace-Id") == "" || resp.TraceID == "" {
		t.Error("trace id must be returned in header and body")
	}

	// 集群内 gRPC 契约：order-service 通过 UserInternalService 读取用户
	got := testkit.Do(t, h, "GET", "/api/v1/users/"+id, signer.Token(t, "1001"), "")
	info, err := internal.GetUser(context.Background(), &userv1.GetUserRequest{Id: mustInt(t, id)})
	if err != nil || info.Username != "alice" || info.Status != "ACTIVE" || got.Status != 200 {
		t.Fatalf("internal get: %v %v (http %d)", info, err, got.Status)
	}
}

func TestErrorsUseUnifiedResult(t *testing.T) {
	h, signer, _ := setup(t)
	admin := signer.Token(t, "1", "ADMIN")
	user := signer.Token(t, "1001", "USER")

	testkit.Do(t, h, "POST", "/api/v1/users", admin, `{"username":"bob"}`)
	cases := []struct {
		name, method, path, token, body string
		status                          int
		code                            string
	}{
		{"重复用户名", "POST", "/api/v1/users", admin, `{"username":"bob"}`, 409, "A1002"},
		{"用户不存在", "GET", "/api/v1/users/42", user, "", 404, "A1001"},
		{"参数错误", "POST", "/api/v1/users", admin, `{"username":"x","mobile":"123"}`, 400, "A0400"},
		{"非管理员", "POST", "/api/v1/users", user, `{"username":"carol"}`, 403, "A0403"},
		{"匿名访问", "GET", "/api/v1/users/1", "", "", 401, "A0401"},
	}
	for _, c := range cases {
		resp := testkit.Do(t, h, c.method, c.path, c.token, c.body)
		if resp.Status != c.status || resp.Code != c.code {
			t.Errorf("%s: got %d %s (%s), want %d %s", c.name, resp.Status, resp.Code, resp.Message, c.status, c.code)
		}
	}
}

func TestLayers(t *testing.T) {
	archtest.CheckLayers(t, ".")
}
