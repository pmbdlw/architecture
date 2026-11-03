// Package conf 网关配置。gateway 段放在 Nacos 的 acme-gateway.yaml 中可热更新（路由、灰度、限流）。
package conf

import pconf "github.com/acme/acme-go/pkg/conf"

type Config struct {
	pconf.Bootstrap
	Gateway Gateway `json:"gateway"`
}

type Gateway struct {
	Routes []Route `json:"routes"`
	// PublicPrefixes 无需登录即可访问的路径前缀
	PublicPrefixes []string       `json:"publicPrefixes"`
	Timeout        pconf.Duration `json:"timeout"`
	RateLimit      RateLimit      `json:"rateLimit"`
	Breaker        Breaker        `json:"breaker"`
	Gray           Gray           `json:"gray"`
	CORS           CORS           `json:"cors"`
}

// Route 按路径前缀转发到注册中心中的服务（<service>.http）。
type Route struct {
	ID      string `json:"id"`
	Prefix  string `json:"prefix"`
	Service string `json:"service"`
}

// RateLimit 令牌桶（GCRA）存放在 Redis，多副本网关共享额度；Redis 故障时放行（可用性优先）。
type RateLimit struct {
	Enabled bool `json:"enabled"`
	// PerSecond 每秒补充速率，Burst 突发容量
	PerSecond int `json:"perSecond"`
	Burst     int `json:"burst"`
}

// Breaker 每条路由一个熔断器：最近窗口内请求数 ≥ MinRequests 且失败率 ≥ FailureRatio 时打开。
type Breaker struct {
	MinRequests  uint32         `json:"minRequests"`
	FailureRatio float64        `json:"failureRatio"`
	OpenTimeout  pconf.Duration `json:"openTimeout"`
	Interval     pconf.Duration `json:"interval"`
}

// Gray 灰度规则，修改后实时生效。
type Gray struct {
	Enabled bool     `json:"enabled"`
	Tag     string   `json:"tag"`
	UIDs    []string `json:"uids"`
	// Percentage 按用户 ID 哈希放量 0-100
	Percentage int `json:"percentage"`
	// AllowClientHeader 是否信任客户端自带的 X-Gray-Tag（仅 test 环境打开，便于测试指定版本）
	AllowClientHeader bool `json:"allowClientHeader"`
}

type CORS struct {
	AllowedOrigins []string `json:"allowedOrigins"`
}
