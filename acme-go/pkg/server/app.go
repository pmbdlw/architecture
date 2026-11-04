package server

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport"

	"github.com/acme/acme-go/pkg/conf"
	reg "github.com/acme/acme-go/pkg/registry"
)

// NewApp 组装应用生命周期：启动服务器 → 注册到 Nacos；
// 收到 SIGTERM → 注销 → 等待 DrainDelay（调用方摘除本实例）→ 优雅停止服务器（最长 StopTimeout）。
func NewApp(b *conf.Bootstrap, logger log.Logger, registrar registry.Registrar, servers ...transport.Server) *kratos.App {
	opts := []kratos.Option{
		kratos.Name(b.App.Name),
		kratos.Version(b.App.Version),
		kratos.Metadata(reg.Metadata(b.App)),
		kratos.Logger(logger),
		kratos.Server(servers...),
		kratos.StopTimeout(b.Server.StopTimeout.Duration),
	}
	if registrar != nil {
		opts = append(opts, kratos.Registrar(drainRegistrar{Registrar: registrar, delay: b.Server.DrainDelay.Duration}))
	}
	return kratos.New(opts...)
}

// drainRegistrar 注销后等待一段时间再返回：Kratos 在注销完成后才停止服务器，
// 这段时间里实例照常处理请求，调用方收到 Nacos 推送后不再选中它，滚动发布不产生失败请求。
type drainRegistrar struct {
	registry.Registrar
	delay time.Duration
}

func (r drainRegistrar) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	err := r.Registrar.Deregister(ctx, service)
	if r.delay > 0 {
		select {
		case <-time.After(r.delay):
		case <-ctx.Done():
		}
	}
	return err
}
