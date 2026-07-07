package deps

import (
	"context"
	"errors"
	"fmt"
	"pkg/logger"
	"slices"
	grpcuserclient "telegram-bot/internal/grpc_clients/grpc_user_client"
	"telegram-bot/internal/server"
	"telegram-bot/internal/server/handlers"
)

// addCloser - добавляет функцию закрытия ресурса.
func (c *Container) addCloser(closer func() error) {
	c.closers = append(c.closers, closer)
}

// initLogger - создаёт единый логгер.
func (c *Container) initLogger(ctx context.Context) error {
	// Инициализируем глобальный синглтон для этого микросервиса
	logger.InitLogger(c.config.LoggerConfig)

	// Получаем экземпляр логгера
	c.logger = logger.GetLogger()

	// Проверяем, что логгер инициализирован
	if c.logger == nil {
		return errors.New("failed to initialize logger")
	}

	c.logger.Info("logger initialized",
		"level", c.config.LoggerConfig.Level,
		"format", c.config.LoggerConfig.Format,
		"service", c.config.LoggerConfig.Service,
	)

	return nil
}

// метод инициализации grpc клиента для user-service.
func (c *Container) initGrpcUserClient(ctx context.Context) error {
	c.logger.Info("initializing grpc client for user-service")

	// создаём grpc client на базе конфига
	userGRPCClient, err := grpcuserclient.NewClient(c.config.GrpcClient)
	if err != nil {
		return err
	}

	// если все успешно, то переопределяем контейнер
	c.userGrpcClient = userGRPCClient

	// Добавляем в closers
	c.addCloser(func() error {
		c.userGrpcClient.Close()
		c.logger.Info("grpc user client resources cleaned up")

		return nil
	})

	c.logger.Info("✅ GRPC User Client initialized")

	return nil
}

// метод инициализации http bot handler
func (c *Container) initBotHandler(ctx context.Context) error {
	c.logger.Info("initializing http bot handler")

	botHttpHandler, err := handlers.NewBotHttpHandler(c.botGateway.GetBot())
	if err != nil {
		return err
	}

	// если все успешно, то переопределяем контейнер
	c.botHandler = botHttpHandler

	c.logger.Info("✅ Bot HTTP Handler initialized")
	return nil
}

// метод для инициализации http сервера (botGateway)
func (c *Container) initBotGateway(ctx context.Context) error {
	c.logger.Info("initializing http botGateway (server)")

	botGateway, err := server.NewBotGateway(ctx, c.config.BotServer, c.config.Bot, c.botHandler)
	if err != nil {
		return err
	}
	// если все успешно, то переопределяем контейнер
	c.botGateway = botGateway

	// Добавляем в closers
	c.addCloser(func() error {
		err := c.botGateway.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("botGateway shutdown - failed:%w", err)
		}
		c.logger.Info("botGateway resources cleaned up")
		return nil
	})

	c.logger.Info("✅ HTTP BotGateway initialized")
	return nil
}

// Close - закрытие всех ресурсов.
func (c *Container) Close() error {
	c.closeOnce.Do(func() {
		c.logger.Info("🛑 Closing DI container resources...")

		var errs []error

		// Закрываем ресурсы в обратном порядке
		for _, v := range slices.Backward(c.closers) {
			err := v()
			if err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			c.closeErr = fmt.Errorf("close errors: %v", errs)
		} else {
			c.logger.Info("✅ Container resources closed successfully")
		}
	})

	return c.closeErr
}

// HealthCheck проверяет здоровье зависимостей.
func (c *Container) HealthCheck(ctx context.Context) error {
	// TODO
	return nil
}

// Геттер для получения экземпляра сервера из контейнера
func (c *Container) GetBotGateway() *server.BotGateway {
	return c.botGateway
}
