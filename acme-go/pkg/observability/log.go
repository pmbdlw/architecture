// Package observability 提供日志、链路、指标的统一初始化。
package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"

	"github.com/acme/acme-go/pkg/conf"
)

// NewLogger 返回 Kratos Logger：K8s 中输出 JSON（含 trace_id / span_id），本地输出文本。
func NewLogger(app conf.App, c conf.Log) log.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(c.Level)}
	var handler slog.Handler
	if c.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	logger := &slogAdapter{logger: slog.New(handler)}
	return log.With(logger,
		"service", app.Name,
		"env", app.Env,
		"version", app.Version,
		"trace_id", tracing.TraceID(),
		"span_id", tracing.SpanID(),
	)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// slogAdapter 把 Kratos 的 key/value 日志转发到 log/slog。
type slogAdapter struct {
	logger *slog.Logger
}

func (a *slogAdapter) Log(level log.Level, keyvals ...any) error {
	var lvl slog.Level
	switch level {
	case log.LevelDebug:
		lvl = slog.LevelDebug
	case log.LevelWarn:
		lvl = slog.LevelWarn
	case log.LevelError, log.LevelFatal:
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	if !a.logger.Enabled(context.Background(), lvl) {
		return nil
	}
	msg := ""
	attrs := make([]any, 0, len(keyvals))
	for i := 0; i+1 < len(keyvals); i += 2 {
		key := fmt.Sprint(keyvals[i])
		if key == log.DefaultMessageKey {
			msg = fmt.Sprint(keyvals[i+1])
			continue
		}
		if v := keyvals[i+1]; v != "" {
			attrs = append(attrs, slog.Any(key, v))
		}
	}
	a.logger.Log(context.Background(), lvl, msg, attrs...)
	return nil
}
