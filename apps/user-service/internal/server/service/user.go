package service

import (
	"context"
	"fmt"
	"log"
	"user-service/internal/domain"
	"user-service/internal/server/repository"
)

// UserLayer - структура сервисного слоя, которая отвечает за работу с пользователями
type UserLayer struct {
	UserRepo repository.UserDBRepository
	Cache    repository.UserCacheRepository
}

// NewUserLayer - конструктор для части сервисного слоя (пользователи)
func NewUserLayer(repo *repository.UserServiceRepository) *UserLayer {
	return &UserLayer{
		UserRepo: repo.DBRepo,
		Cache:    repo.CacheRepo,
	}
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ДЛЯ КЭШИРОВАНИЯ ====================

// getCacheKey - генерация ключа для кэша
func (s *UserLayer) getCacheKey(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

// cacheUser - сохранение пользователя в кэш
func (s *UserLayer) cacheUser(ctx context.Context, user *domain.User) error {
	key := s.getCacheKey(user.ID)
	// TTL 1 час для пользователей
	if err := s.Cache.Set(ctx, key, user, 3600); err != nil {
		log.Printf("⚠️ Failed to cache user %s: %v", user.ID, err)
		return err
	}
	log.Printf("✅ User cached: %s", user.ID)
	return nil
}

// getCachedUser - получение пользователя из кэша
func (s *UserLayer) getCachedUser(ctx context.Context, userID string) (*domain.User, error) {
	key := s.getCacheKey(userID)
	var user domain.User

	if err := s.Cache.Get(ctx, key, &user); err != nil {
		return nil, err
	}

	log.Printf("📦 User retrieved from cache: %s", userID)
	return &user, nil
}

// invalidateUserCache - инвалидация кэша пользователя
func (s *UserLayer) invalidateUserCache(ctx context.Context, userID string) error {
	key := s.getCacheKey(userID)
	if err := s.Cache.Delete(ctx, key); err != nil {
		log.Printf("⚠️ Failed to invalidate cache for user %s: %v", userID, err)
		return err
	}
	log.Printf("🗑️ User cache invalidated: %s", userID)
	return nil
}

// ==================== ОСНОВНЫЕ CRUD ОПЕРАЦИИ ====================

// CreateUser - создание нового пользователя
func (s *UserLayer) CreateUser(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, error) {
	log.Printf("📝 Creating user: email=%s, org=%s", req.Email, req.OrganizationID)

	// 1. Проверяем права создающего пользователя
	requester, err := s.UserRepo.GetByID(ctx, req.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("requester not found: %w", err)
	}

	if !requester.CanManageUsers() {
		return nil, domain.ErrPermissionDenied
	}

	// 2. Проверяем, не существует ли пользователь с таким email
	existing, _ := s.UserRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	// 3. Создаем пользователя через доменную модель
	user := domain.NewUser(req.Email, req.FullName, req.Role, req.OrganizationID)

	// 4. Сохраняем в БД
	if err := s.UserRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 5. Сохраняем в кэш
	s.cacheUser(ctx, user)

	log.Printf("✅ User created successfully: ID=%s", user.ID)
	return user, nil
}

// GetUserByID - получение пользователя по ID (с кэшированием)
func (s *UserLayer) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	log.Printf("📝 Getting user by ID: %s", userID)

	// 1. Пытаемся получить из кэша
	cachedUser, err := s.getCachedUser(ctx, userID)
	if err == nil {
		return cachedUser, nil
	}

	// 2. Если нет в кэше, идем в БД
	user, err := s.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	// 3. Сохраняем в кэш для следующих запросов
	s.cacheUser(ctx, user)

	return user, nil
}

// GetUserWithAccessCheck - получение пользователя с проверкой прав
func (s *UserLayer) GetUserWithAccessCheck(ctx context.Context, req *domain.GetUserRequest) (*domain.User, error) {
	log.Printf("📝 Getting user with access check: user_id=%s, requester=%s", req.UserID, req.RequesterID)

	// 1. Получаем запрашивающего пользователя
	requester, err := s.GetUserByID(ctx, req.RequesterID)
	if err != nil {
		return nil, fmt.Errorf("requester not found: %w", err)
	}

	// 2. Получаем целевого пользователя
	targetUser, err := s.GetUserByID(ctx, req.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	// 3. Проверяем права через доменную модель
	if !requester.CanViewUser(req.UserID) {
		return nil, domain.ErrPermissionDenied
	}

	return targetUser, nil
}

// UpdateUser - обновление данных пользователя
func (s *UserLayer) UpdateUser(ctx context.Context, req *domain.UpdateUserRequest) error {
	log.Printf("📝 Updating user: ID=%s by %s", req.UserID, req.RequesterID)

	// 1. Получаем запрашивающего пользователя
	requester, err := s.GetUserByID(ctx, req.RequesterID)
	if err != nil {
		return fmt.Errorf("requester not found: %w", err)
	}

	// 2. Получаем целевого пользователя
	targetUser, err := s.GetUserByID(ctx, req.UserID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	// 3. Проверяем права
	if !requester.CanUpdateUser(req.UserID, req.Updates) {
		return domain.ErrPermissionDenied
	}

	// 4. Применяем обновления
	if fullName, ok := req.Updates["full_name"].(string); ok {
		targetUser.UpdateProfile(fullName)
	}

	if email, ok := req.Updates["email"].(string); ok {
		if err := targetUser.UpdateEmail(email); err != nil {
			return err
		}
		// Проверяем уникальность нового email
		existing, _ := s.UserRepo.GetByEmail(ctx, email)
		if existing != nil && existing.ID != targetUser.ID {
			return domain.ErrUserAlreadyExists
		}
	}

	if role, ok := req.Updates["role"].(domain.Role); ok {
		switch role {
		case domain.RoleManager:
			if err := targetUser.PromoteToManager(); err != nil {
				return err
			}
		case domain.RoleEmployee:
			if err := targetUser.DemoteToEmployee(); err != nil {
				return err
			}
		}
	}

	if status, ok := req.Updates["status"].(domain.UserStatus); ok {
		switch status {
		case domain.UserStatusSuspended:
			if err := targetUser.Suspend(); err != nil {
				return err
			}
		case domain.UserStatusActive:
			if err := targetUser.Activate(); err != nil {
				return err
			}
		}
	}

	// 5. Сохраняем в БД
	if err := s.UserRepo.Update(ctx, targetUser); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// 6. Инвалидируем кэш
	s.invalidateUserCache(ctx, req.UserID)

	log.Printf("✅ User updated: ID=%s", req.UserID)
	return nil
}

// DeleteUser - удаление пользователя
func (s *UserLayer) DeleteUser(ctx context.Context, req *domain.DeleteUserRequest) error {
	log.Printf("📝 Deleting user: ID=%s by %s (hard=%v)", req.UserID, req.RequesterID, req.HardDelete)

	// 1. Получаем запрашивающего пользователя
	requester, err := s.GetUserByID(ctx, req.RequesterID)
	if err != nil {
		return fmt.Errorf("requester not found: %w", err)
	}

	// 2. Получаем целевого пользователя
	targetUser, err := s.GetUserByID(ctx, req.UserID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	// 3. Проверяем права
	if !requester.CanDeleteUser(req.UserID, targetUser.Role) {
		return domain.ErrPermissionDenied
	}

	// 4. Выполняем удаление
	if req.HardDelete {
		// Жесткое удаление
		if err := s.UserRepo.Delete(ctx, req.UserID); err != nil {
			return fmt.Errorf("failed to hard delete user: %w", err)
		}
	} else {
		// Мягкое удаление - меняем статус
		targetUser.Status = domain.UserStatusInactive
		if err := s.UserRepo.Update(ctx, targetUser); err != nil {
			return fmt.Errorf("failed to soft delete user: %w", err)
		}
	}

	// 5. Инвалидируем кэш
	s.invalidateUserCache(ctx, req.UserID)

	log.Printf("✅ User deleted: ID=%s", req.UserID)
	return nil
}

// ListUsers - получение списка пользователей с пагинацией
func (s *UserLayer) ListUsers(ctx context.Context, offset, limit int) ([]*domain.User, int, error) {
	log.Printf("📝 Listing users: offset=%d, limit=%d", offset, limit)

	// Списки обычно не кэшируем, так как они часто меняются
	users, err := s.UserRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	// Получаем общее количество
	totalCount, err := s.UserRepo.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	log.Printf("✅ Retrieved %d users (total: %d)", len(users), totalCount)
	return users, totalCount, nil
}

// ListUsersByOrganization - получение списка пользователей организации
func (s *UserLayer) ListUsersByOrganization(ctx context.Context, organizationID string, offset, limit int) ([]*domain.User, error) {
	log.Printf("📝 Listing users for organization: %s", organizationID)

	// TODO: Добавить метод GetByOrganizationID в репозиторий
	// Пока используем List и фильтруем
	users, err := s.UserRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	var orgUsers []*domain.User
	for _, user := range users {
		if user.OrganizationID == organizationID {
			orgUsers = append(orgUsers, user)
		}
	}

	log.Printf("✅ Retrieved %d users for organization %s", len(orgUsers), organizationID)
	return orgUsers, nil
}

// ==================== ОПЕРАЦИИ АУТЕНТИФИКАЦИИ ====================

// AuthenticateUser - аутентификация пользователя
func (s *UserLayer) AuthenticateUser(ctx context.Context, email, password string) (*domain.User, error) {
	log.Printf("📝 Authenticating user: email=%s", email)

	// 1. Находим пользователя по email (не кэшируем, так как редко повторяется)
	user, err := s.UserRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// 2. Проверяем пароль (упрощенно, нужно использовать bcrypt)
	// TODO: Использовать bcrypt для сравнения
	if user.PasswordHash != password {
		return nil, domain.ErrInvalidCredentials
	}

	// 3. Проверяем статус пользователя
	if !user.IsActive() {
		return nil, domain.ErrUserSuspended
	}

	// 4. Обновляем время последнего входа
	now := domain.Now()
	user.LastLoginAt = &now
	if err := s.UserRepo.Update(ctx, user); err != nil {
		log.Printf("⚠️ Failed to update last login: %v", err)
	}

	// 5. Сохраняем в кэш
	s.cacheUser(ctx, user)

	log.Printf("✅ User authenticated: ID=%s", user.ID)
	return user, nil
}

// ChangePassword - смена пароля пользователя
func (s *UserLayer) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	log.Printf("📝 Changing password: user_id=%s", userID)

	// 1. Получаем пользователя
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	// 2. Проверяем старый пароль
	// TODO: Использовать bcrypt для сравнения
	if user.PasswordHash != oldPassword {
		return domain.ErrInvalidCredentials
	}

	// 3. Обновляем пароль
	// TODO: Хешировать новый пароль через bcrypt
	user.PasswordHash = newPassword
	user.UpdatedAt = domain.Now()

	// 4. Сохраняем в БД
	if err := s.UserRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// 5. Инвалидируем кэш
	s.invalidateUserCache(ctx, userID)

	log.Printf("✅ Password changed: user_id=%s", userID)
	return nil
}

// ==================== РОЛЕВЫЕ ОПЕРАЦИИ ====================

// TransferOwnership - передача прав OWNER другому пользователю
func (s *UserLayer) TransferOwnership(ctx context.Context, currentOwnerID, newOwnerID, organizationID string) error {
	log.Printf("📝 Transferring ownership: from=%s to=%s", currentOwnerID, newOwnerID)

	// 1. Получаем текущего владельца
	currentOwner, err := s.GetUserByID(ctx, currentOwnerID)
	if err != nil {
		return fmt.Errorf("current owner not found: %w", err)
	}

	// 2. Получаем нового владельца
	newOwner, err := s.GetUserByID(ctx, newOwnerID)
	if err != nil {
		return fmt.Errorf("new owner not found: %w", err)
	}

	// 3. Проверяем, что оба в одной организации
	if currentOwner.OrganizationID != organizationID || newOwner.OrganizationID != organizationID {
		return domain.ErrInvalidInput
	}

	// 4. Проверяем, что текущий пользователь действительно OWNER
	if currentOwner.Role != domain.RoleOwner {
		return domain.ErrPermissionDenied
	}

	// 5. Меняем роли
	if err := currentOwner.DemoteToEmployee(); err != nil {
		return err
	}

	if err := newOwner.PromoteToOwner(); err != nil {
		return err
	}

	// 6. Сохраняем изменения
	if err := s.UserRepo.Update(ctx, currentOwner); err != nil {
		return err
	}

	if err := s.UserRepo.Update(ctx, newOwner); err != nil {
		return err
	}

	// 7. Инвалидируем кэш для обоих пользователей
	s.invalidateUserCache(ctx, currentOwnerID)
	s.invalidateUserCache(ctx, newOwnerID)

	log.Printf("✅ Ownership transferred: new_owner=%s", newOwnerID)
	return nil
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ====================

// GetUserByEmail - получение пользователя по email (без кэширования)
func (s *UserLayer) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.UserRepo.GetByEmail(ctx, email)
}

// CheckUserExists - проверка существования пользователя
func (s *UserLayer) CheckUserExists(ctx context.Context, userID string) (bool, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return false, nil
	}
	return user != nil, nil
}

// GetUsersByRole - получение пользователей по роли
func (s *UserLayer) GetUsersByRole(ctx context.Context, organizationID string, role domain.Role) ([]*domain.User, error) {
	// TODO: Добавить метод в репозиторий для фильтрации по роли
	users, err := s.UserRepo.List(ctx, 0, 1000)
	if err != nil {
		return nil, err
	}

	var filtered []*domain.User
	for _, user := range users {
		if user.OrganizationID == organizationID && user.Role == role {
			filtered = append(filtered, user)
		}
	}

	return filtered, nil
}

// BulkGetUsers - массовое получение пользователей (с кэшированием)
func (s *UserLayer) BulkGetUsers(ctx context.Context, userIDs []string) ([]*domain.User, error) {
	log.Printf("📝 Bulk getting users: %d IDs", len(userIDs))

	var users []*domain.User
	var missingIDs []string

	// 1. Пытаемся получить из кэша
	for _, id := range userIDs {
		user, err := s.getCachedUser(ctx, id)
		if err == nil {
			users = append(users, user)
		} else {
			missingIDs = append(missingIDs, id)
		}
	}

	// 2. Получаем недостающих из БД
	if len(missingIDs) > 0 {
		// TODO: Добавить метод GetByIDs в репозиторий
		for _, id := range missingIDs {
			user, err := s.UserRepo.GetByID(ctx, id)
			if err == nil {
				users = append(users, user)
				s.cacheUser(ctx, user) // Сохраняем в кэш
			}
		}
	}

	log.Printf("✅ Bulk get completed: retrieved %d users", len(users))
	return users, nil
}
