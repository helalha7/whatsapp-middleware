-- +goose Up
CREATE TABLE received_inbound_messages (
    tenant_id BIGINT NOT NULL,
    whatsapp_message_id VARCHAR(255) NOT NULL,
    received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT received_inbound_messages_primary_key
        PRIMARY KEY (tenant_id, whatsapp_message_id),
    CONSTRAINT fk_received_inbound_messages_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id)
        ON DELETE CASCADE
);

-- +goose Down
DROP TABLE received_inbound_messages;
