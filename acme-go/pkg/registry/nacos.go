// Package registry 创建 Nacos 注册中心（自建 Nacos 集群 / 阿里云 MSE Nacos）。
//
// Kratos 的 Nacos 注册实现按协议分别注册：<服务名>.http 与 <服务名>.grpc。
// 网关按 .http 发现，服务间 gRPC 调用按 .grpc 发现（见 pkg/client）。
package registry

import (
	nacosreg "github.com/go-kratos/kratos/contrib/registry/nacos/v2"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/acme/acme-go/pkg/conf"
)

// New 返回同时实现 registry.Registrar 与 registry.Discovery 的 Nacos 注册中心。
func New(n conf.Nacos) (*nacosreg.Registry, error) {
	host, port, err := conf.SplitHostPort(n.Addr)
	if err != nil {
		return nil, err
	}
	client, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig: &constant.ClientConfig{
			NamespaceId:         n.Namespace,
			Username:            n.Username,
			Password:            n.Password,
			NotLoadCacheAtStart: true,
			LogLevel:            "warn",
			LogDir:              "/tmp/nacos/log",
			CacheDir:            "/tmp/nacos/cache",
		},
		ServerConfigs: []constant.ServerConfig{*constant.NewServerConfig(host, port)},
	})
	if err != nil {
		return nil, err
	}
	return nacosreg.New(client, nacosreg.WithGroup(n.Group)), nil
}

// Metadata 返回注册到 Nacos 的实例元数据：灰度标签、版本、可用区。
func Metadata(app conf.App) map[string]string {
	md := map[string]string{"version": app.Version}
	if app.GrayTag != "" {
		md["gray-tag"] = app.GrayTag
	}
	if app.Zone != "" {
		md["zone"] = app.Zone
	}
	return md
}
