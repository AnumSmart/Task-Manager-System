-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    telegram_id BIGINT UNIQUE,
    telegram_username VARCHAR(255),
    role VARCHAR(50) NOT NULL CHECK (role IN ('owner', 'manager', 'employee')),
    status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'suspended')),
    full_name VARCHAR(255) NOT NULL,
    organization_id UUID NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP
);

CREATE INDEX idx_users_organization_id ON users(organization_id);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_role ON users(role);

-- +goose Down
DROP TABLE IF EXISTS users;