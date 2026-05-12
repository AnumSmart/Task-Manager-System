package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"global_models/global_db"
)

// PostgresOutboxRepository - реализация для PostgreSQL
type PostgresOutboxRepository struct{}

// Конструктор для PostgresOutboxRepository
func NewPostgresOutboxRepository() *PostgresOutboxRepository {
	return &PostgresOutboxRepository{}
}

// SaveTx сохраняет сообщение в outbox (использует переданную транзакцию)
func (p *PostgresOutboxRepository) SaveTx(ctx context.Context, tx global_db.Tx, msg *OutboxMessage) error {
	// Сериализуем событие в JSON
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// создаём строку запроса
	query := `
        INSERT INTO outbox (event_id, event_type, payload, routing_key, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
        RETURNING id
    `

	var id int64
	row := tx.QueryRow(ctx, query, msg.EventID, msg.EventType, payloadBytes, msg.RoutingKey, msg.Status)
	if err := row.Scan(&id); err != nil {
		return fmt.Errorf("insert outbox message: %w", err)
	}

	// переопределяем поле ID
	msg.ID = id

	return nil
}

// GetPending возвращает PENDING сообщения
func (r *PostgresOutboxRepository) GetPending(ctx context.Context, pool global_db.Pool, limit int) ([]*OutboxMessage, error) {
	// создаём строку запроса
	query := `
        SELECT id, event_id, event_type, payload, routing_key, status, retry_count, 
               created_at, updated_at, processed_at, last_error
        FROM outbox
        WHERE status = 'PENDING'
        ORDER BY created_at ASC
        LIMIT $1
    `
	// делаем запрос
	rows, err := pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query pending messages: %w", err)
	}

	// освобождаем ресурсы
	defer rows.Close()

	var messages []*OutboxMessage

	for rows.Next() {
		msg := &OutboxMessage{}
		var payloadBytes []byte

		err := rows.Scan(
			&msg.ID,
			&msg.EventID,
			&msg.EventType,
			&payloadBytes,
			&msg.RoutingKey,
			&msg.Status,
			&msg.RetryCount,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&msg.ProcessedAt,
			&msg.LastError,
		)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		// Сохраняем сырой JSON для последующей десериализации
		msg.PayloadRaw = payloadBytes
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return messages, nil
}

// MarkSent отмечает сообщение как отправленное
func (r *PostgresOutboxRepository) MarkSent(ctx context.Context, pool global_db.Pool, id int64) error {
	// создаём строку запроса
	query := `
        UPDATE outbox 
        SET status = 'SENT', processed_at = NOW(), updated_at = NOW()
        WHERE id = $1 AND status = 'PENDING'
    `
	// делаем запрос
	_, err := pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark as sent: %w", err)
	}
	return nil
}

// MarkFailed отмечает сообщение как неудачное
func (r *PostgresOutboxRepository) MarkFailed(ctx context.Context, pool global_db.Pool, id int64, errMsg string) error {
	// создаём строку запроса
	query := `
        UPDATE outbox 
        SET status = 'FAILED', last_error = $2, retry_count = retry_count + 1, updated_at = NOW()
        WHERE id = $1
    `
	// делаем запрос
	_, err := pool.Exec(ctx, query, id, errMsg)
	if err != nil {
		return fmt.Errorf("mark as failed: %w", err)
	}
	return nil
}
