package handler

import (
	pb "api/gen/go/user/v1"
	"context"
	"log"
)

// GetAllUsers - получение всех пользователей (без пагинации)
// Используется аналитикой для построения полных отчётов
func (s *UserServerHandler) GetAllUsers(ctx context.Context, req *pb.GetAllUsersRequest) (*pb.GetAllUsersResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 GetAllUsers вызван")

	return &pb.GetAllUsersResponse{}, nil
}

// GetUsersByRole - получение пользователей с определённой ролью
// Например: "показать всех MANAGER'ов для назначения"
func (s *UserServerHandler) GetUsersByRole(ctx context.Context, req *pb.GetUsersByRoleRequest) (*pb.GetUsersByRoleResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 GetUsersByRole вызван: role=%v", req.GetRole())

	return &pb.GetUsersByRoleResponse{}, nil
}

// GetUserRole - быстрый метод получения только роли пользователя
// Легче, чем полный GetUser, когда нужна только роль
func (s *UserServerHandler) GetUserRole(ctx context.Context, req *pb.GetUserRoleRequest) (*pb.GetUserRoleResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 GetUserRole вызван: user_id=%s", req.GetUserId())

	return &pb.GetUserRoleResponse{}, nil
}
