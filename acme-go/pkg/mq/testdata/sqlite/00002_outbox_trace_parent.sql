-- +goose Up
ALTER TABLE outbox_event ADD COLUMN trace_parent VARCHAR(64);

-- +goose Down
ALTER TABLE outbox_event DROP COLUMN trace_parent;
