-- +goose Up
CREATE TABLE chat_sessions (
    id BIGINT NOT NULL AUTO_INCREMENT,
    conversation_id BIGINT NOT NULL,
    started_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    last_activity_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

    CONSTRAINT chat_sessions_primary_key PRIMARY KEY (id),
    CONSTRAINT fk_chat_sessions_conversation
        FOREIGN KEY (conversation_id) REFERENCES conversations(id)
        ON DELETE CASCADE,
    INDEX idx_chat_sessions_conversation_activity
        (conversation_id, last_activity_at, id)
);

-- +goose Down
DROP TABLE chat_sessions;
