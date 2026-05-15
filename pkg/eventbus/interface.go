package eventbus

import (
	"context"
	"pkg/events"
)

// EventUnmarshaler - интерфейс для десериализации событий
type EventUnmarshaler interface {
	UnmarshalPayload(eventType string, data []byte) (events.Event, error)
}

// EventHandler - обработчик события
type EventHandler interface {
	Handle(ctx context.Context, event events.Event) error
}

// EventPublisherInterface - интерфейс публикатора событий
type EventPublisherInterface interface {
	// PublishAsync - асинхронная публикация (неблокирующая)
	PublishAsync(event events.Event) error

	// PublishSync - синхронная публикация (блокирующая)
	PublishSync(ctx context.Context, event events.Event) error

	// Shutdown - graceful shutdown
	Shutdown(ctx context.Context) error

	// Stats - статистика (опционально, для мониторинга)
	Stats() (submitted, published, failed, dropped int64)

	// IsHealthy - проверка здоровья
	IsHealthy() bool
}
