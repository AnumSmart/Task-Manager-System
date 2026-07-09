package main

import (
	"context"
	"fmt"
	"log"
	"telegram-bot/internal/deps"
)

// healthCheck проверяет здоровье всех зависимостей.
func healthCheck(container *deps.Container) error {
	log.Println("🏥 Running health checks...")

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	// Проверяем все зависимости через метод контейнера
	err := container.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}
