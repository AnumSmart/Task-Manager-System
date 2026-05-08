package rabbitmq

import (
	"log"
	"math/rand"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// handleError обрабатывает ошибку при обработке сообщения.
// Реализует логику повторных попыток (retry) и отправки в DLQ.
//
// Улучшения по сравнению с оригиналом:
//  1. Асинхронное планирование retry (без блокировки time.Sleep)
//  2. Поддержка контекста для graceful shutdown
//  3. Экспоненциальная задержка
//  4. Защита от потери сообщений при закрытии
//  5. Jitter для предотвращения "эффекта стада"

func (b *Broker) handleError(msg amqp.Delivery, handlerErr error) {
	// Получаем счётчик попыток из заголовков
	retries := b.getRetryCount(msg)

	log.Printf("[RabbitMQ] Message processing failed (attempt %d/%d): %v",
		retries+1, b.config.MaxRetries, handlerErr)

	// Проверяем, не превышен ли лимит попыток
	if retries >= b.config.MaxRetries {
		b.handleMaxRetriesExceeded(msg, handlerErr)
		return
	}

	// Увеличиваем счётчик и планируем повторную обработку
	b.scheduleRetry(msg, handlerErr, retries+1)
}

// handleMaxRetriesExceeded отправляет сообщение в DLQ или отклоняет
func (b *Broker) handleMaxRetriesExceeded(msg amqp.Delivery, handlerErr error) {
	log.Printf("[RabbitMQ] Max retries (%d) exceeded for message, sending to DLQ: %v",
		b.config.MaxRetries, handlerErr)

	// Отправляем в Dead Letter Queue
	if err := b.sendToDLQ(msg, handlerErr); err != nil {
		log.Printf("[RabbitMQ] Failed to send to DLQ: %v", err)
		// В случае ошибки просто nack без requeue (теряем сообщение)
		// чтобы избежать бесконечного цикла
		msg.Nack(false, false)
	} else {
		// Успешно отправили в DLQ - подтверждаем исходное сообщение
		msg.Ack(false)
	}
}

// sendToDLQ отправляет сообщение в Dead Letter Queue (улучшенная версия)
func (b *Broker) sendToDLQ(msg amqp.Delivery, handlerErr error) error {
	if b.config.DLQName == "" {
		log.Printf("[RabbitMQ] DLQ not configured, discarding message")
		return nil
	}

	// Формируем заголовки с информацией об ошибке
	headers := make(amqp.Table)
	if msg.Headers != nil {
		for k, v := range msg.Headers {
			headers[k] = v
		}
	}

	headers["x-dlq-reason"] = handlerErr.Error()
	headers["x-dlq-timestamp"] = time.Now().UTC()
	headers["x-dlq-original-routing-key"] = msg.RoutingKey
	headers["x-dlq-failed-at"] = time.Now().UTC().Format(time.RFC3339)

	// Публикуем в DLQ с отдельным routing key для DLQ
	// Обычно DLQ — это отдельный exchange или очередь
	dlqRoutingKey := b.config.DLQName

	return b.Publish(dlqRoutingKey, msg.Body, headers)
}

// scheduleRetry планирует повторную обработку сообщения с задержкой
//
// Ключевое улучшение: НЕ БЛОКИРУЕТ текущую горутину!
// Запускает отдельную горутину с таймером, которая отправит сообщение заново
func (b *Broker) scheduleRetry(msg amqp.Delivery, handlerErr error, nextRetryCount int) {
	// Увеличиваем счётчик активных горутин для graceful shutdown
	b.closeWg.Add(1)

	// Запускаем асинхронную задачу
	go func() {
		defer b.closeWg.Done()

		// Рассчитываем задержку (экспоненциальная + jitter)
		delay := b.calculateRetryDelay(nextRetryCount)

		log.Printf("[RabbitMQ] Scheduling retry #%d in %v", nextRetryCount, delay)

		// Ожидаем с возможностью прерывания при закрытии
		select {
		case <-time.After(delay):
			// Задержка прошла - отправляем на повторную обработку

		case <-b.closeCh:
			// Брокер закрывается - не отправляем повторно
			log.Printf("[RabbitMQ] Broker closing, discarding retry for message")
			// Отклоняем исходное сообщение без возврата в очередь
			msg.Nack(false, false)
			return
		}

		// Проверяем, что брокер всё ещё работает
		if b.getState() == StateClosing {
			log.Printf("[RabbitMQ] Broker is closing, discarding retry")
			msg.Nack(false, false)
			return
		}

		// Подготавливаем заголовки для повторной попытки
		retryHeaders := b.prepareRetryHeaders(msg, handlerErr, nextRetryCount)

		// Публикуем сообщение снова
		err := b.Publish(
			msg.RoutingKey,
			msg.Body,
			retryHeaders,
		)

		if err != nil {
			log.Printf("❌ [RabbitMQ] Failed to republish message for retry #%d: %v",
				nextRetryCount, err)
			// Если не удалось переопубликовать, возвращаем в очередь через Nack
			// но только если брокер ещё жив
			if b.getState() == StateConnected {
				msg.Nack(false, true)
			} else {
				msg.Nack(false, false)
			}
			return
		}

		// Успешно переопубликовали - подтверждаем исходное сообщение
		msg.Ack(false)
		log.Printf("[RabbitMQ] Retry #%d scheduled successfully", nextRetryCount)
	}()
}

// calculateRetryDelay рассчитывает задержку перед следующей попыткой
// Использует экспоненциальную задержку с jitter для предотвращения "эффекта стада"
func (b *Broker) calculateRetryDelay(retryCount int) time.Duration {
	// Базовая задержка из конфига (например, 1 секунда)
	baseDelay := b.config.RetryDelay
	if baseDelay <= 0 {
		baseDelay = 1 * time.Second
	}

	// Экспоненциальная задержка: baseDelay * 2^(retryCount-1)
	// retryCount=1 → 1 сек
	// retryCount=2 → 2 сек
	// retryCount=3 → 4 сек
	// retryCount=4 → 8 сек
	// retryCount=5 → 16 сек
	exponentialDelay := baseDelay * time.Duration(1<<uint(retryCount-1))

	// Ограничиваем максимальную задержку (например, 5 минут)
	maxDelay := 5 * time.Minute
	if exponentialDelay > maxDelay {
		exponentialDelay = maxDelay
	}

	// Добавляем jitter (случайное отклонение ±20%)
	// Это предотвращает ситуацию, когда 1000 сервисов одновременно пытаются
	// переподключиться после сбоя ("эффект стада")
	jitter := time.Duration(float64(exponentialDelay) * (0.8 + 0.4*rand.Float64()))

	return jitter
}

// prepareRetryHeaders подготавливает заголовки для повторной попытки
func (b *Broker) prepareRetryHeaders(msg amqp.Delivery, handlerErr error, retryCount int) amqp.Table {
	// Копируем существующие заголовки
	headers := make(amqp.Table)
	if msg.Headers != nil {
		for k, v := range msg.Headers {
			headers[k] = v
		}
	}

	// Добавляем/обновляем информацию о повторных попытках
	headers["x-retries"] = int32(retryCount)
	headers["x-last-error"] = handlerErr.Error()
	headers["x-last-retry-time"] = time.Now().UTC().Unix()

	// Опционально: сохраняем историю ошибок
	if errors, ok := headers["x-error-history"]; ok {
		// Если уже есть история, добавляем новую ошибку
		if history, ok := errors.([]string); ok {
			headers["x-error-history"] = append(history, handlerErr.Error())
		}
	} else {
		headers["x-error-history"] = []string{handlerErr.Error()}
	}

	return headers
}

// ============================================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ============================================================================

// getRetryCount извлекает счётчик попыток из заголовков сообщения
func (b *Broker) getRetryCount(msg amqp.Delivery) int {
	if msg.Headers == nil {
		return 0
	}

	val, ok := msg.Headers["x-retries"]
	if !ok {
		return 0
	}

	// Безопасное приведение типов
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
