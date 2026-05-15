package configs

import "time"

// ConsumerConfig - конфигурация консьюмера
type ConsumerConfig struct {
	// Bindings - список routing keys для подписки
	Bindings []string

	// MaxProcessingTime - максимальное время обработки одного сообщения
	// Если превышен, обработка прерывается и возвращается ошибка (брокер сделает retry)
	MaxProcessingTime time.Duration
}

// DefaultConsumerConfig возвращает конфигурацию по умолчанию
func DefaultConsumerConfig(bindings ...string) *ConsumerConfig {
	return &ConsumerConfig{
		Bindings:          bindings,
		MaxProcessingTime: 30 * time.Second,
	}
}
