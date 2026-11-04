-- 发件箱记录写入时的链路上下文（W3C traceparent），Relay 投递时以它为父节点，使“下单 → MQ → 消费”成为同一条链路
ALTER TABLE outbox_event ADD COLUMN trace_parent VARCHAR(64);
COMMENT ON COLUMN outbox_event.trace_parent IS 'W3C traceparent';
