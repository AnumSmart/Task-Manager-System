package handler

import (
	pb "api/gen/go/user/v1"
	"context"
	"fmt"
	"log"
	"time"
)

// HealthCheck - проверка состояния user-service
func (s *UserServerHandler) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	// Проверка graceful shutdown
	if s.IsShuttingDown() {
		return &pb.HealthCheckResponse{
			Success: true,
			Status:  "NOT_SERVING",
			Details: map[string]string{
				"status": "shutting_down",
			},
		}, nil
	}

	details := make(map[string]string)
	allHealthy := true

	// 1. Проверка PostgreSQL
	// создаём отдельный контекст
	ctxDB, cancelDB := context.WithTimeout(ctx, 2*time.Second)
	defer cancelDB()

	if err := s.UserServerService.HealthCheck.HealthCheckDB(ctxDB); err != nil {
		allHealthy = false
		details["database"] = fmt.Sprintf("error: %v", err)
		log.Printf("⚠️ HealthCheck: DB не отвечает - %v", err)
	} else {
		details["database"] = "ok"
	}

	// 2. Проверка Redis Cache
	// создаём отдельный контекст
	ctxCache, cancelCache := context.WithTimeout(ctx, 2*time.Second)
	defer cancelCache()

	if err := s.UserServerService.HealthCheck.HealthCheckCache(ctxCache); err != nil {
		allHealthy = false
		details["redis_cache"] = fmt.Sprintf("error: %v", err)
		log.Printf("⚠️ HealthCheck: Cache не отвечает - %v", err)
	} else {
		details["redis_cache"] = "ok"
	}

	// 3. Проверка Auth Service (Redis Black List)
	ctxAuth, cancelAuth := context.WithTimeout(ctx, 2*time.Second)
	defer cancelAuth()

	if err := s.UserServerService.HealthCheck.HealthCheckBlackList(ctxAuth); err != nil {
		allHealthy = false
		details["auth_blacklist"] = fmt.Sprintf("error: %v", err)
		log.Printf("⚠️ HealthCheck: Auth Service (blacklist) не отвечает - %v", err)
	} else {
		details["auth_blacklist"] = "ok"
	}

	// Определяем общий статус
	status := "SERVING"
	if !allHealthy {
		status = "NOT_SERVING"
	}

	// Логируем только если сервис нездоров
	if !allHealthy {
		log.Printf("❌ HealthCheck: сервис нездоров, статус=%s, details=%v", status, details)
	}

	return &pb.HealthCheckResponse{
		Success: true,
		Status:  status,
		Details: details,
	}, nil
}

// ValidateToken - проверка JWT токена (для сервисов без JWT_SECRET)
func (s *UserServerHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 ValidateToken вызван: telegram_id=%s", req.GetRequestId())

	return &pb.ValidateTokenResponse{}, nil
}
