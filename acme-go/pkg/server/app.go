package server

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport"

	"github.com/acme/acme-go/pkg/conf"
	reg "github.com/acme/acme-go/pkg/registry"
)

// NewApp 组装应用生命周期：启动服务器 → 注册到 Nacos；收到 SIGTERM → 注销 → 优雅停止（最长 StopTimeout）。
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
		opts = append(opts, kratos.Registrar(registrar))
	}
	return kratos.New(opts...)
}
