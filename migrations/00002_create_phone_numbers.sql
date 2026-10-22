-- +goose Up
CREATE TABLE phone_numbers (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    tenant_id BIGINT UNSIGNED NOT NULL,
    phone_number_id VARCHAR(32) NOT NULL UNIQUE,
    display_phone_number VARCHAR(30),

    CONSTRAINT fk_phone_tenant
        FOREIGN KEY (tenant_id)
        REFERENCES tenants(id)
        ON DELETE CASCADE
);

-- +goose Down
DROP TABLE phone_numbers;
