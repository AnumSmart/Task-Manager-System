package events

import (
	"context"
	"fmt"
	"log"
	"pkg/configs"
	"pkg/rabbitmq"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================================
// ОШИБКИ PUBLISHER
// ============================================================================

var (
	// ErrQueueFull - возвращается, когда очередь событий переполнена.
	// Это признак того, что система не справляется с нагрузкой.
	ErrQueueFull = fmt.Errorf("event queue is full, backpressure applied")
)

// ============================================================================
// EVENT PUBLISHER (ОСНОВНАЯ СТРУКТУРА)
// ============================================================================

// EventPublisher - публикатор событий с worker pool.
//
// Архитектура:
//   UserService вызывает PublishAsync() → событие попадает в канал tasks
//   Воркеры читают из tasks → публикуют в RabbitMQ с retry
//
// Преимущества:
//   1. Контролируемое количество горутин (WorkerCount)
//   2. Буферизация (QueueSize) - сглаживает пики нагрузки
//   3. Backpressure - защита от перегрузки
//   4. Централизованные метрики
//   5. Graceful shutdown - дожидается всех публикаций

type EventPublisher struct {
	broker rabbitmq.BrokerInterface      // клиент RabbitMQ
	config *configs.EventPublisherConfig // конфигурация

	// tasks - канал для событий, ожидающих публикации.
	// Worker'ы читают из этого канала.
	tasks chan Event

	// closeCh - канал для сигнала завершения работы.
	// При закрытии closeCh воркеры перестают принимать новые задачи.
	closeCh chan struct{}

	// wg - ожидание завершения всех воркеров при graceful shutdown.
	wg sync.WaitGroup

	// ========== Метрики (атомарные счётчики) ==========
	// Используем atomic.Int64 для потокобезопасного инкремента без блокировок.

	// submitted - количество отправленных событий (всего)
	submitted atomic.Int64

	// published - количество успешно опубликованных событий
	published atomic.Int64

	// failed - количество событий, не опубликованных после всех retry
	failed atomic.Int64

	// dropped - количество событий, отклонённых из-за переполнения очереди
	dropped atomic.Int64
}

// NewEventPublisher - конструктор EventPublisher.
// Создаёт пул воркеров и запускает их.
func NewEventPublisher(broker rabbitmq.BrokerInterface, config *configs.EventPublisherConfig) *EventPublisher {
	p := &EventPublisher{
		broker:  broker,
		config:  config,
		tasks:   make(chan Event, config.QueueSize),
		closeCh: make(chan struct{}),
	}

	// Запускаем указанное количество воркеров
	for i := 0; i < config.WorkerCount; i++ {
		p.wg.Add(1)    // увеличиваем счётчик для WaitGroup
		go p.worker(i) // запускаем воркера с номером (для логов)
	}

	log.Printf("[EventPublisher] ✅ Started: workers=%d, queue_size=%d, retry_count=%d",
		config.WorkerCount, config.QueueSize, config.RetryCount)

	return p
}

// ============================================================================
// ПУБЛИЧНЫЕ МЕТОДЫ
// ============================================================================

// PublishAsync - асинхронная публикация события.
// Не блокирует вызывающую горутину (за исключением кратковременной блокировки при отправке в канал).
//
// Возвращает:
//   - nil: событие принято в очередь
//   - ErrQueueFull: очередь переполнена, событие отклонено
//
// Важно: ошибка публикации в RabbitMQ НЕ возвращается здесь,
// она логируется внутри worker'а и учитывается в метриках.
func (p *EventPublisher) PublishAsync(event *BaseEvent) error {
	// Увеличиваем счётчик отправленных событий
	p.submitted.Add(1)

	// Пытаемся отправить событие в канал.
	// select с default делает отправку неблокирующей.
	select {
	case p.tasks <- event:
		// Успешно отправлено в очередь
		return nil
	default:
		// Канал заполнен - применяем backpressure
		p.dropped.Add(1)
		log.Printf("[EventPublisher] ⚠️ Queue full, event dropped: %s", event.EventType)
		return ErrQueueFull
	}
}

// PublishSync - синхронная публикация события.
// Блокирует вызов до тех пор, пока событие не будет опубликовано (или не истечёт таймаут).
//
// Используйте для КРИТИЧЕСКИ ВАЖНЫХ событий, где потеря недопустима.
// Для большинства случаев используйте PublishAsync (быстрее, не блокирует).
func (p *EventPublisher) PublishSync(ctx context.Context, event *BaseEvent) error {
	p.submitted.Add(1)
	return p.publishWithRetry(ctx, event)
}

// Shutdown - завершение работы Publisher'а.
// Дожидается завершения всех текущих публикаций, но не принимает новые.
func (p *EventPublisher) Shutdown(ctx context.Context) error {
	log.Println("[EventPublisher] Shutting down...")

	// Закрываем closeCh - сигнал воркерам остановиться
	close(p.closeCh)

	// Канал для сигнала о завершении воркеров
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	// Ждём завершения или таймаута
	select {
	case <-done:
		log.Println("[EventPublisher] ✅ Shutdown complete")
		return nil
	case <-ctx.Done():
		log.Println("[EventPublisher] ⚠️ Shutdown timeout")
		return ctx.Err()
	}
}

// Stats - возвращает статистику работы Publisher'а.
// Полезно для мониторинга и health check.
func (p *EventPublisher) Stats() (submitted, published, failed, dropped int64) {
	return p.submitted.Load(), p.published.Load(), p.failed.Load(), p.dropped.Load()
}

// IsHealthy - проверка здоровья Publisher'а.
// Возвращает false, если слишком много событий падают (failed > 10% от submitted).
func (p *EventPublisher) IsHealthy() bool {
	submitted := p.submitted.Load()
	if submitted == 0 {
		return true
	}

	failed := p.failed.Load()
	dropped := p.dropped.Load()

	// Если упало или потеряно больше 10% событий - считается нездоровым
	failureRate := float64(failed+dropped) / float64(submitted)
	return failureRate < 0.1
}

// ============================================================================
// ПРИВАТНЫЕ МЕТОДЫ (WORKER И ПУБЛИКАЦИЯ С RETRY)
// ============================================================================

// worker - воркер, обрабатывающий события из очереди.
// workerId нужен только для логирования (чтобы понять, какой воркер что сделал).
func (p *EventPublisher) worker(workerId int) {
	defer p.wg.Done() // при выходе уменьшаем счётчик WaitGroup

	log.Printf("[EventPublisher] Worker %d started", workerId)

	for {
		select {
		case event := <-p.tasks:
			// Получили событие - публикуем его
			p.publishEvent(event, workerId)

		case <-p.closeCh:
			// Получили сигнал завершения - выходим
			log.Printf("[EventPublisher] Worker %d stopping", workerId)
			return
		}
	}
}

// publishEvent - публикация одного события с retry.
// Вызывается внутри воркера.
func (p *EventPublisher) publishEvent(event Event, workerId int) {
	// Создаём контекст с таймаутом на всю операцию публикации
	ctx, cancel := context.WithTimeout(context.Background(), p.config.PublishTimeout())
	defer cancel()

	// Пытаемся опубликовать с retry
	if err := p.publishWithRetry(ctx, event); err != nil {
		// Не удалось опубликовать после всех попыток
		p.failed.Add(1)
		log.Printf("[EventPublisher] Worker %d: ❌ Failed to publish %s: %v",
			workerId, event.GetEventType(), err)

		// Если включён DLQ - сохраняем упавшее событие
		if p.config.EnableDLQ {
			p.saveToDeadLetter(event, err)
		}
		return
	}

	// Успешно опубликовано
	p.published.Add(1)
	log.Printf("[EventPublisher] Worker %d: ✅ Published %s", workerId, event.GetEventType())
}

// publishWithRetry - публикация с повторными попытками.
// Использует exponential backoff для увеличения задержки между попытками.
func (p *EventPublisher) publishWithRetry(ctx context.Context, event Event) error {
	// Сериализуем событие в JSON
	eventBytes, err := event.Marshal()
	if err != nil {
		// Ошибка сериализации - бессмысленно повторять
		return fmt.Errorf("marshal failed: %w", err)
	}

	// Начальная задержка
	backoff := p.config.RetryBackoff()

	// Пытаемся опубликовать RetryCount раз
	for attempt := 0; attempt < p.config.RetryCount; attempt++ {
		// Если это не первая попытка - ждём
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// Увеличиваем задержку для следующей попытки (exponential backoff)
				backoff *= 2
				log.Printf("[EventPublisher] Retry %d/%d for %s", attempt+1, p.config.RetryCount, event.GetEventType())
			}
		}

		// Пробуем опубликовать
		if err := p.broker.Publish(event.RoutingKey(), eventBytes, nil); err == nil {
			// Успех!
			return nil
		} else if attempt == p.config.RetryCount-1 {
			// Последняя попытка не удалась
			return fmt.Errorf("publish failed after %d attempts: %w", p.config.RetryCount, err)
		}
		// Не последняя попытка - продолжаем цикл
	}

	return fmt.Errorf("publish failed after %d attempts", p.config.RetryCount)
}

// saveToDeadLetter - сохраняет событие, которое не удалось опубликовать, в DLQ.
// В текущей реализации просто логируем. В будущем можно:
//   - сохранять в файл
//   - сохранять в отдельную очередь RabbitMQ
//   - сохранять в базу данных
func (p *EventPublisher) saveToDeadLetter(event Event, reason error) {
	// Для начала достаточно подробного лога
	log.Printf("[EventPublisher] 💀 DEAD LETTER: event=%s, id=%s, reason=%v", event.GetEventType(), event.GetEventID(), reason)
}
