-- +goose Up
CREATE TABLE consumed_event
(
    consumer    VARCHAR(64) NOT NULL,
    event_id    VARCHAR(36) NOT NULL,
    consumed_at DATETIME    NOT NULL,
    PRIMARY KEY (consumer, event_id)
);

-- +goose Down
DROP TABLE consumed_event;
