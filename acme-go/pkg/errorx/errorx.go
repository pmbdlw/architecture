// Package errorx 定义平台统一错误码。
//
// 编码规则与 Java 版一致（参考《阿里巴巴 Java 开发手册》）：5 位字符串，
// A 开头为调用方错误、B 开头为本系统错误、C 开头为第三方/下游错误；平台保留 0000-0999，
// 各服务按号段分配（user-service A1xxx、order-service A2xxx ……）。
//
// 实现上直接复用 Kratos errors：Reason 存放业务错误码，Code 存放 HTTP 状态码。
// 经 gRPC 传输时 Kratos 会把它们放进 ErrorInfo，调用方用 errors.FromError 可原样还原。
package errorx

import (
	"fmt"
	"regexp"

	"github.com/go-kratos/kratos/v2/errors"
)

// Code 是一个错误码定义。Message 可包含 %v 占位符，在 New 时格式化。
type Code struct {
	Reason  string
	Message string
	HTTP    int
}

// New 创建该错误码的错误实例。
func (c Code) New(args ...any) *errors.Error {
	msg := c.Message
	if len(args) > 0 {
		msg = fmt.Sprintf(c.Message, args...)
	}
	return errors.New(c.HTTP, c.Reason, msg)
}

// Is 判断 err 是否为该错误码。
func (c Code) Is(err error) bool {
	return err != nil && errors.Reason(err) == c.Reason
}

// 平台通用错误码（号段 0000-0999）。
var (
	Success           = Code{"00000", "成功", 200}
	BadRequest        = Code{"A0400", "请求参数错误", 400}
	Unauthorized      = Code{"A0401", "未登录或令牌已失效", 401}
	Forbidden         = Code{"A0403", "无访问权限", 403}
	NotFound          = Code{"A0404", "资源不存在", 404}
	Conflict          = Code{"A0409", "资源状态冲突", 409}
	TooManyRequests   = Code{"A0429", "请求过于频繁", 429}
	SystemError       = Code{"B0001", "系统繁忙，请稍后再试", 500}
	RemoteError       = Code{"C0001", "下游服务调用失败", 502}
	RemoteUnavailable = Code{"C0503", "下游服务暂不可用", 503}
)

var bizReason = regexp.MustCompile(`^[ABC]\d{4}$`)

// IsBizReason 判断 reason 是否是平台格式的业务错误码。
func IsBizReason(reason string) bool {
	return bizReason.MatchString(reason)
}

// Normalize 把任意错误规整为平台错误码：
//   - 已是平台错误码：原样返回；
//   - 框架错误（JWT 中间件、熔断器、404 路由等）：按 HTTP 状态映射；
//   - 未知错误：B0001，隐藏内部细节。
func Normalize(err error) *errors.Error {
	se := errors.FromError(err)
	if IsBizReason(se.Reason) {
		return se
	}
	var c Code
	switch {
	case se.Code == 400:
		c = BadRequest
	case se.Code == 401:
		c = Unauthorized
	case se.Code == 403:
		c = Forbidden
	case se.Code == 404:
		c = NotFound
	case se.Code == 429:
		c = TooManyRequests
	case se.Code == 503 || se.Code == 504:
		c = RemoteUnavailable
	default:
		c = SystemError
	}
	return c.New()
}

// Msg 使用自定义提示语创建该错误码的错误实例（如参数校验的具体原因）。
func (c Code) Msg(message string) *errors.Error {
	return errors.New(c.HTTP, c.Reason, message)
}
