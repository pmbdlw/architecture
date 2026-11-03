// Package conf user-service 的配置结构：平台公共段 + 本服务特有段。
package conf

import pconf "github.com/acme/acme-go/pkg/conf"

type Config struct {
	pconf.Bootstrap
}
