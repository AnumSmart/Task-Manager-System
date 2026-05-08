package rabbitmq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publish публикует сообщение в RabbitMQ.
//
// Параметры:
//   - routingKey: ключ маршрутизации (зависит от типа exchange)
//   - body: тело сообщения (обычно JSON)
//   - headers: дополнительные заголовки (например, для retry)
//
// Почему метод не возвращает ошибку при включённом confirm mode:
//   - Подтверждения приходят асинхронно (см. WaitForConfirmation)
//   - Синхронное ожидание каждого подтверждения сильно снизит производительность
//   - Для гарантии доставки используйте PublishWithConfirm
func (b *Broker) Publish(routingKey string, body []byte, headers amqp.Table) error {
	return b.PublishWithContext(context.Background(), routingKey, body, headers)
}

// PublishWithContext публикует сообщение с поддержкой контекста.
//
// Поддержка контекста позволяет:
//   - Отменить публикацию при превышении timeout'а
//   - Передать trace-id для распределённой трассировки
func (b *Broker) PublishWithContext(ctx context.Context, routingKey string, body []byte, headers amqp.Table) error {
	// Проверяем состояние брокера
	if b.getState() != StateConnected {
		return ErrNotConnected
	}

	b.connMu.RLock()
	ch := b.channel
	b.connMu.RUnlock()

	if ch == nil {
		return ErrNotConnected
	}

	// Если routingKey не указан, используем из конфига
	if routingKey == "" {
		routingKey = b.config.RoutingKey
	}

	// Публикуем сообщение
	//
	// Почему publishing содержит DeliveryMode=Persistent:
	//   - Сообщение будет сохранено на диск (если очередь durable)
	//   - При перезапуске RabbitMQ сообщение не потеряется
	//   - Persistent сообщения немного медленнее, но надёжнее
	err := ch.PublishWithContext(
		ctx,
		b.config.ExchangeName,
		routingKey,
		false, // mandatory (если true - вернёт сообщение, если нет очередей)
		false, // immediate (deprecated, не используем)
		amqp.Publishing{
			ContentType:  "application/json",
			Headers:      headers,
			Body:         body,
			DeliveryMode: amqp.Persistent, // сохранять на диск
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}

	return nil
}

// PublishWithConfirm публикует сообщение и синхронно ждёт подтверждения.
// Используйте для критически важных сообщений, где недопустима потеря.
//
// Почему это медленнее:
//   - Делает round-trip к RabbitMQ и обратно
//   - Блокирует горутину до получения подтверждения
//   - Рекомендуется только для важных событий (< 1% всех сообщений)
func (b *Broker) PublishWithConfirm(ctx context.Context, routingKey string, body []byte, headers amqp.Table) error {
	// Confirm mode должен быть включён
	if !b.config.EnableConfirmMode {
		return ErrNoConfirmMode
	}

	// Публикуем сообщение
	if err := b.PublishWithContext(ctx, routingKey, body, headers); err != nil {
		return err
	}

	// Ждём подтверждение
	return b.WaitForConfirmation(ctx)
}

// WaitForConfirmation ожидает подтверждение от RabbitMQ (после публикации).
func (b *Broker) WaitForConfirmation(ctx context.Context) error {
	b.publishMu.RLock()
	confirmsCh := b.confirms
	b.publishMu.RUnlock()

	if confirmsCh == nil {
		return ErrNoConfirmMode
	}

	select {
	case confirm := <-confirmsCh:
		if !confirm.Ack {
			return fmt.Errorf("message not confirmed (nack)")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
