// Package migrations 按数据库方言分目录的 goose 迁移脚本（sqlite 仅用于测试）。
package migrations

import "embed"

//go:embed mysql/*.sql postgres/*.sql sqlite/*.sql
var FS embed.FS
