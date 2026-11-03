// Package archtest 把团队分层规范变成可执行的测试（对应 Java 版 ArchUnit 规则）。
//
// 服务内部采用 Kratos 推荐布局：
//
//	internal/service  接口层：实现 proto 定义的 HTTP / gRPC 接口、消息消费入口
//	internal/biz      业务层：领域模型 + 用例 + 仓储/网关接口（不依赖任何技术框架）
//	internal/data     数据层：仓储实现、数据库、下游 gRPC 客户端、发件箱
//	internal/server   装配：把 service 注册到 HTTP / gRPC 服务器
package archtest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Rule 某一层禁止导入的包（按前缀或包含匹配）。
type Rule struct {
	Layer     string
	Forbidden []string
	Reason    string
}

// DefaultRules 平台默认分层规则。
var DefaultRules = []Rule{
	{"biz", []string{"/internal/data", "/internal/service", "/internal/server"}, "biz 不得依赖外层"},
	{"biz", []string{"gorm.io/", "net/http", "google.golang.org/grpc", "github.com/go-kratos/kratos/v2/transport",
		"github.com/ThreeDotsLabs/", "github.com/redis/"}, "biz 不依赖数据库 / 传输 / 消息框架"},
	{"service", []string{"/internal/data", "gorm.io/"}, "service 不得直接访问数据层"},
	{"data", []string{"/internal/service", "/internal/server"}, "data 不得依赖接口层"},
}

// CheckLayers 检查 dir/internal/<layer> 下所有 Go 源文件（含测试以外）的导入是否违反规则。
func CheckLayers(t *testing.T, dir string, rules ...Rule) {
	t.Helper()
	if len(rules) == 0 {
		rules = DefaultRules
	}
	for _, rule := range rules {
		root := filepath.Join(dir, "internal", rule.Layer)
		if _, err := os.Stat(root); err != nil {
			continue // 服务没有这一层（如网关）
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, perr := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if perr != nil {
				t.Errorf("parse %s: %v", path, perr)
				return nil
			}
			for _, imp := range f.Imports {
				p, _ := strconv.Unquote(imp.Path.Value)
				for _, bad := range rule.Forbidden {
					if strings.Contains(p, bad) {
						t.Errorf("%s 导入了 %s（%s）", path, p, rule.Reason)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Errorf("walk %s: %v", root, err)
		}
	}
}
