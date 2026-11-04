package observability

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/acme/acme-go/pkg/conf"
)

// Telemetry 汇总一个进程的可观测组件。
type Telemetry struct {
	Logger   log.Logger
	Meter    metric.Meter
	shutdown []func(context.Context) error
}

// Setup 初始化日志、链路（OTLP/HTTP 上报）与指标（Prometheus 格式，由管理端口 /metrics 暴露）。
func Setup(ctx context.Context, b *conf.Bootstrap) (*Telemetry, error) {
	t := &Telemetry{Logger: NewLogger(b.App, b.Observability.Log)}
	// OTEL_RESOURCE_ATTRIBUTES（Helm Chart 注入 Pod 名、灰度标签等）与服务自身属性合并
	res, err := resource.New(ctx, resource.WithFromEnv(), resource.WithAttributes(
		attribute.String("service.name", b.App.Name),
		attribute.String("service.version", b.App.Version),
		attribute.String("deployment.environment", b.App.Env),
	))
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(b.Observability.Tracing.Sampling))
	tpOpts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res), sdktrace.WithSampler(sampler)}
	if b.Observability.Tracing.Enabled {
		exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(b.Observability.Tracing.Endpoint))
		if err != nil {
			return nil, err
		}
		tpOpts = append(tpOpts, sdktrace.WithBatcher(exporter))
	}
	tp := sdktrace.NewTracerProvider(tpOpts...)
	otel.SetTracerProvider(tp)
	t.shutdown = append(t.shutdown, tp.Shutdown)

	exporter, err := otelprom.New()
	if err != nil {
		return nil, err
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter), sdkmetric.WithResource(res))
	otel.SetMeterProvider(mp)
	t.Meter = mp.Meter(b.App.Name)
	t.shutdown = append(t.shutdown, mp.Shutdown)
	return t, nil
}

// Shutdown 刷新并关闭链路与指标导出。
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var errs []error
	for _, fn := range t.shutdown {
		errs = append(errs, fn(ctx))
	}
	return errors.Join(errs...)
}
