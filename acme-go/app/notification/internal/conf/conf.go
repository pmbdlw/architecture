// Package conf notification-service 的配置结构。
package conf

import pconf "github.com/acme/acme-go/pkg/conf"

type Config struct {
	pconf.Bootstrap
	Consumer Consumer `json:"consumer"`
}

// Consumer 消费重试策略：重试耗尽后进入死信 topic。
type Consumer struct {
	Retries         int            `json:"retries"`
	InitialInterval pconf.Duration `json:"initialInterval"`
}
