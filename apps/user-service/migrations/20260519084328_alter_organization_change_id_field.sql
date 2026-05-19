-- +goose Up
-- +goose StatementBegin

-- Удаляем индекс
DROP INDEX IF EXISTS idx_organizations_is_active;

-- Убираем DEFAULT у поля id
ALTER TABLE organizations ALTER COLUMN id DROP DEFAULT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Восстанавливаем DEFAULT
ALTER TABLE organizations ALTER COLUMN id SET DEFAULT gen_random_uuid();

-- Восстанавливаем индекс
CREATE INDEX idx_organizations_is_active ON organizations(is_active);

-- +goose StatementEnd
