// order-service：订单域，下单与订单查询，发布订单事件。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2/registry"

	userv1 "github.com/acme/acme-go/api/user/v1"
	"github.com/acme/acme-go/app/order/internal/conf"
	"github.com/acme/acme-go/app/order/internal/server"
	"github.com/acme/acme-go/app/order/migrations"
	"github.com/acme/acme-go/pkg/auth"
	"github.com/acme/acme-go/pkg/client"
	pconf "github.com/acme/acme-go/pkg/conf"
	"github.com/acme/acme-go/pkg/database"
	"github.com/acme/acme-go/pkg/mq"
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
	defer cfg.Close()

	ctx := context.Background()
	tel, err := observability.Setup(ctx, &c.Bootstrap)
	if err != nil {
		return err
	}
	defer tel.Shutdown(context.Background())

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
	publisher, err := mq.NewPublisher(c.MQ)
	if err != nil {
		return err
	}
	defer publisher.Close()

	var reg interface {
		registry.Registrar
		registry.Discovery
	}
	var registrar registry.Registrar
	var discovery registry.Discovery
	if c.Nacos.Enabled {
		if reg, err = preg.New(c.Nacos); err != nil {
			return err
		}
		registrar, discovery = reg, reg
	}
	conn, err := client.Dial(ctx, discovery, c.Client, "user-service")
	if err != nil {
		return err
	}
	defer conn.Close()

	hs, relay := server.New(server.Deps{Conf: &c, Telemetry: tel, DB: db, KeyFunc: keyFunc,
		Users: userv1.NewUserInternalServiceClient(conn), Publisher: publisher})
	mgmt := pserver.NewManagement(c.Server.Management.Addr, map[string]pserver.Check{"db": database.Ping(db)})
	return pserver.NewApp(&c.Bootstrap, tel.Logger, registrar, hs, relay, mgmt).Run()
}
