-- +goose Up
CREATE TABLE outbox_event
(
    event_id     VARCHAR(36)  NOT NULL PRIMARY KEY,
    topic        VARCHAR(128) NOT NULL,
    event_type   VARCHAR(128) NOT NULL,
    aggregate_id VARCHAR(64)  NOT NULL,
    payload      TEXT         NOT NULL,
    source       VARCHAR(64)  NOT NULL,
    status       SMALLINT     NOT NULL DEFAULT 0,
    retry_count  INT          NOT NULL DEFAULT 0,
    created_at   DATETIME     NOT NULL,
    sent_at      DATETIME
);

-- +goose Down
DROP TABLE outbox_event;
