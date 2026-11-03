// Package migrations 按数据库方言分目录的 goose 迁移脚本。
// 已在任何共享环境执行过的脚本禁止修改，只能追加新版本。sqlite 目录仅用于测试。
package migrations

import "embed"

//go:embed mysql/*.sql postgres/*.sql sqlite/*.sql
var FS embed.FS
