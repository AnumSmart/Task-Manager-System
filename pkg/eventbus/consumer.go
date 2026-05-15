package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"pkg/configs"
	"pkg/events"
	"pkg/rabbitmq"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// EventConsumer - консьюмер событий
type EventConsumer struct {
	broker          rabbitmq.BrokerInterface
	config          *configs.ConsumerConfig
	eventRegistry   *EventRegistry
	handlerRegistry *HandlerRegistry
	middlewares     []Middleware

	// метрики
	received  atomic.Int64
	processed atomic.Int64
	failed    atomic.Int64
}

// Middleware - middleware для обработчиков
type Middleware func(next EventHandler) EventHandler

// NewEventConsumer создаёт нового консьюмера
func NewEventConsumer(
	broker rabbitmq.BrokerInterface,
	config *configs.ConsumerConfig,
	eventRegistry *EventRegistry,
	handlerRegistry *HandlerRegistry,
) *EventConsumer {
	if config.MaxProcessingTime == 0 {
		config.MaxProcessingTime = 30 * time.Second
	}

	return &EventConsumer{
		broker:          broker,
		config:          config,
		eventRegistry:   eventRegistry,
		handlerRegistry: handlerRegistry,
		middlewares:     []Middleware{},
	}
}

// Use добавляет middleware ко всем обработчикам
func (c *EventConsumer) Use(middlewares ...Middleware) {
	c.middlewares = append(c.middlewares, middlewares...)
}

// Start запускает консьюмера
func (c *EventConsumer) Start() error {
	log.Printf("[EventConsumer] Starting with bindings: %v", c.config.Bindings)

	// Создаём обработчик, который будет передан брокеру
	handler := func(delivery amqp.Delivery) error {
		return c.handleDelivery(delivery)
	}

	// Запускаем потребление через брокера
	// Брокер сам управляет:
	//   - очередями и bindings
	//   - retry логикой (через handleError)
	//   - DLQ
	//   - graceful shutdown
	err := c.broker.Consume(handler, c.config.Bindings...)
	if err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	log.Printf("[EventConsumer] ✅ Started")
	return nil
}

// handleDelivery - обработка одного сообщения
// Эта функция вызывается из rabbitmq.messageProcessor
//
// ВАЖНО:
//   - Возвращает nil → брокер сделает msg.Ack()
//   - Возвращает error → брокер вызовет handleError() (retry/DLQ)
func (c *EventConsumer) handleDelivery(delivery amqp.Delivery) error {
	c.received.Add(1)

	// Создаём контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), c.config.MaxProcessingTime)
	defer cancel()

	// 1. Парсим базовое событие, чтобы узнать тип
	var baseEvent events.BaseEvent
	if err := json.Unmarshal(delivery.Body, &baseEvent); err != nil {
		// Невалидный JSON — нет смысла повторять
		// Возвращаем nil, чтобы брокер сделал Ack (сообщение удаляется)
		log.Printf("[EventConsumer] Invalid JSON, ack and discard: %v", err)
		return nil
	}

	// Получаем счётчик retry из заголовков (для логов)
	retryCount := getRetryCount(delivery)
	log.Printf("[EventConsumer] Received %s (id=%s, retry=%d)",
		baseEvent.EventType, baseEvent.EventID, retryCount)

	// 2. Десериализуем в конкретное событие
	event, err := c.eventRegistry.UnmarshalPayload(baseEvent.EventType, delivery.Body)
	if err != nil {
		// Ошибка десериализации — нет смысла повторять
		log.Printf("[EventConsumer] Unmarshal failed for %s: %v", baseEvent.EventType, err)
		return nil
	}

	// 3. Находим обработчик
	handler, found := c.handlerRegistry.Get(baseEvent.EventType)
	if !found {
		// Нет обработчика — нет смысла повторять
		log.Printf("[EventConsumer] No handler for %s", baseEvent.EventType)
		return nil
	}

	// 4. Применяем middleware (собираем цепочку)
	finalHandler := handler
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		finalHandler = c.middlewares[i](finalHandler)
	}

	// 5. Вызываем бизнес-логику
	//    Если здесь ошибка → брокер сделает retry или отправит в DLQ
	err = finalHandler.Handle(ctx, event)
	if err != nil {
		c.failed.Add(1)
		log.Printf("[EventConsumer] Handler error for %s: %v", baseEvent.EventType, err)
		return err // ← возвращаем ошибку брокеру
	}

	c.processed.Add(1)
	log.Printf("[EventConsumer] ✅ Processed %s (id=%s)", baseEvent.EventType, baseEvent.EventID)

	return nil // успех — брокер сделает Ack
}

// getRetryCount извлекает счётчик попыток из заголовков
func getRetryCount(delivery amqp.Delivery) int {
	if delivery.Headers == nil {
		return 0
	}

	val, ok := delivery.Headers["x-retries"]
	if !ok {
		return 0
	}

	switch v := val.(type) {
	case int32:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

// Stop - остановка консьюмера
func (c *EventConsumer) Stop() error {
	log.Printf("[EventConsumer] Stopping...")
	// Остановка брокера происходит отдельно
	// Consumer просто перестаёт получать новые сообщения
	return nil
}

// Stats - статистика
func (c *EventConsumer) Stats() (received, processed, failed int64) {
	return c.received.Load(), c.processed.Load(), c.failed.Load()
}

// IsHealthy - проверка здоровья
func (c *EventConsumer) IsHealthy() bool {
	processed := c.processed.Load()
	failed := c.failed.Load()

	if processed+failed == 0 {
		return true
	}

	failureRate := float64(failed) / float64(processed+failed)
	return failureRate < 0.3
}
