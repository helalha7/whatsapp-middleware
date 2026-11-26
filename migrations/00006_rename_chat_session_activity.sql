-- +goose Up
ALTER TABLE chat_sessions
    CHANGE COLUMN last_inbound_at last_activity_at
        DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6);

-- +goose Down
ALTER TABLE chat_sessions
    CHANGE COLUMN last_activity_at last_inbound_at
        DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6);
