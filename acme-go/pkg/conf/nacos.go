package conf

import (
	"context"
	"path"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// nacosSource 基于 nacos-sdk-go v2 的配置源：加载时读取一次，之后监听变更实时推送。
// （Kratos 官方 contrib 仍依赖 nacos-sdk-go v1，这里自行实现以统一 SDK 版本。）
type nacosSource struct {
	client config_client.IConfigClient
	group  string
	dataID string
}

func nacosConfig(client config_client.IConfigClient, group, dataID string) config.Source {
	return &nacosSource{client: client, group: group, dataID: dataID}
}

func (s *nacosSource) Load() ([]*config.KeyValue, error) {
	content, err := s.client.GetConfig(vo.ConfigParam{DataId: s.dataID, Group: s.group})
	if err != nil {
		return nil, err
	}
	return []*config.KeyValue{s.kv(content)}, nil
}

func (s *nacosSource) kv(content string) *config.KeyValue {
	format := path.Ext(s.dataID)
	if len(format) > 0 {
		format = format[1:]
	}
	return &config.KeyValue{Key: s.dataID, Value: []byte(content), Format: format}
}

func (s *nacosSource) Watch() (config.Watcher, error) {
	w := &nacosWatcher{source: s, ch: make(chan string, 1)}
	w.ctx, w.cancel = context.WithCancel(context.Background())
	err := s.client.ListenConfig(vo.ConfigParam{
		DataId: s.dataID,
		Group:  s.group,
		OnChange: func(_, _, _, data string) {
			select {
			case w.ch <- data:
			case <-w.ctx.Done():
			}
		},
	})
	return w, err
}

type nacosWatcher struct {
	source *nacosSource
	ch     chan string
	ctx    context.Context
	cancel context.CancelFunc
}

func (w *nacosWatcher) Next() ([]*config.KeyValue, error) {
	select {
	case data := <-w.ch:
		return []*config.KeyValue{w.source.kv(data)}, nil
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	}
}

func (w *nacosWatcher) Stop() error {
	w.cancel()
	return w.source.client.CancelListenConfig(vo.ConfigParam{DataId: w.source.dataID, Group: w.source.group})
}
