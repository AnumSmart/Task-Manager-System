-- +goose Up
-- +goose StatementBegin

-- Создание таблицы organizations
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    owner_id UUID NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    
    -- Внешний ключ
    CONSTRAINT fk_organizations_owner_id FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE RESTRICT,
    -- Уникальность имени для активных записей
    CONSTRAINT unique_organizations_name_deleted UNIQUE (name, deleted_at)
);

-- Создание индексов
CREATE INDEX idx_organizations_owner_id ON organizations(owner_id);
CREATE INDEX idx_organizations_deleted_at ON organizations(deleted_at);
CREATE INDEX idx_organizations_is_active ON organizations(is_active);

-- Комментарии
COMMENT ON TABLE organizations IS 'Organizations table';
COMMENT ON COLUMN organizations.id IS 'Organization unique identifier (UUID)';
COMMENT ON COLUMN organizations.name IS 'Organization name';
COMMENT ON COLUMN organizations.owner_id IS 'Organization owner user ID';
COMMENT ON COLUMN organizations.is_active IS 'Organization active status';
COMMENT ON COLUMN organizations.created_at IS 'Creation timestamp';
COMMENT ON COLUMN organizations.updated_at IS 'Last update timestamp';
COMMENT ON COLUMN organizations.deleted_at IS 'Soft delete timestamp (NULL if not deleted)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Удаляем индексы
DROP INDEX IF EXISTS idx_organizations_owner_id;
DROP INDEX IF EXISTS idx_organizations_deleted_at;
DROP INDEX IF EXISTS idx_organizations_is_active;

-- Удаляем таблицу
DROP TABLE IF EXISTS organizations CASCADE;

-- +goose StatementEnd