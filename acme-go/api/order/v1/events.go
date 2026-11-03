package orderv1

// 订单域事件的 Topic 与事件类型。
const (
	// EventsTopic Kafka topic / RabbitMQ exchange，格式 <系统>.<领域>.events。
	EventsTopic = "acme.order.events"
	// EventOrderCreated 订单已创建。
	EventOrderCreated = "order.created"
)
