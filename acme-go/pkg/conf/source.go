package conf

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/go-kratos/kratos/v2/config"
)

// memory 是一个只读的内存配置源（平台默认值、测试覆盖）。
type memorySource struct {
	kv *config.KeyValue
}

func memory(name string, data []byte) config.Source {
	return &memorySource{kv: &config.KeyValue{Key: name, Value: data, Format: "yaml"}}
}

func (m *memorySource) Load() ([]*config.KeyValue, error) { return []*config.KeyValue{m.kv}, nil }

func (m *memorySource) Watch() (config.Watcher, error) {
	return &noopWatcher{done: make(chan struct{})}, nil
}

type noopWatcher struct{ done chan struct{} }

func (w *noopWatcher) Next() ([]*config.KeyValue, error) {
	<-w.done
	return nil, context.Canceled
}

func (w *noopWatcher) Stop() error {
	select {
	case <-w.done:
	default:
		close(w.done)
	}
	return nil
}

// SplitHostPort 解析 "host:port"。
func SplitHostPort(addr string) (string, uint64, error) {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid address %q: %w", addr, err)
	}
	port, err := strconv.ParseUint(p, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in %q: %w", addr, err)
	}
	return host, port, nil
}
