package main

import (
	"context"
	"fmt"
	"log"
	"user-service/internal/deps"
)

// healthCheck проверяет здоровье всех зависимостей
func healthCheck(container *deps.Container) error {
	log.Println("🏥 Running health checks...")

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	// Проверяем все зависимости через метод контейнера
	if err := container.HealthCheck(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	log.Println("  ✓ PostgreSQL: responsive")
	log.Println("  ✓ Redis: responsive")

	return nil
}
