package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Check 是一个就绪检查（如数据库 Ping）。
type Check func(ctx context.Context) error

// Management 管理端口服务器：
//
//	/healthz  存活探针（进程能响应即为存活，不检查下游依赖）
//	/readyz   就绪探针（检查数据库等关键依赖；停机开始后立即返回 503，配合 preStop 摘流量）
//	/metrics  Prometheus 指标
type Management struct {
	srv      *http.Server
	checks   map[string]Check
	stopping atomic.Bool
}

func NewManagement(addr string, checks map[string]Check) *Management {
	m := &Management{checks: checks}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", m.ready)
	mux.Handle("/metrics", promhttp.Handler())
	m.srv = &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	return m
}

// Handler 返回管理端口的处理器（测试中直接调用）。
func (m *Management) Handler() http.Handler { return m.srv.Handler }

func (m *Management) ready(w http.ResponseWriter, r *http.Request) {
	if m.stopping.Load() {
		http.Error(w, "stopping", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	for name, check := range m.checks {
		if err := check(ctx); err != nil {
			http.Error(w, name+": "+err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	_, _ = w.Write([]byte("ready"))
}

// Start 实现 transport.Server。
func (m *Management) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", m.srv.Addr)
	if err != nil {
		return err
	}
	m.srv.BaseContext = func(net.Listener) context.Context { return ctx }
	if err := m.srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Stop 实现 transport.Server。
func (m *Management) Stop(ctx context.Context) error {
	m.stopping.Store(true)
	return m.srv.Shutdown(ctx)
}
