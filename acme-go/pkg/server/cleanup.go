package server

import (
	"fmt"
	"os"
)

// Cleanup 收集进程退出前要执行的清理动作（关闭连接、刷新链路数据），按注册的相反顺序执行并报告失败。
type Cleanup struct {
	fns []func() error
}

// Add 注册一个清理动作。
func (c *Cleanup) Add(fn func() error) { c.fns = append(c.fns, fn) }

// Run 执行全部清理动作，通常在 main 中 defer 调用。
func (c *Cleanup) Run() {
	for i := len(c.fns) - 1; i >= 0; i-- {
		if err := c.fns[i](); err != nil {
			fmt.Fprintln(os.Stderr, "cleanup:", err)
		}
	}
}
