package handler

import (
	pb "api/gen/go/user/v1"
	"context"
	"log"
)

// HealthCheck - проверка состояния user-service
func (s *UserServerHandler) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 HealthCheck вызван: telegram_id=%s", req.GetRequestId())

	return &pb.HealthCheckResponse{}, nil
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
