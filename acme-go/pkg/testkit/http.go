package testkit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Response 解析后的统一响应。
type Response struct {
	Status  int
	Header  http.Header
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	TraceID string          `json:"traceId"`
}

// Do 对 handler 发起请求并解析统一响应体。token 为空表示匿名请求。
func Do(t testing.TB, h http.Handler, method, path, token, body string) Response {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	resp := Response{Status: rec.Code, Header: rec.Header()}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("%s %s: invalid response body %q: %v", method, path, rec.Body.String(), err)
	}
	return resp
}

// DataField 读取 data 中的字段（字符串形式）。
func (r Response) DataField(t testing.TB, field string) any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(r.Data, &m); err != nil {
		t.Fatalf("data is not an object: %s", r.Data)
	}
	return m[field]
}
