package main

import (
	"context"
	"fmt"
	"log"
	"telegram-bot/internal/config"
	"telegram-bot/internal/deps"
	"time"
)

// createDIContainer создает DI контейнер со всеми зависимостями.
func createDIContainer(cfg *config.AppConfig) (*deps.Container, error) {
	log.Println("🔧 Creating DI container...")

	// Создаем контекст с таймаутом для инициализации
	initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Создаем контейнер (инициализирует БД, Redis, репозитории, сервисы, хендлеры)
	container, err := deps.NewContainer(initCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	log.Println("  ✓ GRPC Client for user-service initialized")
	log.Println("  ✓ Bot HTTP Handler created")
	log.Println("  ✓ HTTP BotGateway created")

	return container, nil
}
