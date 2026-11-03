package testkit

import (
	"context"
	"sync"

	"github.com/go-kratos/kratos/v2/registry"
)

// Registry 内存注册中心，测试中替代 Nacos（同时实现 Registrar 与 Discovery）。
type Registry struct {
	mu        sync.Mutex
	instances map[string][]*registry.ServiceInstance
	watchers  map[string][]*watcher
}

func NewRegistry() *Registry {
	return &Registry{instances: map[string][]*registry.ServiceInstance{}, watchers: map[string][]*watcher{}}
}

func (r *Registry) Register(_ context.Context, si *registry.ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances[si.Name] = append(r.instances[si.Name], si)
	r.notify(si.Name)
	return nil
}

func (r *Registry) Deregister(_ context.Context, si *registry.ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.instances[si.Name][:0]
	for _, s := range r.instances[si.Name] {
		if s.ID != si.ID {
			list = append(list, s)
		}
	}
	r.instances[si.Name] = list
	r.notify(si.Name)
	return nil
}

func (r *Registry) GetService(_ context.Context, name string) ([]*registry.ServiceInstance, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*registry.ServiceInstance(nil), r.instances[name]...), nil
}

func (r *Registry) Watch(ctx context.Context, name string) (registry.Watcher, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w := &watcher{ch: make(chan []*registry.ServiceInstance, 8), ctx: ctx}
	w.ch <- append([]*registry.ServiceInstance(nil), r.instances[name]...)
	r.watchers[name] = append(r.watchers[name], w)
	return w, nil
}

func (r *Registry) notify(name string) {
	for _, w := range r.watchers[name] {
		select {
		case w.ch <- append([]*registry.ServiceInstance(nil), r.instances[name]...):
		default:
		}
	}
}

type watcher struct {
	ch  chan []*registry.ServiceInstance
	ctx context.Context
}

func (w *watcher) Next() ([]*registry.ServiceInstance, error) {
	select {
	case list := <-w.ch:
		return list, nil
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	}
}

func (w *watcher) Stop() error { return nil }
