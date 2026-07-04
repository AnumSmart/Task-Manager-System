package deps

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"telegram-bot/internal/config"
	grpcuserclient "telegram-bot/internal/grpc_clients/grpc_user_client"
)

// Container - DI контейнер
type Container struct {
	// ==================== КОНФИГУРАЦИЯ ====================
	config *config.AppConfig
	// ====================== ЛОГГЕР =======================
	logger *slog.Logger
	// ==================== РЕСУРСЫ (Closeable) ====================

	// ==================== AUTH LAYER ====================

	// ==================== СЕРВИСЫ (БИЗНЕС-ЛОГИКА) ====================

	// ==================== ХЕНДЛЕРЫ ====================

	// ==================== Клиент (GRPC) ======================
	userGrpcClient grpcuserclient.FullUserGrpcService

	// ==================== УПРАВЛЕНИЕ РЕСУРСАМИ ====================
	closers   []func() error // closers - список функций для закрытия ресурсов. Каждый closer вызывается только один раз
	closeOnce sync.Once      // closeOnce - гарантирует однократное закрытие ресурсов
	closeErr  error          // closeErr - ошибка, возникшая при закрытии ресурсов
}

func NewContainer(ctx context.Context, cfg *config.AppConfig) (*Container, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic inside DI container constructor: %v\n", r)
		}
	}()

	c := &Container{
		config:  cfg,
		closers: make([]func() error, 0),
	}

	// инициализируем логгер
	if err := c.initLogger(ctx); err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	log := c.logger
	log.Info("starting DI container initialization")

	// инициализация grpc клиента для сервиса user-service
	if err := c.initGrpcUserClient(ctx); err != nil {
		c.Close()
		return nil, fmt.Errorf("Failde to innit grpc user client: %w", err)
	}

	log.Info("✅ DI container initialized successfully with single Kafka client")
	return c, nil
}
