package handler

import (
	pb "api/gen/go/user/v1"
	"context"
	"log"
)

// CreateUser - создание нового пользователя в организации
// Только пользователи с ролью OWNER или MANAGER могут создавать новых пользователей
func (s *UserServerHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 CreateUser вызван: email=%s, role=%v", req.GetEmail(), req.GetRole())

	return &pb.CreateUserResponse{}, nil
}

// GetUser - получение информации о пользователе по ID
// Доступна всем авторизованным пользователям (но разные роли видят разный набор полей)
func (s *UserServerHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 GetUser вызван: user_id=%s", req.GetUserId())

	return &pb.GetUserResponse{}, nil
}

// UpdateUser - обновление данных пользователя
// OWNER может менять любые поля, MANAGER - только некоторые, EMPLOYEE - только свой профиль
func (s *UserServerHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 UpdateUser вызван: user_id=%s", req.GetUserId())

	return &pb.UpdateUserResponse{}, nil
}

// DeleteUser - удаление или деактивация пользователя
// Только OWNER может удалять пользователей (soft delete рекомендуется)
func (s *UserServerHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 DeleteUser вызван: user_id=%s", req.GetUserId())

	return &pb.DeleteUserResponse{}, nil
}

// ListUsers - получение списка пользователей с фильтрацией и пагинацией
// Доступно для OWNER и MANAGER, EMPLOYEE видит только себя
func (s *UserServerHandler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	return &pb.ListUsersResponse{}, nil
}
