// Package conf 定义所有服务共用的启动配置，并按平台约定的优先级加载：
//
//	① 平台默认值（本包内嵌 defaults.yaml，优先级最低）
//	② 服务 configs/config.yaml（随镜像发布）
//	③ Nacos：acme-common.yaml、<服务名>.yaml（命名空间 = 环境，可热更新）
//	④ 环境变量（Helm values / platform-env / Secret），通过 ${VAR:默认值} 占位符生效
package conf

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Bootstrap 是平台公共配置段，各服务在自己的配置结构体中内嵌它。
type Bootstrap struct {
	App           App           `json:"app"`
	Server        Server        `json:"server"`
	Nacos         Nacos         `json:"nacos"`
	Observability Observability `json:"observability"`
	Auth          Auth          `json:"auth"`
	Data          Data          `json:"data"`
	MQ            MQ            `json:"mq"`
	Client        Client        `json:"client"`
}

type App struct {
	Name    string `json:"name"`
	Env     string `json:"env"`
	Version string `json:"version"`
	// GrayTag 非空表示灰度实例，写入注册中心元数据 gray-tag。
	GrayTag string `json:"grayTag"`
	Zone    string `json:"zone"`
	// ServedByHeader 为 true 时在响应头 X-Served-By 中返回处理实例（观察负载均衡 / 灰度用）。
	// 会暴露 Pod 名，仅本地集群与测试环境开启。
	ServedByHeader bool `json:"servedByHeader"`
}

type Server struct {
	HTTP       Listen `json:"http"`
	GRPC       Listen `json:"grpc"`
	Management Listen `json:"management"`
	// StopTimeout 优雅停机等待时间，需小于 Pod terminationGracePeriodSeconds。
	StopTimeout Duration `json:"stopTimeout"`
	// DrainDelay 从注册中心注销后继续服务的时间：等调用方（网关、其他服务）收到实例下线推送，再停止服务器。
	DrainDelay Duration `json:"drainDelay"`
}

type Listen struct {
	Addr    string   `json:"addr"`
	Timeout Duration `json:"timeout"`
}

type Nacos struct {
	Enabled   bool   `json:"enabled"`
	Addr      string `json:"addr"`
	Namespace string `json:"namespace"`
	Group     string `json:"group"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type Observability struct {
	Log     Log     `json:"log"`
	Tracing Tracing `json:"tracing"`
}

type Log struct {
	// Format：text（本地）/ json（K8s，供 Loki / SLS 采集）
	Format string `json:"format"`
	Level  string `json:"level"`
}

type Tracing struct {
	Enabled  bool    `json:"enabled"`
	Endpoint string  `json:"endpoint"`
	Sampling float64 `json:"sampling"`
}

type Auth struct {
	Issuer  string `json:"issuer"`
	JWKSURL string `json:"jwksUrl"`
}

type Data struct {
	Database Database `json:"database"`
	Redis    Redis    `json:"redis"`
}

type Database struct {
	// Driver：mysql / postgres / sqlite（sqlite 仅用于测试）
	Driver          string   `json:"driver"`
	DSN             string   `json:"dsn"`
	MaxOpenConns    int      `json:"maxOpenConns"`
	MaxIdleConns    int      `json:"maxIdleConns"`
	ConnMaxLifetime Duration `json:"connMaxLifetime"`
	// Migrate 启动时执行 goose 迁移；生产可改为发布前的 K8s Job 执行。
	Migrate bool `json:"migrate"`
}

type Redis struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type MQ struct {
	// Driver：kafka / rabbitmq / gochannel（gochannel 仅用于测试）
	Driver   string   `json:"driver"`
	Kafka    Kafka    `json:"kafka"`
	RabbitMQ RabbitMQ `json:"rabbitmq"`
	Outbox   Outbox   `json:"outbox"`
}

type Kafka struct {
	Brokers string `json:"brokers"`
}

// BrokerList 把逗号分隔的地址拆成列表。
func (k Kafka) BrokerList() []string {
	var out []string
	for _, b := range strings.Split(k.Brokers, ",") {
		if b = strings.TrimSpace(b); b != "" {
			out = append(out, b)
		}
	}
	return out
}

type RabbitMQ struct {
	URL string `json:"url"`
}

type Outbox struct {
	PollInterval Duration `json:"pollInterval"`
	BatchSize    int      `json:"batchSize"`
	MaxRetries   int      `json:"maxRetries"`
	// LockClause：MySQL 8 / PostgreSQL 使用 FOR UPDATE SKIP LOCKED；SQLite（测试）置空。
	LockClause string `json:"lockClause"`
}

type Client struct {
	// Timeout 服务间 gRPC 调用默认超时。
	Timeout Duration `json:"timeout"`
	// Endpoints 未启用 Nacos 时（本地开发）按服务名直连的地址，如 user-service: 127.0.0.1:9101。
	Endpoints map[string]string `json:"endpoints"`
}

// Duration 支持在配置中书写 "3s"、"500ms" 这样的字符串。
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch x := v.(type) {
	case string:
		parsed, err := time.ParseDuration(x)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", x, err)
		}
		d.Duration = parsed
	case float64:
		d.Duration = time.Duration(x)
	default:
		return fmt.Errorf("invalid duration %v", v)
	}
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}
