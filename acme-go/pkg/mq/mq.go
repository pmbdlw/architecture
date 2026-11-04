// Package mq 异步消息规范：
// Watermill 统一抽象 Kafka / RabbitMQ，
// 业务代码只依赖 message.Publisher / message.Subscriber，中间件由配置 mq.driver 选择。
package mq

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/acme/acme-go/pkg/conf"
)

// 事件消息头（Watermill Metadata，Kafka header / AMQP header 通用）。
// 消息体只放业务数据 JSON（protojson），元数据全部放在消息头中。
const (
	HeaderID     = "acme_event_id"     // 事件 ID（UUID），消费端据此幂等
	HeaderType   = "acme_event_type"   // 事件类型，如 order.created
	HeaderSource = "acme_event_source" // 来源服务名
	HeaderTime   = "acme_event_time"   // 发生时间 RFC3339
	HeaderKey    = "acme_event_key"    // 聚合根 ID：Kafka 消息 key，保证同一聚合有序
)

// Logger Watermill 日志适配。
func Logger() watermill.LoggerAdapter { return watermill.NewSlogLogger(slog.Default()) }

// NewPublisher 按 mq.driver 创建发布者。
func NewPublisher(c conf.MQ) (message.Publisher, error) {
	switch c.Driver {
	case "kafka":
		return kafka.NewPublisher(kafka.PublisherConfig{
			Brokers: c.Kafka.BrokerList(),
			// 以聚合根 ID 为消息 key，同一订单的事件落在同一分区
			Marshaler: kafka.NewWithPartitioningMarshaler(func(_ string, msg *message.Message) (string, error) {
				return msg.Metadata.Get(HeaderKey), nil
			}),
			OverwriteSaramaConfig: KafkaProducerConfig(),
		}, Logger())
	case "rabbitmq":
		return amqp.NewPublisher(amqp.NewDurablePubSubConfig(c.RabbitMQ.URL, nil), Logger())
	case "gochannel":
		return GoChannel(), nil
	default:
		return nil, fmt.Errorf("unsupported mq driver %q", c.Driver)
	}
}

// KafkaProducerConfig 幂等生产者：acks=all、单连接单飞行请求，避免重试导致的重复与乱序。
func KafkaProducerConfig() *sarama.Config {
	cfg := kafka.DefaultSaramaSyncPublisherConfig()
	cfg.Version = sarama.V3_6_0_0
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Idempotent = true
	cfg.Net.MaxOpenRequests = 1
	return cfg
}

// NewSubscriber 按 mq.driver 创建订阅者。group 为消费组（= 服务名）：
// 同组多副本负载均衡，不同服务各自收到全量事件。
//   - Kafka：消费组 = group；
//   - RabbitMQ：每个 topic 一个 fanout 交换机，每个消费组一个持久化队列 <topic>_<group>。
func NewSubscriber(c conf.MQ, group string) (message.Subscriber, error) {
	switch c.Driver {
	case "kafka":
		cfg := kafka.DefaultSaramaSubscriberConfig()
		cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
		return kafka.NewSubscriber(kafka.SubscriberConfig{
			Brokers:               c.Kafka.BrokerList(),
			Unmarshaler:           kafka.DefaultMarshaler{},
			ConsumerGroup:         group,
			OverwriteSaramaConfig: cfg,
		}, Logger())
	case "rabbitmq":
		cfg := amqp.NewDurablePubSubConfig(c.RabbitMQ.URL, amqp.GenerateQueueNameTopicNameWithSuffix(group))
		return amqp.NewSubscriber(cfg, Logger())
	case "gochannel":
		return GoChannel(), nil
	default:
		return nil, fmt.Errorf("unsupported mq driver %q", c.Driver)
	}
}

var (
	goChannel     *gochannel.GoChannel
	goChannelOnce sync.Once
)

// GoChannel 进程内消息总线，仅用于测试（mq.driver=gochannel）。
func GoChannel() *gochannel.GoChannel {
	goChannelOnce.Do(func() {
		goChannel = gochannel.NewGoChannel(gochannel.Config{Persistent: true, OutputChannelBuffer: 64}, Logger())
	})
	return goChannel
}
