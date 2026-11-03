// acme-gateway：统一入口，路由、鉴权、限流、熔断降级、灰度打标。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/redis/go-redis/v9"

	"github.com/acme/acme-go/app/gateway/internal/conf"
	"github.com/acme/acme-go/app/gateway/internal/gateway"
	"github.com/acme/acme-go/pkg/auth"
	pconf "github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/observability"
	preg "github.com/acme/acme-go/pkg/registry"
	pserver "github.com/acme/acme-go/pkg/server"
)

var configPath = flag.String("conf", "configs/config.yaml", "配置文件路径")

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var c conf.Config
	cfg, err := pconf.Load(*configPath, &c, func() *pconf.Bootstrap { return &c.Bootstrap })
	if err != nil {
		return err
	}
	var cleanup pserver.Cleanup
	defer cleanup.Run()
	cleanup.Add(cfg.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tel, err := observability.Setup(ctx, &c.Bootstrap)
	if err != nil {
		return err
	}
	cleanup.Add(func() error { return tel.Shutdown(context.Background()) })

	keyFunc, err := auth.RemoteKeyFunc(ctx, c.Auth.JWKSURL)
	if err != nil {
		return err
	}
	if !c.Nacos.Enabled {
		return fmt.Errorf("网关依赖 Nacos 发现后端服务，请设置 NACOS_ENABLED=true")
	}
	discovery, err := preg.New(c.Nacos)
	if err != nil {
		return err
	}
	rdb := redis.NewClient(&redis.Options{Addr: c.Data.Redis.Addr, Password: c.Data.Redis.Password, DB: c.Data.Redis.DB})
	cleanup.Add(rdb.Close)

	gw := gateway.New(ctx, c.Gateway, c.Auth.Issuer, keyFunc, discovery, rdb, tel.Logger)
	// Nacos 中修改路由 / 灰度 / 限流后实时生效
	if err := cfg.Watch("gateway", func(_ string, v config.Value) {
		var next conf.Gateway
		if err := v.Scan(&next); err == nil {
			gw.Update(next)
		}
	}); err != nil {
		return err
	}

	// 限流在 Redis 故障时放行，因此就绪检查不依赖 Redis，避免 Redis 抖动导致网关整体摘流量
	mgmt := pserver.NewManagement(c.Server.Management.Addr, nil)
	return pserver.NewApp(&c.Bootstrap, tel.Logger, nil, pserver.NewHandlerServer(c.Server.HTTP.Addr, gw), mgmt).Run()
}
