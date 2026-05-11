package deps

import (
	"context"
	"fmt"
	"global_models/global_cache"
	"global_models/global_db"
	"log"
	"pkg/auth"
	postgresdb "pkg/db"
	"pkg/rabbitmq"
	"pkg/redis"
	"sync"
	"user-service/internal/config"
	"user-service/internal/server"
	"user-service/internal/server/handler"
	"user-service/internal/server/repository"
	"user-service/internal/server/service"
)

// Container - DI контейнер (приватная структура)
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

	// ==================== Сервер (GRPC) ====================
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

	// 3. Инициализация репозиториев
	if err := c.initRepositories(); err != nil {
		c.Close()
		return nil, fmt.Errorf("init repositories: %w", err)
	}

	// 4. Инициализация сервисов
	if err := c.initServices(); err != nil {
		c.Close()
		return nil, fmt.Errorf("init services: %w", err)
	}

	// 5. Инициализация хендлеров
	if err := c.initHandlers(); err != nil {
		c.Close()
		return nil, fmt.Errorf("init handlers: %w", err)
	}

	// 6. Инициализация gRPC сервера
	if err := c.initGRPCServer(); err != nil {
		c.Close()
		return nil, fmt.Errorf("init grpc server: %w", err)
	}

	log.Println("DI container initialized successfully")
	return c, nil
}

// Управление ресурсами (добавляем функцию закрытия в слайс)
func (c *Container) addCloser(closer func() error) {
	c.closers = append(c.closers, closer)
}

// Close закрывает ТОЛЬКО ресурсы (БД, Redis и т.д.)
// Сервер не закрывается здесь!
func (c *Container) Close() error {
	c.closeOnce.Do(func() {
		log.Println("closing container resources (DB, Redis, etc)...")

		var errs []error

		// Закрываем в обратном порядке
		for i := len(c.closers) - 1; i >= 0; i-- {
			if err := c.closers[i](); err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			c.closeErr = fmt.Errorf("close errors: %v", errs)
		} else {
			log.Println("container resources closed successfully")
		}
	})

	return c.closeErr
}

// внутренний метод иницализации ресурсов
func (c *Container) initResources(ctx context.Context) error {
	// PostgreSQL
	pgPool, err := postgresdb.NewPoolWithConfig(ctx, c.config.PostgresDBConfig)
	if err != nil {
		return fmt.Errorf("create postgres pool: %w", err)
	}
	// если пул соединений был успешно создан, инициализируем его в структуре контейнера
	c.pgPool = pgPool

	// регистрируем функцию освобождения ресурсов
	c.addCloser(func() error {
		if err := c.pgPool.Close(); err != nil {
			return fmt.Errorf("postgres close: %w", err)
		}
		log.Println("PostgreSQL connection closed")
		return nil
	})

	// Redis (cache)
	redisCache, err := redis.NewRedisCacheRepository(ctx, c.config.RedisConf)
	if err != nil {
		return fmt.Errorf("failed to create redis cache: %w", err)
	}
	// если редис успешно создан, инициализируем его в структуре контейнера
	c.redisCache = redisCache

	// регистрируем функцию освобождения ресурсов
	c.addCloser(func() error {
		if err := c.redisCache.Close(); err != nil {
			return fmt.Errorf("redis close: %w", err)
		}
		log.Println("Redis Cache connection closed")
		return nil
	})

	// Redis (blacklist)
	redisBlackList, err := redis.NewRedisCacheRepository(ctx, c.config.RedisBlackListConf)
	if err != nil {
		return fmt.Errorf("failed to create redis blacklist: %w", err)
	}
	// если редис успешно создан, инициализируем его в структуре контейнера
	c.redisBlackList = redisBlackList

	// регистрируем функцию освобождения ресурсов
	c.addCloser(func() error {
		if err := c.redisBlackList.Close(); err != nil {
			return fmt.Errorf("redis close: %w", err)
		}
		log.Println("Redis BlackList connection closed")
		return nil
	})

	// RabbitMQ client
	brokerClient, err := rabbitmq.New(c.config.RabbitConfig)
	if err != nil {
		return fmt.Errorf("failed to create rabbitMQ client: %w", err)
	}

	// если rabbitMQ клиент успешно создан, инициализируем его в структуре контейнера
	c.brokerClient = brokerClient

	// регистрируем функцию освобождения ресурсов
	c.addCloser(func() error {
		if err := c.brokerClient.Close(); err != nil {
			return fmt.Errorf("rabbitMQ close: %w", err)
		}
		log.Println("RabbitMQ connection closed")
		return nil
	})

	return nil
}

// внутренний метод инициализации логики авторизации
func (c *Container) initAuthService() error {
	// проверка того, что blacklist != nil
	if c.redisBlackList == nil {
		return fmt.Errorf("Redis blacklist could not be nil")
	}

	// создаём конфиг для провайдера
	providerConf := auth.ProviderConfig{
		EnableBlacklist: true,
	}

	// создаём провайдера
	provider := auth.NewProvider(c.config.JWTConfig)
	provider = provider.WithCache(c.redisBlackList)
	provider = provider.WithConfig(providerConf)

	authService, err := provider.Build()
	if err != nil {
		return fmt.Errorf("failed to create authService: %w", err)
	}

	// если authService успешно создан, инициализируем его в структуре контейнера
	c.authService = authService

	log.Println("✓ Auth service initialized")

	return nil
}

// внутренний метод инициализации репозитория
func (c *Container) initRepositories() error {
	// репозиторий для работы с postgres
	pgRepo, err := repository.NewUserServiceDBRepository(c.pgPool)
	// проверка на ошибку или на nil репозиторий
	if err != nil || pgRepo == nil {
		return fmt.Errorf("failed to create db repository")
	}

	// если все успешно, то инициализируем в структуре контейнера
	c.dbRepo = pgRepo

	// репозиторий для работы с кэшом (на базе Redis)
	cacheRepo, err := repository.NewUserServiceCacheRepo(c.redisCache, "UserService")
	// проверка на ошибку или на nil репозиторий
	if err != nil || cacheRepo == nil {
		return fmt.Errorf("failed to create cache repository")
	}

	// если все успешно, то инициализируем в структуре контейнера
	c.cacheRepo = cacheRepo

	// комбинированный репозиторий
	repo, err := repository.NewUserServiceRepository(c.dbRepo, c.cacheRepo)
	// проверка на ошибку или на nil репозиторий
	if err != nil || repo == nil {
		return fmt.Errorf("failed to create user repository")
	}
	// если все успешно, то инициализируем в структуре контейнера
	c.repo = repo

	return nil
}

// initServices инициализирует сервисы с внедрением auth
func (c *Container) initServices() error {
	// Передаем auth сервис в UserService
	userService := service.NewUserService(c.repo, c.authService, c.brokerClient)
	if userService == nil {
		return fmt.Errorf("failed to create user service")
	}

	c.userService = userService
	log.Println("✓ User service initialized with auth")

	return nil
}

// внутренний метод инициализации хэндлеров
func (c *Container) initHandlers() error {
	userHandler := handler.NewUserServerHandler(c.userService)
	if userHandler == nil {
		return fmt.Errorf("failed to create user handler")
	}

	// если все успешно - инициализируем зависимость в контейнере
	c.userHandler = userHandler

	return nil
}

// initGRPCServer инициализирует gRPC сервер с JWT interceptors
func (c *Container) initGRPCServer() error {
	grpcConfig := c.config.GRPCServerConfig

	// Создаем сервер с JWT interceptor
	// Для этого нужно обновить server.NewGRCPUserServer, чтобы он принимал interceptors
	c.grpcServer = server.NewGRCPUserServer(
		grpcConfig,
		c.userHandler,
		c.authService, // Передаем auth для интерсепторов
	)

	if c.grpcServer == nil {
		return fmt.Errorf("failed to create gRPC server")
	}

	log.Println("✓ gRPC server initialized with JWT interceptor")
	return nil
}

// Геттеры для внешнего использования
func (c *Container) GetGRPCServer() *server.GRPCUserServer {
	return c.grpcServer
}

func (c *Container) GetUserHandler() *handler.UserServerHandler {
	return c.userHandler
}

// GetAuthService возвращает auth сервис (если нужен в других местах)
func (c *Container) GetAuthService() auth.AuthInterface {
	return c.authService
}

// HealthCheck проверяет здоровье зависимостей
func (c *Container) HealthCheck(ctx context.Context) error {
	// Проверка PostgreSQL
	if _, err := c.pgPool.Exec(ctx, "SELECT 1"); err != nil {
		return fmt.Errorf("postgres health check failed: %w", err)
	}

	// Проверка Redis
	if err := c.redisCache.Set(ctx, "health_check", []byte("ok"), 1); err != nil {
		return fmt.Errorf("redis cache health check failed: %w", err)
	}

	// Проверка Auth сервиса (Redis blacklist)
	if c.authService != nil {
		// Простая проверка через валидацию тестового токена или ping Redis
		if err := c.authService.HealthCheck(ctx); err != nil {
			return err
		}
	}

	// Проверка RabbitMQ клиента
	// проверяем, что указатель не nil
	if c.brokerClient != nil {
		if err := c.brokerClient.HealthCheck(); err != nil {
			return fmt.Errorf("rabbitMQ health check failed: %w", err)
		}
	}

	return nil
}
