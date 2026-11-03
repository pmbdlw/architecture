package gateway

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"

	"github.com/acme/acme-go/pkg/selector/gray"
)

// balancer 从注册中心订阅 <service>.http 的实例列表，按灰度规则筛选后轮询选择。
type balancer struct {
	discovery registry.Discovery
	log       *log.Helper
	mu        sync.Mutex
	services  map[string]*serviceNodes
}

type serviceNodes struct {
	nodes atomic.Pointer[[]*registry.ServiceInstance]
	next  atomic.Uint64
}

func newBalancer(d registry.Discovery, logger log.Logger) *balancer {
	return &balancer{discovery: d, log: log.NewHelper(logger), services: map[string]*serviceNodes{}}
}

// watch 开始订阅一个服务（幂等）。
func (b *balancer) watch(ctx context.Context, service string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.services[service]; ok {
		return
	}
	s := &serviceNodes{}
	empty := []*registry.ServiceInstance{}
	s.nodes.Store(&empty)
	b.services[service] = s
	w, err := b.discovery.Watch(ctx, service+".http")
	if err != nil {
		b.log.Errorw("msg", "订阅服务失败", "service", service, "error", err)
		return
	}
	go func() {
		defer func() { _ = w.Stop() }()
		for {
			list, err := w.Next()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				b.log.Warnw("msg", "服务实例更新失败", "service", service, "error", err)
				continue
			}
			s.nodes.Store(&list)
		}
	}()
}

// pick 返回一个实例的 http 地址；没有可用实例时返回空字符串。
func (b *balancer) pick(service, grayTag string) string {
	b.mu.Lock()
	s := b.services[service]
	b.mu.Unlock()
	if s == nil {
		return ""
	}
	nodes := gray.Select(*s.nodes.Load(), grayTag, func(n *registry.ServiceInstance) map[string]string { return n.Metadata })
	if len(nodes) == 0 {
		return ""
	}
	n := nodes[(s.next.Add(1)-1)%uint64(len(nodes))]
	for _, ep := range n.Endpoints {
		if strings.HasPrefix(ep, "http://") {
			return ep
		}
	}
	return ""
}
