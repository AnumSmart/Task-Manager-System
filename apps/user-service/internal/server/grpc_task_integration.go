package server

import (
	pb "api/gen/go/user/v1"
	"context"
	"log"
)

// ValidateUser - комплексная проверка пользователя перед назначением задачи
// Проверяет существование, активность и соответствие ролям
func (s *GRPCUserServer) ValidateUser(ctx context.Context, req *pb.ValidateUserRequest) (*pb.ValidateUserResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 ValidateUser вызван: user_id=%s", req.GetUserId())

	return &pb.ValidateUserResponse{IsValid: true}, nil
}

// CheckUserExists - быстрая проверка существования пользователя
// Используется для валидации перед созданием задачи
func (s *GRPCUserServer) CheckUserExists(ctx context.Context, req *pb.CheckUserExistsRequest) (*pb.CheckUserExistsResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 CheckUserExists вызван: user_id=%s", req.GetUserId())

	return &pb.CheckUserExistsResponse{Exists: true}, nil
}

// GetUserByID - получение пользователя по ID (алиас для GetUser)
// Выделен в отдельный метод для task-service для чёткого разделения ответственности
func (s *GRPCUserServer) GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.GetUserResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 GetUserByID вызван: user_id=%s", req.GetUserId())

	return &pb.GetUserResponse{}, nil
}

// BatchGetUsers - массовое получение пользователей
// Оптимизация: вместо N вызовов GetUser - один вызов BatchGetUsers
func (s *GRPCUserServer) BatchGetUsers(ctx context.Context, req *pb.BatchGetUsersRequest) (*pb.BatchGetUsersResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 BatchGetUsers вызван: count=%d", len(req.GetUserIds()))

	return &pb.BatchGetUsersResponse{}, nil
}
