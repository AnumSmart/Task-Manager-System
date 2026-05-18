-- +goose Up
-- +goose StatementBegin

-- Создание таблицы outbox
CREATE TABLE IF NOT EXISTS outbox (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    routing_key VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    
    CONSTRAINT chk_outbox_status CHECK (status IN ('PENDING', 'SENT', 'FAILED'))
);

-- Минимальный набор индексов для работы
CREATE INDEX idx_outbox_pending ON outbox(status, created_at) WHERE status = 'PENDING';
CREATE INDEX idx_outbox_event_id ON outbox(event_id);

-- Комментарии
COMMENT ON TABLE outbox IS 'Transactional Outbox table';
COMMENT ON COLUMN outbox.id IS 'Sequential identifier';
COMMENT ON COLUMN outbox.event_id IS 'Unique event identifier';
COMMENT ON COLUMN outbox.event_type IS 'Event type';
COMMENT ON COLUMN outbox.payload IS 'Event payload in JSON';
COMMENT ON COLUMN outbox.routing_key IS 'RabbitMQ routing key';
COMMENT ON COLUMN outbox.status IS 'PENDING, SENT, FAILED';
COMMENT ON COLUMN outbox.retry_count IS 'Number of attempts';
COMMENT ON COLUMN outbox.created_at IS 'Creation timestamp';
COMMENT ON COLUMN outbox.updated_at IS 'Last update timestamp';
COMMENT ON COLUMN outbox.processed_at IS 'Success timestamp';
COMMENT ON COLUMN outbox.last_error IS 'Last error message';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_outbox_pending;
DROP INDEX IF EXISTS idx_outbox_event_id;
DROP TABLE IF EXISTS outbox CASCADE;

-- +goose StatementEnd