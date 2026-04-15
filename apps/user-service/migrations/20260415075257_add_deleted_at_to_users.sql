-- +goose Up
-- Добавляем поле deleted_at для soft delete
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP;

-- Создаём индекс для быстрого поиска неудалённых пользователей
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- Опционально: можно добавить комментарий
COMMENT ON COLUMN users.deleted_at IS 'Timestamp of soft delete, NULL means user is active';

-- +goose Down
-- Удаляем индекс
DROP INDEX IF EXISTS idx_users_deleted_at;

-- Удаляем колонку
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
