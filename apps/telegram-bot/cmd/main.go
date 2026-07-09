package main

import (
	"log"
	"telegram-bot/internal/config"
	"time"
)

// Настройки graceful shutdown.
const (
	// GracefulShutdownTimeout - максимальное время ожидания завершения текущих запросов.
	GracefulShutdownTimeout = 30 * time.Second

	// ServerStartDelay - задержка перед запуском сервера (для отладки).
	ServerStartDelay = 0 * time.Second

	// HealthCheckTimeout - таймаут для проверки здоровья зависимостей.
	HealthCheckTimeout = 5 * time.Second

	envPath = "c:\\Son_Alex\\GO_projects\\task_management_system\\apps\\telegram-bot\\.env"
)

func main() {
	// Создаем логгер с timestamp
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.Println("========================================")
	log.Println("Starting Telegram-bot Service")
	log.Println("========================================")

	// 1. Загрузка конфигурации
	cfg, err := config.LoadAppConfig(envPath)
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

	// 5. Получение http сервера из контейнера
	botHttpServer := container.GetBotGateway()
	if botHttpServer == nil {
		log.Fatal("❌ http botGateway is nil")
	}

	log.Println("✓ HTTP BotGateway created")

	// 6. Запуск http сервера в отдельной горутине
	serverErrors := make(chan error, 1)
	startHTTPServer(botHttpServer, serverErrors)

	// 7. Ожидание сигнала завершения или ошибки
	waitForShutdown(botHttpServer, serverErrors)

	log.Println("========================================")
	log.Println("Tekegram-bot Service stopped successfully")
	log.Println("========================================")

}
