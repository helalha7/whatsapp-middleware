-- +goose Up
CREATE TABLE chat_sessions (
    id BIGINT NOT NULL AUTO_INCREMENT,
    conversation_id BIGINT NOT NULL,
    started_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    last_inbound_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

    CONSTRAINT chat_sessions_primary_key PRIMARY KEY (id),
    CONSTRAINT fk_chat_sessions_conversation
        FOREIGN KEY (conversation_id) REFERENCES conversations(id)
        ON DELETE CASCADE,
    INDEX idx_chat_sessions_conversation_activity
        (conversation_id, last_inbound_at, id)
);

ALTER TABLE messages
    ADD COLUMN chat_session_id BIGINT NULL AFTER conversation_id,
    ADD CONSTRAINT fk_messages_chat_session
        FOREIGN KEY (chat_session_id) REFERENCES chat_sessions(id)
        ON DELETE SET NULL,
    ADD INDEX idx_messages_chat_session_created
        (chat_session_id, created_at, id);

-- +goose Down
ALTER TABLE messages
    DROP FOREIGN KEY fk_messages_chat_session,
    DROP INDEX idx_messages_chat_session_created,
    DROP COLUMN chat_session_id;

DROP TABLE chat_sessions;
