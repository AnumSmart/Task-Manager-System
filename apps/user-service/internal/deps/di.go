package deps

import (
	"context"
	"fmt"
	"global_models/global_cache"
	"global_models/global_db"
	"log"
	"pkg/auth"
	"pkg/events"
	"pkg/outbox"
	"pkg/rabbitmq"
	"sync"
	"user-service/internal/config"
	"user-service/internal/server"
	"user-service/internal/server/handler"
	"user-service/internal/server/repository"
	"user-service/internal/server/service"
)

// Container - DI контейнер
type Container struct {
	// ==================== КОНФИГУРАЦИЯ ====================
	config *config.UserServiceConfig // config - конфигурация сервиса

	// ==================== РЕСУРСЫ (Closeable) ====================
	pgPool         global_db.Pool     // pgPool - пул соединений с PostgreSQL (интерфейс)
	redisCache     global_cache.Cache // redisCache - клиент для работы с Redis (интерфейс)
	redisBlackList global_cache.Cache // клиент для работы с Redis (интерфейс) - черный список для JWT
	brokerClient   *rabbitmq.Broker   // клиент для работы с RabbitMQ

	// ==================== AUTH LAYER ====================
	authService auth.AuthInterface // НОВЫЙ: сервис авторизации (JWT)

	// ==================== РЕПОЗИТОРИИ (СЛОИ ДОСТУПА К ДАННЫМ) ====================
	dbRepo    *repository.UserServiceDBRepository    // dbRepo - репозиторий для работы с базой данных (PostgreSQL)
	cacheRepo *repository.UserServiceCacheRepository // cacheRepo - репозиторий для работы с кэшем (Redis)
	repo      *repository.UserServiceRepository      // repo - КОМПОЗИТНЫЙ репозиторий (основной для сервисов)

	// ==================== СЕРВИСЫ (БИЗНЕС-ЛОГИКА) ====================
	userService *service.UserService // userService - сервис пользователей

	// ==================== ХЕНДЛЕРЫ (GRPC) ====================
	userHandler *handler.UserServerHandler // userHandler - gRPC хендлер для работы с пользователями

	// ==================== OUTBOX компоненты ==================
	eventPublisher *events.EventPublisher           // публикатор событий, которые будут отправлены в брокер
	outboxRepo     *outbox.PostgresOutboxRepository // репоизторий (интерфейс) для работы с outbox таблицей
	outboxRegistry *outbox.EventRegistry            // реестр событий
	outboxRelay    *outbox.Relay                    // логика асинхронной публикации событий согласно outbox

	// ==================== Сервер (GRPC) ======================
	grpcServer *server.GRPCUserServer // grpc сервер

	// ==================== УПРАВЛЕНИЕ РЕСУРСАМИ ====================
	closers   []func() error // closers - список функций для закрытия ресурсов. Каждый closer вызывается только один раз
	closeOnce sync.Once      // closeOnce - гарантирует однократное закрытие ресурсов
	closeErr  error          // closeErr - ошибка, возникшая при закрытии ресурсов
}

// NewContainer создает контейнер
func NewContainer(ctx context.Context, cfg *config.UserServiceConfig) (*Container, error) {
	// пытаемся отловить панику
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic inside DI container constructor: %v\n", r)
		}
	}()

	// создаём начальный экземпляр контейнера, чтобы для его наполнения вызывать инициализацию зависимостей
	c := &Container{
		config:  cfg,
		closers: make([]func() error, 0),
	}

	// 1. Инициализация ресурсов
	if err := c.initResources(ctx); err != nil {
		return nil, fmt.Errorf("init resources: %w", err)
	}

	// 2. Инициализация Auth сервиса
	if err := c.initAuthService(); err != nil {
		c.Close()
		return nil, fmt.Errorf("init auth service: %w", err)
	}

	// 3. Инициализация outbox (регистрация событий)
	if err := c.initOutbox(); err != nil {
		c.Close()
		return nil, fmt.Errorf("init outbox: %w", err)
	}

	// 4. Репозитории (БД + Cache, Outbox)
	if err := c.initRepositories(); err != nil {
		c.Close()
		return nil, fmt.Errorf("init repositories: %w", err)
	}

	// 5. Инициализация EventPublisher
	if err := c.initEventPublisher(); err != nil {
		c.Close()
		return nil, fmt.Errorf("init event publisher: %w", err)
	}

	// 6. Инициализация сервисов (с передачей eventPublisher)
	if err := c.initServices(); err != nil {
		c.Close()
		return nil, fmt.Errorf("init services: %w", err)
	}

	// 7. Запуск Outbox Relay
	if err := c.startOutboxRelay(ctx); err != nil {
		c.Close()
		return nil, fmt.Errorf("start outbox relay: %w", err)
	}

	// 8. Инициализация хендлеров
	if err := c.initHandlers(); err != nil {
		c.Close()
		return nil, fmt.Errorf("init handlers: %w", err)
	}

	// 9. Инициализация gRPC сервера
	if err := c.initGRPCServer(); err != nil {
		c.Close()
		return nil, fmt.Errorf("init grpc server: %w", err)
	}

	log.Println("DI container initialized successfully")
	return c, nil
}
