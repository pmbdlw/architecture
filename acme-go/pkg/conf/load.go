package conf

import (
	_ "embed"
	"fmt"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

//go:embed defaults.yaml
var defaultsYAML []byte

// Load 按平台优先级加载配置并扫描到 out（out 需内嵌 Bootstrap）。
// 返回的 config.Config 可用于 Watch 动态配置（Nacos 修改后实时生效）。
//
// overrides 用于测试：以最高优先级覆盖任意配置（YAML 文本）。
func Load(path string, out any, bootstrap func() *Bootstrap, overrides ...string) (config.Config, error) {
	base := []config.Source{memory("defaults.yaml", defaultsYAML)}
	if path != "" {
		base = append(base, file.NewSource(path))
	}
	tail := []config.Source{env.NewSource()}
	for i, o := range overrides {
		tail = append(tail, memory(fmt.Sprintf("override-%d.yaml", i), []byte(o)))
	}

	// 第一遍：不含 Nacos，读出 Nacos 连接信息
	c, err := build(append(append([]config.Source{}, base...), tail...), out)
	if err != nil {
		return nil, err
	}
	b := bootstrap()
	if !b.Nacos.Enabled {
		return c, nil
	}
	_ = c.Close()

	// 第二遍：插入 Nacos 共享配置与服务配置
	client, err := newConfigClient(b.Nacos)
	if err != nil {
		return nil, fmt.Errorf("create nacos config client: %w", err)
	}
	group := b.Nacos.Group
	sources := append([]config.Source{}, base...)
	sources = append(sources,
		nacosConfig(client, group, "acme-common.yaml"),
		nacosConfig(client, group, b.App.Name+".yaml"),
	)
	return build(append(sources, tail...), out)
}

func build(sources []config.Source, out any) (config.Config, error) {
	c := config.New(config.WithSource(sources...), config.WithResolveActualTypes(true))
	if err := c.Load(); err != nil {
		return nil, err
	}
	if err := c.Scan(out); err != nil {
		return nil, fmt.Errorf("scan config: %w", err)
	}
	return c, nil
}

func newConfigClient(n Nacos) (config_client.IConfigClient, error) {
	host, port, err := SplitHostPort(n.Addr)
	if err != nil {
		return nil, err
	}
	return clients.NewConfigClient(vo.NacosClientParam{
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
}
