package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// HandlerServer 把任意 http.Handler（网关、认证中心）包装为 transport.Server，交给 Kratos App 管理优雅停机。
type HandlerServer struct {
	srv *http.Server
}

func NewHandlerServer(addr string, h http.Handler) *HandlerServer {
	return &HandlerServer{srv: &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 10 * time.Second}}
}

func (s *HandlerServer) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return err
	}
	s.srv.BaseContext = func(net.Listener) context.Context { return ctx }
	if err := s.srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *HandlerServer) Stop(ctx context.Context) error { return s.srv.Shutdown(ctx) }
