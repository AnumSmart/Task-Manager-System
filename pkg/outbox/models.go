package outbox

import (
	"pkg/events"
	"time"
)

type Status string

const (
	StatusPending Status = "PENDING" // Ожидает отправки в RabbitMQ
	StatusSent    Status = "SENT"    // Успешно отправлено
	StatusFailed  Status = "FAILED"  // Ошибка при отправке (после всех retry)
)

// OutboxMessage представляет запись в outbox таблице.
// Каждая запись соответствует одному событию, которое должно быть отправлено в RabbitMQ.
// Паттерн Transactional Outbox гарантирует, что событие будет отправлено
// только после успешного коммита бизнес-транзакции.
type OutboxMessage struct {
	ID          int64        `db:"id"`           // ID - первичный ключ, автоинкремент (BIGSERIAL в PostgreSQL).
	EventID     string       `db:"event_id"`     // EventID - уникальный идентификатор события (UUID) (дедупликация)
	EventType   string       `db:"event_type"`   // EventType - тип события (стринговая константа) (Примеры: "user.created", "user.telegram_linked"...)
	Payload     events.Event `db:"payload"`      // Payload - само событие, реализующее интерфейс events.Event.
	PayloadRaw  []byte       `db:"-"`            // PayloadRaw - сырое JSON представление события, прочитанное из БД. (Тег `db:"-"` означает, что это поле не маппится на колонку БД.)
	RoutingKey  string       `db:"routing_key"`  // RoutingKey - ключ маршрутизации RabbitMQ. (Примеры: "user.created", "user.telegram_linked", "task.#"...)
	Status      Status       `db:"status"`       // Status - текущий статус обработки события. (Relay вычитывает только PENDING записи)
	RetryCount  int          `db:"retry_count"`  // RetryCount - количество попыток отправки
	CreatedAt   time.Time    `db:"created_at"`   // CreatedAt - время создания записи (timestamp with time zone)
	UpdatedAt   time.Time    `db:"updated_at"`   // UpdatedAt - время последнего обновления записи
	ProcessedAt *time.Time   `db:"processed_at"` // ProcessedAt - время успешной отправки события (NULL для PENDING и FAILED)
	LastError   *string      `db:"last_error"`   // LastError - текст последней ошибки при отправке (Заполняется только для FAILED записей)
}

// Важное замечание по разделению Payload и PayloadRaw:
//
// При сохранении (SaveTx):
//   - Используется Payload (реализует Event)
//   - Сериализуется в JSON и сохраняется в колонку payload
//
// При чтении (GetPending):
//   - Читаем JSON из колонки payload в PayloadRaw ([]byte)
//   - Payload остаётся nil
//   - Затем через EventRegistry.UnmarshalPayload() восстанавливаем конкретное событие
//
// Это сделано потому, что pkg/outbox не знает о конкретных типах событий
// (user-created, task-assigned и т.д.). Регистрация типов происходит на
// уровне конкретного сервиса через EventRegistry.
