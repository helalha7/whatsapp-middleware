-- +goose Up
CREATE TABLE messages (
    id BIGINT NOT NULL AUTO_INCREMENT,
    conversation_id BIGINT NOT NULL,
    whatsapp_message_id VARCHAR(255) NULL,
    direction ENUM('inbound', 'outbound') NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

    CONSTRAINT messages_primary_key PRIMARY KEY (id),
    CONSTRAINT uq_messages_whatsapp_message_id UNIQUE KEY (whatsapp_message_id),
    CONSTRAINT fk_messages_conversation
        FOREIGN KEY (conversation_id) REFERENCES conversations(id)
        ON DELETE CASCADE,
    INDEX idx_messages_conversation_created
        (conversation_id, created_at, id)
);

-- +goose Down
DROP TABLE messages;
