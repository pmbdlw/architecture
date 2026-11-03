// Package result 实现统一响应体：所有对外 HTTP 接口返回
//
//	{"code":"00000","message":"成功","data":{...},"traceId":"..."}
//
// 同时 HTTP 状态码与错误码一致（400/401/403/404/409/422/429/5xx）。
package result

import (
	"encoding/json"
	"net/http"

	"github.com/go-kratos/kratos/v2/encoding"
	kjson "github.com/go-kratos/kratos/v2/encoding/json"
	"github.com/go-kratos/kratos/v2/log"

	"github.com/acme/acme-go/pkg/errorx"
)

// TraceHeader 响应头中的链路 ID，与日志中的 trace_id 一致。
const TraceHeader = "X-Trace-Id"

// Envelope 统一响应体。
type Envelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	TraceID string          `json:"traceId,omitempty"`
}

var codec = encoding.GetCodec(kjson.Name)

// ResponseEncoder 把成功结果包装为 Envelope（proto 消息用 protojson 编码：int64 输出为字符串，字段为小驼峰）。
func ResponseEncoder(w http.ResponseWriter, _ *http.Request, v any) error {
	data := json.RawMessage("null")
	if v != nil {
		b, err := codec.Marshal(v)
		if err != nil {
			return err
		}
		data = b
	}
	return write(w, http.StatusOK, Envelope{
		Code:    errorx.Success.Reason,
		Message: errorx.Success.Message,
		Data:    data,
		TraceID: w.Header().Get(TraceHeader),
	})
}

// ErrorEncoder 返回一个把任意错误规整为 Envelope 的编码器；未知错误记录 ERROR 日志并隐藏细节。
func ErrorEncoder(logger log.Logger) func(http.ResponseWriter, *http.Request, error) {
	helper := log.NewHelper(logger)
	return func(w http.ResponseWriter, r *http.Request, err error) {
		se := errorx.Normalize(err)
		if se.Code >= 500 && se.Reason == errorx.SystemError.Reason {
			helper.Errorw("msg", "未处理异常", "path", r.URL.Path, "error", err)
		}
		_ = write(w, int(se.Code), Envelope{
			Code:    se.Reason,
			Message: se.Message,
			Data:    json.RawMessage("null"),
			TraceID: w.Header().Get(TraceHeader),
		})
	}
}

// Write 输出一个错误 Envelope（供网关等非 Kratos 处理器复用）。
func Write(w http.ResponseWriter, c errorx.Code, message string) {
	if message == "" {
		message = c.Message
	}
	_ = write(w, c.HTTP, Envelope{Code: c.Reason, Message: message, Data: json.RawMessage("null"),
		TraceID: w.Header().Get(TraceHeader)})
}

func write(w http.ResponseWriter, status int, e Envelope) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(e)
}
