package main

import (
	"fmt"
	"log"
	"telegram-bot/internal/server"
	"time"
)

func startHTTPServer(server *server.BotGateway, serverErrors chan<- error) {
	log.Println("🚀 Starting HTTP server (botGateway)...")

	// Небольшая задержка перед запуском (опционально)
	if ServerStartDelay > 0 {
		time.Sleep(ServerStartDelay)
	}

	// Запускаем сервер в горутине, чтобы не блокировать main
	go func() {
		log.Printf("✓ HTTP server listening on port %s", server.GetPort())
		log.Println("========================================")
		log.Println("HTTP Server is ready to accept requests")
		log.Println("========================================")

		// Run блокирует выполнение, пока сервер не остановится или не произойдет ошибка
		err := server.Run()
		if err != nil {
			serverErrors <- fmt.Errorf("HTTP Server (BotGateway) server error: %w", err)
		}
	}()
}
