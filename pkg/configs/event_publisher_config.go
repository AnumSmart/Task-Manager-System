package configs

import "time"

// ============================================================================
// КОНФИГУРАЦИЯ PUBLISHER
// ============================================================================

// PublisherConfig - конфигурация EventPublisher.
type EventPublisherConfig struct {
	// WorkerCount - количество воркеров (горутин), которые обрабатывают события.
	// Каждый воркер в бесконечном цикле читает события из канала и публикует их.
	// Рекомендуемые значения:
	//   - 1-2: для low-load (разработка)
	//   - 5-10: для production (стандарт)
	//   - 20-50: для high-load (тысячи событий/сек)
	WorkerCount int

	// QueueSize - размер буфера канала для событий.
	// Если канал заполнен, новые события будут отклонены (с логированием).
	// Это механизм backpressure - защита от перегрузки.
	// Рекомендуемые значения:
	//   - 100: для разработки
	//   - 1000-5000: для production
	QueueSize int

	// RetryCount - количество повторных попыток при ошибке публикации.
	// При ошибке делаем RetryCount попыток, затем событие считается потерянным.
	// Рекомендуемые значения: 3-5
	RetryCount int

	// RetryBackoff - начальная задержка между повторными попытками.
	// Задержка увеличивается экспоненциально: backoff, backoff*2, backoff*4...
	// Рекомендуемое значение: 100ms
	// RetryBackoffMs - начальная задержка в миллисекундах
	RetryBackoffMs int `yaml:"retry_backoff_ms"`

	// PublishTimeout - таймаут на одну попытку публикации.
	// Если публикация не укладывается в таймаут, считаем её неудачной.
	// Рекомендуемое значение: 5-10 секунд
	// PublishTimeoutSec - таймаут публикации в секундах
	PublishTimeoutSec int `yaml:"publish_timeout_sec"`

	// EnableDLQ - сохранять ли события, которые не удалось опубликовать после всех retry.
	// Включение помогает не терять важные события для последующего анализа.
	// Рекомендуемое значение: true для production
	EnableDLQ bool
}

// RetryBackoff возвращает time.Duration из RetryBackoffMs
func (c *EventPublisherConfig) RetryBackoff() time.Duration {
	return time.Duration(c.RetryBackoffMs) * time.Millisecond
}

// PublishTimeout возвращает time.Duration из PublishTimeoutSec
func (c *EventPublisherConfig) PublishTimeout() time.Duration {
	return time.Duration(c.PublishTimeoutSec) * time.Second
}

// DefaultEventPublisherConfig - значения по умолчанию (если не указаны в YAML)
func DefaultEventPublisherConfig() *EventPublisherConfig {
	return &EventPublisherConfig{
		WorkerCount:       5,
		QueueSize:         1000,
		RetryCount:        3,
		RetryBackoffMs:    100,
		PublishTimeoutSec: 5,
		EnableDLQ:         true,
	}
}

// Validate проверяет корректность конфигурации
func (c *EventPublisherConfig) Validate() error {
	if c.WorkerCount <= 0 {
		c.WorkerCount = 5
	}
	if c.QueueSize <= 0 {
		c.QueueSize = 1000
	}
	if c.RetryCount < 0 {
		c.RetryCount = 3
	}
	if c.RetryBackoffMs <= 0 {
		c.RetryBackoffMs = 100
	}
	if c.PublishTimeoutSec <= 0 {
		c.PublishTimeoutSec = 5
	}
	return nil
}
