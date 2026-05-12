package outbox

import (
	"context"
	"global_models/global_db"
	"log"
	"pkg/configs"
	"pkg/events"
	"time"
)

// Relay - реле для отправки событий из outbox
type Relay struct {
	repo      OutboxRepository           // интерфейс репозитрия для работы с outbox логикой
	pool      global_db.Pool             // общая абстрация для БД
	publisher *events.EventPublisher     // экземпляр event publisher
	registry  *EventRegistry             // экземпляр реестра событий
	config    *configs.OutboxRelayConfig // конфиг реле
	stopCh    chan struct{}              // стоп-канал
}

// конструктор для реле outbox
func NewRelay(
	repo OutboxRepository,
	pool global_db.Pool,
	publisher *events.EventPublisher,
	registry *EventRegistry,
	config *configs.OutboxRelayConfig,
) *Relay {
	return &Relay{
		repo:      repo,
		pool:      pool,
		publisher: publisher,
		registry:  registry,
		config:    config,
		stopCh:    make(chan struct{}),
	}
}

// Start запускает реле
func (r *Relay) Start(ctx context.Context) {
	ticker := time.NewTicker(r.config.PollInterval)
	defer ticker.Stop()

	log.Printf("[Outbox Relay] started: poll_interval=%v, batch_size=%d", r.config.PollInterval, r.config.BatchSize)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Outbox Relay] stopping (context done)...")
			return
		case <-r.stopCh:
			log.Println("[Outbox Relay] stopped by signal")
			return
		case <-ticker.C:
			r.processPending(ctx)
		}
	}
}

// Stop останавливает реле
func (r *Relay) Stop() {
	close(r.stopCh)
}

// processPending обрабатывает ожидающие сообщения
func (r *Relay) processPending(ctx context.Context) {
	messages, err := r.repo.GetPending(ctx, r.pool, r.config.BatchSize)
	if err != nil {
		log.Printf("[Outbox Relay] failed to get pending messages: %v", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	log.Printf("[Outbox Relay] processing %d pending messages", len(messages))

	for _, msg := range messages {
		r.publishMessage(ctx, msg)
	}
}

// publishMessage публикует одно сообщение
func (r *Relay) publishMessage(ctx context.Context, msg *OutboxMessage) {
	// Восстанавливаем событие из сырого JSON
	event, err := r.registry.UnmarshalPayload(msg.EventType, msg.PayloadRaw)
	if err != nil {
		log.Printf("[Outbox Relay] failed to unmarshal payload (id=%d, type=%s): %v", msg.ID, msg.EventType, err)

		if markErr := r.repo.MarkFailed(ctx, r.pool, msg.ID, err.Error()); markErr != nil {
			log.Printf("[Outbox Relay] failed to mark message as failed: %v", markErr)
		}
		return
	}

	// Публикуем событие с таймаутом
	publishCtx, cancel := context.WithTimeout(ctx, r.config.PublishTimeout)
	defer cancel()

	if err := r.publisher.PublishSync(publishCtx, event); err != nil {
		log.Printf("[Outbox Relay] failed to publish %s (id=%d): %v",
			msg.EventType, msg.ID, err)

		if markErr := r.repo.MarkFailed(ctx, r.pool, msg.ID, err.Error()); markErr != nil {
			log.Printf("[Outbox Relay] failed to mark message as failed: %v", markErr)
		}
		return
	}

	// Отмечаем как успешно отправленное
	if err := r.repo.MarkSent(ctx, r.pool, msg.ID); err != nil {
		log.Printf("[Outbox Relay] failed to mark message as sent (id=%d): %v", msg.ID, err)
		return
	}

	log.Printf("[Outbox Relay] successfully published %s (id=%d)", msg.EventType, msg.ID)
}
