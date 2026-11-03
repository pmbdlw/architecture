// acme-auth：认证中心，OAuth 2.1 / OIDC，签发 JWT。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/acme/acme-go/app/auth/internal/authserver"
	"github.com/acme/acme-go/app/auth/internal/conf"
	pconf "github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/observability"
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

	tel, err := observability.Setup(context.Background(), &c.Bootstrap)
	if err != nil {
		return err
	}
	cleanup.Add(func() error { return tel.Shutdown(context.Background()) })

	srv, err := authserver.New(c.OAuth, tel.Logger)
	if err != nil {
		return err
	}
	mgmt := pserver.NewManagement(c.Server.Management.Addr, nil)
	return pserver.NewApp(&c.Bootstrap, tel.Logger, nil, pserver.NewHandlerServer(c.Server.HTTP.Addr, srv), mgmt).Run()
}
