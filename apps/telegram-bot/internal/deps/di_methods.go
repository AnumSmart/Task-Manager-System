package deps

import (
	"context"
	"fmt"
	"pkg/logger"
	grpcuserclient "telegram-bot/internal/grpc_clients/grpc_user_client"
)

// addCloser - добавляет функцию закрытия ресурса
func (c *Container) addCloser(closer func() error) {
	c.closers = append(c.closers, closer)
}

// initLogger - создаёт единый логгер
func (c *Container) initLogger(ctx context.Context) error {
	// Инициализируем глобальный синглтон для этого микросервиса
	logger.InitLogger(c.config.LoggerConfig)

	// Получаем экземпляр логгера
	c.logger = logger.GetLogger()

	// Проверяем, что логгер инициализирован
	if c.logger == nil {
		return fmt.Errorf("failed to initialize logger")
	}

	c.logger.Info("logger initialized",
		"level", c.config.LoggerConfig.Level,
		"format", c.config.LoggerConfig.Format,
		"service", c.config.LoggerConfig.Service,
	)

	return nil
}

// метож инициализации grpc клиента для user-service
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

// Close - закрытие всех ресурсов
func (c *Container) Close() error {
	c.closeOnce.Do(func() {
		c.logger.Info("🛑 Closing DI container resources...")

		var errs []error

		// Закрываем ресурсы в обратном порядке
		for i := len(c.closers) - 1; i >= 0; i-- {
			if err := c.closers[i](); err != nil {
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
