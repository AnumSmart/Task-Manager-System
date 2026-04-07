package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"user-service/internal/config"
	"user-service/internal/deps"
)

// createDIContainer создает DI контейнер со всеми зависимостями
func createDIContainer(cfg *config.UserServiceConfig) (*deps.Container, error) {
	log.Println("🔧 Creating DI container...")

	// Создаем контекст с таймаутом для инициализации
	initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Создаем контейнер (инициализирует БД, Redis, репозитории, сервисы, хендлеры)
	container, err := deps.NewContainer(initCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	log.Println("  ✓ Database connection pool created")
	log.Println("  ✓ Redis connection created")
	log.Println("  ✓ Repositories initialized")
	log.Println("  ✓ Services initialized")
	log.Println("  ✓ Handlers initialized")

	return container, nil
}
