package main

import (
	"fmt"
	"log"
	"time"
	"user-service/internal/server"
)

// startGRPCServer запускает gRPC сервер в горутине
func startGRPCServer(grpcServer *server.GRPCUserServer, serverErrors chan<- error) {
	log.Println("🚀 Starting gRPC server...")

	// Небольшая задержка перед запуском (опционально)
	if ServerStartDelay > 0 {
		time.Sleep(ServerStartDelay)
	}

	// Запускаем сервер в горутине, чтобы не блокировать main
	go func() {
		log.Printf("✓ gRPC server listening on port %s", grpcServer.GetPort())
		log.Println("========================================")
		log.Println("Server is ready to accept requests")
		log.Println("========================================")

		// Run блокирует выполнение, пока сервер не остановится или не произойдет ошибка
		if err := grpcServer.Run(); err != nil {
			serverErrors <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()
}
