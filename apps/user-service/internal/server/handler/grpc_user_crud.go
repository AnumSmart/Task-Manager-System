package handler

import (
	pb "api/gen/go/user/v1"
	"context"
	"errors"
	"log"
	"user-service/internal/converter"
	"user-service/internal/domain"
	"user-service/internal/server/interceptors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateUser - создание нового пользователя в организации
// Только пользователи с ролью OWNER или MANAGER могут создавать новых пользователей
func (s *UserServerHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	// 1. Проверка контекста
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 2. Извлечение данных из контекста (через интерсептор)
	createdBy, ok := ctx.Value(interceptors.ContextKeyUserID).(string)
	if !ok || createdBy == "" {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	organizationID, ok := ctx.Value(interceptors.ContextKeyOrganizationID).(string)
	if !ok || organizationID == "" {
		return nil, status.Error(codes.Unauthenticated, "organization not found")
	}

	role, ok := ctx.Value(interceptors.ContextKeyRole).(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "role not found")
	}

	// 3. Проверка прав на хэндлер-уровне
	if role != "OWNER" && role != "MANAGER" {
		return nil, status.Error(codes.PermissionDenied, "only OWNER or MANAGER can create users")
	}

	// 4. Конвертация с передачей organizationID
	userToCreate := converter.ToDomainUserFromCreateRequest(req, organizationID)
	if userToCreate == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user data")
	}

	// создаём доменный запрос
	domainRequest := domain.CreateUserRequest{
		OrganizationID: userToCreate.OrganizationID,
		Email:          userToCreate.Email,
		FullName:       userToCreate.FullName,
		Role:           userToCreate.Role,
	}

	// Вызываем сервис, передавая createdBy отдельно
	user, err := s.UserServerService.User.CreateUser(ctx, &domainRequest, createdBy)
	if err != nil {
		if errors.Is(err, domain.ErrPermissionDenied) {
			return nil, status.Error(codes.PermissionDenied, "недостаточно прав")
		}
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "пользователь уже существует")
		}
		return nil, status.Error(codes.Internal, "ошибка создания пользователя")
	}

	log.Printf("📝 CreateUser вызван: email=%s, role=%v", req.GetEmail(), req.GetRole())

	return &pb.CreateUserResponse{
		Success: true,
		User:    converter.ToProtoUser(user),
	}, nil
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

// Logout - - выход из системы (отзыв JWT токена)
func (s *UserServerHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	return &pb.LogoutResponse{}, nil
}
