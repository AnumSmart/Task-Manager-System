package service

import (
	"context"
	"fmt"
	"log"
	"user-service/internal/domain"
	"user-service/internal/server/repository"
)

// конструктор для части сервисного слоя, который отвечает за работу с аналитикой
type AnalyticsLayer struct {
	AnRepo repository.AnalyticsDBRepository
}

// констркутор для части сервисного слоя (аналитика)
// в конструктор передаём составной репозиторий (на будущее)
func NewAnalyticsLayer(repo *repository.UserServiceRepository) *AnalyticsLayer {
	return &AnalyticsLayer{
		AnRepo: repo.DBRepo,
	}
}

// получаем список всех пользователей для аналитики
func (s *AnalyticsLayer) GetAllUsers(ctx context.Context, inactiveFlag bool, limit, offset int) ([]*domain.User, int, error) {
	log.Printf("📊 [Service] GetAllUsers: inactiveFlag=%v, limit=%d, offset=%d", inactiveFlag, limit, offset)

	// 1. Валидация параметров
	if err := validateGetAllUsersParams(limit, offset); err != nil {
		log.Printf("❌ [Service] GetAllUsers: ошибка валидации: %v", err)
		return nil, 0, err
	}

	// 2. Проверка контекста
	select {
	case <-ctx.Done():
		log.Printf("❌ [Service] GetAllUsers: контекст отменён: %v", ctx.Err())
		return nil, 0, ctx.Err()
	default:
	}

	// 3. Вызов репозитория
	users, total, err := s.AnRepo.GetAllUsers(ctx, inactiveFlag, limit, offset)
	if err != nil {
		log.Printf("❌ [Service] GetAllUsers: ошибка репозитория: %v", err)
		return nil, 0, fmt.Errorf("failed to get users from repository: %w", err)
	}

	log.Printf("✅ [Service] GetAllUsers: получено %d пользователей (всего: %d)",
		len(users), total)

	return users, total, nil
}

// получаем список пользователей по ролям для аналитики
func (s *AnalyticsLayer) GetUsersByRole(ctx context.Context, role domain.Role, includeInactive bool) ([]*domain.User, error) {
	log.Printf("📊 [Service] GetUsersByRole: role=%s, includeInactive=%v", role, includeInactive)

	// 1. Валидация
	if err := validateGetUsersByRoleParams(role); err != nil {
		log.Printf("❌ [Service] GetUsersByRole: ошибка валидации: %v", err)
		return nil, err
	}

	// 2. Проверка контекста
	select {
	case <-ctx.Done():
		log.Printf("❌ [Service] GetUsersByRole: контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	// 3. Вызов репозитория
	users, err := s.AnRepo.GetUsersByRole(ctx, role, includeInactive)
	if err != nil {
		log.Printf("❌ [Service] GetUsersByRole: ошибка репозитория: %v", err)
		return nil, fmt.Errorf("failed to get users by role: %w", err)
	}

	log.Printf("✅ [Service] GetUsersByRole: получено %d пользователей с ролью %s", len(users), role)

	return users, nil
}

// получаем роль пользовател по его ID
func (s *AnalyticsLayer) GetUserRole(ctx context.Context, userID string) (domain.Role, error) {
	log.Printf("📊 [Service] GetUserRole: user_id=%s", userID)

	// 1. Валидация
	if userID == "" {
		return domain.RoleEmployee, domain.ErrInvalidInputMess("user_id не может быть пустым")
	}

	// 2. Проверка контекста
	select {
	case <-ctx.Done():
		log.Printf("❌ [Service] GetUserRole: контекст отменён: %v", ctx.Err())
		return domain.RoleEmployee, ctx.Err()
	default:
	}

	// 3. Вызов репозитория (только роль, не весь пользователь)
	role, err := s.AnRepo.GetUserRole(ctx, userID)
	if err != nil {
		log.Printf("❌ [Service] GetUserRole: ошибка репозитория: %v", err)
		return domain.RoleEmployee, fmt.Errorf("failed to get user role: %w", err)
	}

	log.Printf("✅ [Service] GetUserRole: пользователь %s имеет роль %s", userID, role)

	return role, nil
}

// validateGetAllUsersParams - валидация параметров на сервисном уровне
func validateGetAllUsersParams(limit, offset int) error {
	if limit < 1 && limit != 0 { // limit=0 допустимо (будут дефолтные значения)
		return domain.ErrInvalidInputMess(fmt.Sprintf("limit должен быть >= 0, получено: %d", limit))
	}
	if limit > 1000 {
		return domain.ErrInvalidInputMess(fmt.Sprintf("limit не может превышать 1000, получено: %d", limit))
	}
	if offset < 0 {
		return domain.ErrInvalidInputMess(fmt.Sprintf("offset не может быть отрицательным, получено: %d", offset))
	}
	return nil
}

// validateGetUsersByRoleParams - валидация параметров
func validateGetUsersByRoleParams(role domain.Role) error {
	if role != domain.RoleOwner && role != domain.RoleManager && role != domain.RoleEmployee {
		return domain.ErrInvalidInputMess(fmt.Sprintf("некорректная роль: %s", role))
	}
	return nil
}
