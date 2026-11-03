// Package gray 实现灰度实例筛选（网关与服务间 gRPC 调用共用）：
//   - 请求带 x-gray-tag: v2：优先选择元数据 gray-tag=v2 的实例，没有则回落到稳定实例；
//   - 请求不带灰度标签：只选择未打标签的稳定实例，灰度实例不接收普通流量；
//   - 所有实例都打了标签时不过滤，可用性优先。
package gray

import (
	"context"

	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/selector"
)

const (
	// Header 灰度标签请求头（HTTP Header / gRPC Metadata，统一小写）。
	Header = "x-gray-tag"
	// MetadataKey 注册中心实例元数据中的灰度标签。
	MetadataKey = "gray-tag"
)

// Select 按灰度规则筛选实例。meta 用于读取实例元数据，便于网关与 gRPC 复用同一规则。
func Select[T any](nodes []T, tag string, meta func(T) map[string]string) []T {
	var stable, gray []T
	for _, n := range nodes {
		t := meta(n)[MetadataKey]
		switch {
		case t == "":
			stable = append(stable, n)
		case tag != "" && t == tag:
			gray = append(gray, n)
		}
	}
	if len(gray) > 0 {
		return gray
	}
	if len(stable) == 0 {
		return nodes
	}
	return stable
}

// Filter 是 Kratos 选择器的节点过滤器，从请求上下文读取灰度标签。
func Filter() selector.NodeFilter {
	return func(ctx context.Context, nodes []selector.Node) []selector.Node {
		return Select(nodes, TagFromContext(ctx), func(n selector.Node) map[string]string { return n.Metadata() })
	}
}

// TagFromContext 读取当前请求携带的灰度标签（由 metadata 中间件从上游请求头透传而来）。
func TagFromContext(ctx context.Context) string {
	if md, ok := metadata.FromServerContext(ctx); ok {
		if v := md.Get(Header); v != "" {
			return v
		}
	}
	if md, ok := metadata.FromClientContext(ctx); ok {
		return md.Get(Header)
	}
	return ""
}
