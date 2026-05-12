package outbox

import (
	"context"
	"global_models/global_db"
)

// OutboxRepository - интерфейс для работы с outbox таблицей
type OutboxRepository interface {
	// SaveTx сохраняет сообщение в outbox (в рамках переданной транзакции)
	SaveTx(ctx context.Context, tx global_db.Tx, msg *OutboxMessage) error

	// GetPending возвращает список PENDING сообщений
	GetPending(ctx context.Context, pool global_db.Pool, limit int) ([]*OutboxMessage, error)

	// MarkSent отмечает сообщение как успешно отправленное
	MarkSent(ctx context.Context, pool global_db.Pool, id int64) error

	// MarkFailed отмечает сообщение как неудачное
	MarkFailed(ctx context.Context, pool global_db.Pool, id int64, errMsg string) error
}
