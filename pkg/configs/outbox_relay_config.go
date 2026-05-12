package configs

import (
	"fmt"
	"time"
)

// OutboxRelayConfig - конфигурация outbox реле
// Загружается из YAML файла в каждом сервисе индивидуально
type OutboxRelayConfig struct {
	PollInterval   time.Duration // PollInterval - интервал опроса базы данных для поиска новых PENDING событий
	BatchSize      int           // BatchSize - максимальное количество событий, вычитываемых за один раз
	PublishTimeout time.Duration // PublishTimeout - таймаут на публикацию одного события в RabbitMQ.
}

// DefaultRelayConfig возвращает конфигурацию по умолчанию
func DefaultRelayConfig() *OutboxRelayConfig {
	return &OutboxRelayConfig{
		PollInterval:   1 * time.Second,
		BatchSize:      100,
		PublishTimeout: 5 * time.Second,
	}
}

// Validate проверяет корректность конфигурации и возвращает ошибку,
// если какие-то значения выходят за допустимые пределы.
// Вызывать при старте сервиса, после загрузки конфигурации.
func (c *OutboxRelayConfig) Validate() error {
	if c.PollInterval < 100*time.Millisecond {
		return fmt.Errorf("PollInterval too small: %v (minimum 100ms)", c.PollInterval)
	}
	if c.PollInterval > 30*time.Second {
		return fmt.Errorf("PollInterval too large: %v (maximum 30s)", c.PollInterval)
	}

	if c.BatchSize < 1 {
		return fmt.Errorf("BatchSize must be at least 1, got %d", c.BatchSize)
	}
	if c.BatchSize > 1000 {
		return fmt.Errorf("BatchSize too large: %d (maximum 1000)", c.BatchSize)
	}

	if c.PublishTimeout < 1*time.Second {
		return fmt.Errorf("PublishTimeout too small: %v (minimum 1s)", c.PublishTimeout)
	}
	if c.PublishTimeout > 30*time.Second {
		return fmt.Errorf("PublishTimeout too large: %v (maximum 30s)", c.PublishTimeout)
	}

	return nil
}
