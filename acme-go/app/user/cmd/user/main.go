// user-service：用户域，账号与资料。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2/registry"

	"github.com/acme/acme-go/app/user/internal/conf"
	"github.com/acme/acme-go/app/user/internal/server"
	"github.com/acme/acme-go/app/user/migrations"
	"github.com/acme/acme-go/pkg/auth"
	pconf "github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/database"
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

	ctx := context.Background()
	tel, err := observability.Setup(ctx, &c.Bootstrap)
	if err != nil {
		return err
	}
	cleanup.Add(func() error { return tel.Shutdown(context.Background()) })

	db, err := database.Open(c.Data.Database)
	if err != nil {
		return err
	}
	if c.Data.Database.Migrate {
		if err := database.Migrate(ctx, db, c.Data.Database.Driver, migrations.FS); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	keyFunc, err := auth.RemoteKeyFunc(ctx, c.Auth.JWKSURL)
	if err != nil {
		return err
	}

	hs, gs := server.New(server.Deps{Conf: &c, Telemetry: tel, DB: db, KeyFunc: keyFunc})
	mgmt := pserver.NewManagement(c.Server.Management.Addr, map[string]pserver.Check{"db": database.Ping(db)})

	var registrar registry.Registrar
	if c.Nacos.Enabled {
		if registrar, err = preg.New(c.Nacos); err != nil {
			return err
		}
	}
	return pserver.NewApp(&c.Bootstrap, tel.Logger, registrar, hs, gs, mgmt).Run()
}
