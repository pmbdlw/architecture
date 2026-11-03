package conf

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testConfig struct {
	Bootstrap
	Order struct {
		Feature bool `json:"feature"`
	} `json:"order"`
}

func TestLoadPrecedence(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(file, []byte(`
app:
  name: order-service
server:
  http:
    addr: ${HTTP_ADDR:0.0.0.0:8102}
order:
  feature: true
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HTTP_ADDR", "0.0.0.0:18080")
	t.Setenv("NACOS_ENABLED", "false")

	var c testConfig
	cfg, err := Load(file, &c, func() *Bootstrap { return &c.Bootstrap }, "client:\n  timeout: 5s\n")
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()

	if c.App.Name != "order-service" {
		t.Errorf("service file not applied: %q", c.App.Name)
	}
	if c.Server.HTTP.Addr != "0.0.0.0:18080" {
		t.Errorf("env placeholder not applied: %q", c.Server.HTTP.Addr)
	}
	if c.Server.GRPC.Addr != "0.0.0.0:9090" || c.Server.StopTimeout.Duration != 30*time.Second {
		t.Errorf("platform defaults not applied: %+v", c.Server)
	}
	if c.Client.Timeout.Duration != 5*time.Second {
		t.Errorf("override not applied: %v", c.Client.Timeout)
	}
	if c.Nacos.Enabled || !c.Order.Feature || c.MQ.Outbox.BatchSize != 100 {
		t.Errorf("typed values wrong: nacos=%v feature=%v batch=%d", c.Nacos.Enabled, c.Order.Feature, c.MQ.Outbox.BatchSize)
	}
}
