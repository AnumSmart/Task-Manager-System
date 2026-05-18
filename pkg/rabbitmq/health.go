package rabbitmq

import (
	"errors"
	"log"
	"time"
)

// ============================================================================
// МЕТОДЫ УПРАВЛЕНИЯ (STATE, HEALTH, CLOSE)
// ============================================================================

// getState возвращает текущее состояние соединения (атомарно)
func (b *Broker) getState() ConnectionState {
	return ConnectionState(b.state.Load())
}

// setState устанавливает новое состояние (атомарно)
func (b *Broker) setState(state ConnectionState) {
	b.state.Store(int32(state))
}

// IsConnected проверяет, активно ли соединение с RabbitMQ.
func (b *Broker) IsConnected() bool {
	return b.getState() == StateConnected
}

// HealthCheck возвращает статус здоровья брокера.
// Используется для health check эндпоинтов.
func (b *Broker) HealthCheck() error {
	if b.getState() != StateConnected {
		return errors.New("❌ rabbitmq not connected")
	}

	b.connMu.RLock()
	defer b.connMu.RUnlock()

	if b.conn == nil || b.conn.IsClosed() {
		return errors.New("❌ rabbitmq connection is closed")
	}

	return nil
}

// Close gracefully закрывает брокер и все соединения.
//
// Почему graceful:
//  1. Перестаём принимать новые сообщения
//  2. Дожидаемся завершения обработки текущих (closeWg)
//  3. Закрываем канал confirms
//  4. Закрываем канал и соединение
func (b *Broker) Close() error {
	var closeErr error

	b.closeOnce.Do(func() {
		log.Printf("[RabbitMQ] Closing broker...")
		b.setState(StateClosing)

		// Сигналим всем горутинам о завершении
		close(b.closeCh)

		// Ждём завершения всех горутин (максимум 30 секунд)
		done := make(chan struct{})
		go func() {
			b.closeWg.Wait()
			close(done)
		}()

		select {
		case <-done:
			log.Printf("[RabbitMQ] All goroutines finished")
		case <-time.After(30 * time.Second):
			log.Printf("[RabbitMQ] Timeout waiting for goroutines to finish")
		}

		// Закрываем канал и соединение
		b.connMu.Lock()
		defer b.connMu.Unlock()

		if b.channel != nil {
			if err := b.channel.Close(); err != nil {
				log.Printf("❌ [RabbitMQ] Error closing channel: %v", err)
				closeErr = err
			}
		}

		if b.conn != nil {
			if err := b.conn.Close(); err != nil {
				log.Printf("❌ [RabbitMQ] Error closing connection: %v", err)
				closeErr = err
			}
		}

		log.Printf("[RabbitMQ] Broker closed")
	})

	return closeErr
}
