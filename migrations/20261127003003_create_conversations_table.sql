-- +goose Up
CREATE TABLE conversations (
    id BIGINT NOT NULL AUTO_INCREMENT,
    tenant_id BIGINT NOT NULL,
    customer_wa_id VARCHAR(50) NOT NULL,
    customer_name VARCHAR(255) NOT NULL DEFAULT '',
    last_message_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

    CONSTRAINT conversations_primary_key PRIMARY KEY (id),
    CONSTRAINT uq_conversations_tenant_customer
        UNIQUE KEY (tenant_id, customer_wa_id),
    CONSTRAINT fk_conversations_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id)
        ON DELETE CASCADE,
    INDEX idx_conversations_tenant_last_message
        (tenant_id, last_message_at, id)
);

-- +goose Down
DROP TABLE conversations;
