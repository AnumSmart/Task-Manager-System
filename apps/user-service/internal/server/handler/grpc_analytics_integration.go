package handler

import (
	types "api/gen/go/common/v1"
	pb "api/gen/go/user/v1"
	"context"
	"log"
	"user-service/internal/converter"
	"user-service/internal/domain"
)

// GetAllUsers - получение всех пользователей (без пагинации)
// Используется аналитикой для построения полных отчётов
// 🔒 API Key авторизация
func (s *UserServerHandler) GetAllUsers(ctx context.Context, req *pb.GetAllUsersRequest) (*pb.GetAllUsersResponse, error) {
	// 1. Проверка контекста
	select {
	case <-ctx.Done():
		log.Printf("❌ GetAllUsers: контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 GetAllUsers вызван: include_inactive=%v, limit=%d, offset=%d",
		req.IncludeInactive, req.Limit, req.Offset)

	// 2. Валидация входных параметров
	if err := s.validateGetAllUsersRequest(req); err != nil {
		log.Printf("❌ GetAllUsers: ошибка валидации: %v", err)
		return &pb.GetAllUsersResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Нормализация параметров
	limit, offset := normalizeGetAllUsersParams(int(req.Limit), int(req.Offset))

	// Вызов сервисного слоя
	users, total, err := s.UserServerService.Analytics.GetAllUsers(ctx, req.IncludeInactive, limit, offset)
	if err != nil {
		log.Printf("❌ GetAllUsers: ошибка сервиса: %v", err)
		return &pb.GetAllUsersResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// ✅ Используем конвертер
	pbUsers := converter.ToProtoUsers(users)

	log.Printf("✅ GetAllUsers: возвращено %d из %d пользователей", len(pbUsers), total)

	return &pb.GetAllUsersResponse{
		Success: true,
		Users:   pbUsers,
		Total:   int32(total),
	}, nil
}

// GetUsersByRole - получение пользователей с определённой ролью
// Используется task-service для показа всех MANAGER'ов при назначении
// 🔒 API Key авторизация
func (s *UserServerHandler) GetUsersByRole(ctx context.Context, req *pb.GetUsersByRoleRequest) (*pb.GetUsersByRoleResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ GetUsersByRole: контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 GetUsersByRole: role=%v, include_inactive=%v", req.GetRole(), req.GetIncludeInactive())

	// 1. Валидация входных параметров
	if err := validateGetUsersByRoleRequest(req); err != nil {
		log.Printf("❌ GetUsersByRole: ошибка валидации: %v", err)
		return &pb.GetUsersByRoleResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// 2. Конвертация protobuf роли в доменную
	domainRole := converter.ToDomainRole(req.GetRole())

	// 3. Вызов сервисного слоя
	users, err := s.UserServerService.Analytics.GetUsersByRole(ctx, domainRole, req.GetIncludeInactive())
	if err != nil {
		log.Printf("❌ GetUsersByRole: ошибка сервиса: %v", err)
		return &pb.GetUsersByRoleResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// 4. Конвертация domain → protobuf с помощью конвертера
	pbUsers := converter.ToProtoUsers(users)

	log.Printf("✅ GetUsersByRole: найдено %d пользователей с ролью %v", len(pbUsers), req.GetRole())

	return &pb.GetUsersByRoleResponse{
		Success: true,
		Users:   pbUsers,
		Total:   int32(len(pbUsers)),
	}, nil
}

// GetUserRole - быстрый метод получения только роли пользователя
// Используется analytics-service и другими сервисами для проверки прав
// 🔒 API Key авторизация
func (s *UserServerHandler) GetUserRole(ctx context.Context, req *pb.GetUserRoleRequest) (*pb.GetUserRoleResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ GetUserRole: контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 GetUserRole: user_id=%s", req.GetUserId())

	// 1. Валидация входных параметров
	if err := validateGetUserRoleRequest(req); err != nil {
		log.Printf("❌ GetUserRole: ошибка валидации: %v", err)
		return &pb.GetUserRoleResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// 2. Вызов сервисного слоя
	role, err := s.UserServerService.Analytics.GetUserRole(ctx, req.GetUserId())
	if err != nil {
		log.Printf("❌ GetUserRole: ошибка сервиса: %v", err)
		return &pb.GetUserRoleResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// 3. Конвертация доменной роли в protobuf
	pbRole := converter.ToProtoRole(role)

	log.Printf("✅ GetUserRole: пользователь %s имеет роль %v", req.GetUserId(), pbRole)

	return &pb.GetUserRoleResponse{
		Success: true,
		Role:    pbRole,
	}, nil
}

// validateGetAllUsersRequest - валидация параметров запроса
func (h *UserServerHandler) validateGetAllUsersRequest(req *pb.GetAllUsersRequest) error {
	// limit: максимум 1000, минимум 1 (если указан)
	if req.Limit < 0 {
		return domain.ErrInvalidInputMess("limit не может быть отрицательным")
	}
	if req.Limit > 1000 {
		return domain.ErrInvalidInputMess("limit не может превышать 1000")
	}

	// offset: не может быть отрицательным
	if req.Offset < 0 {
		return domain.ErrInvalidInputMess("offset не может быть отрицательным")
	}

	return nil
}

// normalizeGetAllUsersParams - нормализация лимитов
func normalizeGetAllUsersParams(limit, offset int) (int, int) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// validateGetUsersByRoleRequest - валидация параметров запроса
func validateGetUsersByRoleRequest(req *pb.GetUsersByRoleRequest) error {
	if req.GetRole() == types.Role_ROLE_UNSPECIFIED {
		return domain.ErrInvalidInputMess("role не может быть UNSPECIFIED")
	}
	return nil
}

// validateGetUserRoleRequest - валидация параметров запроса
func validateGetUserRoleRequest(req *pb.GetUserRoleRequest) error {
	if req.GetUserId() == "" {
		return domain.ErrInvalidInputMess("user_id не может быть пустым")
	}
	return nil
}
