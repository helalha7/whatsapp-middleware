-- +goose Up
CREATE TABLE tenants(
    id BIGINT NOT NULL AUTO_INCREMENT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    version INT NOT NULL DEFAULT 1,
    name VARCHAR(150) NOT NULL,
    waba_id VARCHAR(100) NOT NULL,
    phone_number_id VARCHAR(100) NOT NULL,
    display_phone_number VARCHAR(30) NOT NULL,
    access_token TEXT NOT NULL,

    CONSTRAINT tenants_primary_key PRIMARY KEY (id),
    CONSTRAINT uq_tenants_waba_id UNIQUE KEY (waba_id),
    CONSTRAINT uq_tenants_phone_number_id UNIQUE KEY (phone_number_id),
    CONSTRAINT uq_tenants_display_phone_number UNIQUE KEY (display_phone_number)
);
-- +goose Down
DROP TABLE tenants;
