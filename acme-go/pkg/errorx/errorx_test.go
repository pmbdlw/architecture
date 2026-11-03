package errorx

import (
	"testing"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/circuitbreaker"
)

func TestNormalize(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
		code int32
	}{
		{"biz", Code{"A1001", "用户不存在: %v", 404}.New(7), "A1001", 404},
		{"jwt", errors.Unauthorized("UNAUTHORIZED", "JWT token is missing"), "A0401", 401},
		{"breaker", circuitbreaker.ErrNotAllowed, "C0503", 503},
		{"unknown", errors.New(500, "", "sql: connection refused"), "B0001", 500},
	}
	for _, c := range cases {
		got := Normalize(c.err)
		if got.Reason != c.want || got.Code != c.code {
			t.Errorf("%s: got %s/%d, want %s/%d", c.name, got.Reason, got.Code, c.want, c.code)
		}
	}
	if Normalize(errors.New(500, "", "secret detail")).Message == "secret detail" {
		t.Error("unknown error message must be hidden")
	}
}
