package main

import (
	"log"
	"time"
	"user-service/internal/config"
)

// Настройки graceful shutdown
const (
	// GracefulShutdownTimeout - максимальное время ожидания завершения текущих запросов
	GracefulShutdownTimeout = 30 * time.Second

	// ServerStartDelay - задержка перед запуском сервера (для отладки)
	ServerStartDelay = 0 * time.Second

	// HealthCheckTimeout - таймаут для проверки здоровья зависимостей
	HealthCheckTimeout = 5 * time.Second
)

func main() {
	// Создаем логгер с timestamp
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.Println("========================================")
	log.Println("Starting User Service")
	log.Println("========================================")

	// 1. Загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}
	log.Println("✓ Configuration loaded successfully")

	// 2. Создание DI контейнера
	container, err := createDIContainer(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to create DI container: %v", err)
	}

	// 3. Настройка graceful shutdown (отложенное закрытие ресурсов)
	defer gracefulShutdown(container)

	// 4. Проверка здоровья зависимостей
	if err := healthCheck(container); err != nil {
		log.Fatalf("❌ Health check failed: %v", err)
	}
	log.Println("✓ All dependencies healthy")

	// 5. Получение gRPC сервера из контейнера
	grpcServer := container.GetGRPCServer()
	if grpcServer == nil {
		log.Fatal("❌ gRPC server is nil")
	}
	log.Println("✓ gRPC server created")

	// 6. Запуск gRPC сервера в отдельной горутине
	serverErrors := make(chan error, 1)
	startGRPCServer(grpcServer, serverErrors)

	// 7. Ожидание сигнала завершения или ошибки
	waitForShutdown(grpcServer, serverErrors)

	log.Println("========================================")
	log.Println("User Service stopped successfully")
	log.Println("========================================")
}
