package service

import (
	"context"
	"errors"
	"fmt"
	"global_models/global_db"
	"log"
	"pkg/auth"
	baseevent "pkg/events"
	"pkg/outbox"
	"time"
	"user-service/internal/domain"
	"user-service/internal/events"
	"user-service/internal/server/repository"

	"golang.org/x/crypto/bcrypt"
)

// UserServiceLayer - структура сервисного слоя, которая отвечает за работу с пользователями
type UserServiceLayer struct {
	UserRepo    repository.UserDBRepository    // использую интерфейс из repo слоя
	Cache       repository.UserCacheRepository // использую интерфейс из repo слоя
	authService auth.AuthInterface             // логика авторизации из пакета pkg/auth
	pool        global_db.Pool                 // для начала транзакций
	outboxRepo  outbox.OutboxRepository        // для сохранения в outbox
}

// NewUserLayer - конструктор для части сервисного слоя (пользователи)
func NewUserLayer(repo *repository.UserServiceRepository, authService auth.AuthInterface, pool global_db.Pool, outboxRepo outbox.OutboxRepository) *UserServiceLayer {
	return &UserServiceLayer{
		UserRepo:    repo.DBRepo,
		Cache:       repo.CacheRepo,
		authService: authService,
		pool:        pool,
		outboxRepo:  outboxRepo,
	}
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ДЛЯ КЭШИРОВАНИЯ ====================

// getCacheKey - генерация ключа для кэша
func (s *UserServiceLayer) getCacheKey(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

// cacheUser - сохранение пользователя в кэш
func (s *UserServiceLayer) cacheUser(ctx context.Context, user *domain.User) error {
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
func (s *UserServiceLayer) getCachedUser(ctx context.Context, userID string) (*domain.User, error) {
	key := s.getCacheKey(userID)
	var user domain.User

	if err := s.Cache.Get(ctx, key, &user); err != nil {
		return nil, err
	}

	log.Printf("📦 User retrieved from cache: %s", userID)
	return &user, nil
}

// invalidateUserCache - инвалидация кэша пользователя
func (s *UserServiceLayer) invalidateUserCache(ctx context.Context, userID string) error {
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
// Транзакция специально вынесена в сервисный слой
func (s *UserServiceLayer) CreateUser(ctx context.Context, req *domain.CreateUserRequest, createdBy string) (*domain.User, error) {
	log.Printf("📝 Creating user: email=%s, org=%s", req.Email, req.OrganizationID)

	// ========== ВАЛИДАЦИЯ (вне транзакции) ==========

	// 1. Проверяем права создающего пользователя
	requester, err := s.UserRepo.GetByID(ctx, createdBy)
	if err != nil {
		return nil, fmt.Errorf("requester not found: %w", err)
	}

	if !requester.CanManageUsers() {
		return nil, domain.ErrPermissionDenied
	}

	// Валидация входных данных
	if req.Email == "" {
		return nil, domain.ErrInvalidReqEmail
	}
	if req.Password == "" {
		return nil, domain.ErrReqPasswordRequired
	}
	if len(req.Password) < 6 {
		return nil, domain.ErrReqPasswordTooShort
	}
	if req.FullName == "" {
		return nil, domain.ErrReqFullNameRequired
	}

	// Проверка, что MANAGER не создаёт OWNER или MANAGER
	if requester.Role == domain.RoleManager {
		if req.Role == domain.RoleOwner || req.Role == domain.RoleManager {
			return nil, domain.ErrPermissionDenied
		}
		// Manager может создавать только EMPLOYEE
		req.Role = domain.RoleEmployee
	}

	// 2. Проверка уникальности email (гонку решает БД)
	existing, _ := s.UserRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	// Хеширование пароля (как в CreateUserSystem)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 3. Создаем пользователя через доменную модель
	user := domain.NewUser(req.Email, req.FullName, req.Role, req.OrganizationID)
	user.PasswordHash = string(hashedPassword)

	// Дополнительная валидация
	if err := user.Validate(); err != nil {
		return nil, err
	}

	// 4. работы с транзакцией (создание пользователя + outbox)
	// ==========  ТРАНЗАКЦИОННАЯ ЧАСТЬ ==========

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// defer для отката при ошибке
	success := false
	defer func() {
		if !success {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
			log.Printf("transaction rolled back for user: %s", user.Email)
		}
	}()

	// Сохраняем пользователя
	if err := s.UserRepo.CreateWithTx(ctx, tx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Создаём outbox сообщение
	event := events.NewUserCreatedEvent(user)
	outboxMsg := &outbox.OutboxMessage{
		EventID:    event.GetEventID(),
		EventType:  event.GetEventType(),
		Payload:    event,
		RoutingKey: event.RoutingKey(),
		Status:     outbox.StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Сохраняем в outbox
	if err := s.outboxRepo.SaveTx(ctx, tx, outboxMsg); err != nil {
		return nil, fmt.Errorf("failed to save outbox message: %w", err)
	}

	// Коммит
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	success = true

	// ==========  ПОСТ-КОММИТ (кэш) ==========

	// 5. Сохраняем в кэш
	s.cacheUser(ctx, user)

	log.Printf("✅ User created successfully: ID=%s", user.ID)
	return user, nil
}

// CreateUserSystem - создание самого первого пользователя в организации с ролью OWNER
// Не требует авторизации. Вызывается только при создании и инициализации организации
func (s UserServiceLayer) CreateUserSystem(ctx context.Context, req *domain.CreateUserRequest, ownerPass string) (*domain.User, error) {
	// 1. Валидация входных данных
	if req == nil {
		return nil, domain.ErrInvalidRequest
	}
	if req.Email == "" {
		return nil, domain.ErrInvalidReqEmail
	}
	if req.FullName == "" {
		return nil, domain.ErrReqFullNameRequired
	}
	if ownerPass == "" {
		return nil, domain.ErrReqPasswordRequired
	}
	if req.OrganizationID == "" {
		return nil, domain.ErrReqOrganizationIDRequired
	}
	if len(ownerPass) < 6 {
		return nil, domain.ErrReqPasswordTooShort
	}

	// 2. Проверка, что пользователь с таким email не существует
	existingUser, err := s.UserRepo.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("check existing user failed: %w", err)
	}
	if existingUser != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	// 3. Хэширование пароля
	// специально cost = bcrypt.DefaultCost (это 10, так как это пэт-проект)
	// в перспективе можно вынести это значение в конфиг и повышать (при необходимости)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(ownerPass), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 4. Создаём доменную модель пользователя (роль принудительно OWNER)
	user := domain.NewUser(req.Email, req.FullName, domain.RoleOwner, req.OrganizationID)
	user.PasswordHash = string(hashedPassword)
	user.Status = domain.UserStatusActive

	// 5. Дополнительная валидация доменной модели
	if err := user.Validate(); err != nil {
		return nil, err
	}

	// 6. Сохраняем в БД через репозиторий
	if err := s.UserRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 7. Логируем создание (опционально)
	log.Printf("✅ System user created: id=%s, email=%s, org_id=%s, role=OWNER", user.ID, user.Email, user.OrganizationID)

	return user, nil
}

// GetUserByID - получение пользователя по ID (с кэшированием)
func (s *UserServiceLayer) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
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
func (s *UserServiceLayer) GetUserWithAccessCheck(ctx context.Context, req *domain.GetUserRequest) (*domain.User, error) {
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
func (s *UserServiceLayer) UpdateUser(ctx context.Context, req *domain.UpdateUserRequest) error {
	log.Printf("📝 Updating user: ID=%s by %s", req.UserID, req.RequesterID)

	// 1. Получаем запрашивающего пользователя
	requester, err := s.GetUserByID(ctx, req.RequesterID)
	if err != nil {
		return fmt.Errorf("requester not found: %w", err)
	}

	// 2. Проверяем права
	if !requester.CanUpdateUser(req.UserID, req.Updates) {
		return domain.ErrPermissionDenied
	}

	// ========== ТРАНЗАКЦИЯ ==========

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	success := false
	defer func() {
		if !success {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log.Printf("failed to rollback transaction: %v", rbErr)
			}
		}
	}()

	// 3. Получаем старого пользователя (в транзакции)
	oldUser, err := s.UserRepo.GetByIDWithTx(ctx, tx, req.UserID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	// 4. Создаём копию для нового состояния
	newUser := oldUser.Clone() // ← нужно добавить метод Clone()

	// 5. Применяем обновления к newUser
	if err := s.applyUpdates(newUser, req.Updates); err != nil {
		return err
	}
	// 6. Сохраняем обновления
	if err := s.UserRepo.UpdateWithTx(ctx, tx, newUser); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// 7. Генерируем и сохраняем события
	events, err := s.buildAndSaveUserUpdateEvents(ctx, tx, oldUser, newUser, requester)
	if err != nil {
		return fmt.Errorf("failed to build and save update events to outbox: %w", err)
	}

	// 9. Коммит
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	success = true

	// ========== ПОСТ-КОММИТ ==========

	// 10. Инвалидируем кэш
	s.invalidateUserCache(ctx, req.UserID)

	log.Printf("✅ User updated: ID=%s, events=%d", req.UserID, len(events))
	return nil
}

// buildAndSaveUserEvents - анализирует изменения и сохраняет события в outbox
func (s *UserServiceLayer) buildAndSaveUserUpdateEvents(ctx context.Context, tx global_db.Tx, oldUser, newUser *domain.User, requester *domain.User) ([]baseevent.Event, error) {
	var savedEvents []baseevent.Event

	// 1. Событие: изменение роли
	if oldUser.Role != newUser.Role {
		event := events.NewUserRoleChangedEvent(oldUser, newUser, requester.ID, requester.Role)
		if err := s.saveOutboxEvent(ctx, tx, event); err != nil {
			return savedEvents, fmt.Errorf("failed to save role changed event: %w", err)
		}
		savedEvents = append(savedEvents, event)
		log.Printf("📤 Outbox event saved: user.role_changed for user %s", newUser.ID)
	}

	// 2. Событие: изменение статуса
	if oldUser.Status != newUser.Status {
		event := events.NewUserStatusChangedEvent(oldUser, newUser, requester.ID, requester.Role)
		if err := s.saveOutboxEvent(ctx, tx, event); err != nil {
			return savedEvents, fmt.Errorf("failed to save status changed event: %w", err)
		}
		savedEvents = append(savedEvents, event)
		log.Printf("📤 Outbox event saved: user.status_changed for user %s", newUser.ID)
	}

	// 3. Событие: изменение email
	if oldUser.Email != newUser.Email {
		event := events.NewUserEmailChangedEvent(oldUser, newUser, requester.ID, requester.Role)
		if err := s.saveOutboxEvent(ctx, tx, event); err != nil {
			return savedEvents, fmt.Errorf("failed to save email changed event: %w", err)
		}
		savedEvents = append(savedEvents, event)
		log.Printf("📤 Outbox event saved: user.email_changed for user %s", newUser.ID)
	}

	return savedEvents, nil
}

// saveOutboxEvent - сохраняет событие в outbox таблицу
func (s *UserServiceLayer) saveOutboxEvent(ctx context.Context, tx global_db.Tx, event baseevent.Event) error {

	// Создаём outbox сообщение
	outboxMsg := &outbox.OutboxMessage{
		EventID:    event.GetEventID(),
		EventType:  event.GetEventType(),
		Payload:    event,
		RoutingKey: event.RoutingKey(),
		Status:     outbox.StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Сохраняем в outbox
	if err := s.outboxRepo.SaveTx(ctx, tx, outboxMsg); err != nil {
		return fmt.Errorf("failed to save outbox message: %w", err)
	}

	return nil
}

// Вспомогательная функция для применения обновлений
func (s *UserServiceLayer) applyUpdates(user *domain.User, updates map[string]interface{}) error {
	if fullName, ok := updates["full_name"].(string); ok {
		user.UpdateProfile(fullName)
	}

	if email, ok := updates["email"].(string); ok {
		if err := user.UpdateEmail(email); err != nil {
			return err
		}
	}

	if role, ok := updates["role"].(domain.Role); ok {
		switch role {
		case domain.RoleManager:
			if err := user.PromoteToManager(); err != nil {
				return err
			}
		case domain.RoleEmployee:
			if err := user.DemoteToEmployee(); err != nil {
				return err
			}
		case domain.RoleOwner:
			// OWNER нельзя создать через UpdateUser, только через TransferOwnership
			return domain.ErrInvalidRoleTransition
		}
	}

	if status, ok := updates["status"].(domain.UserStatus); ok {
		switch status {
		case domain.UserStatusSuspended:
			if err := user.Suspend(); err != nil {
				return err
			}
		case domain.UserStatusActive:
			if err := user.Activate(); err != nil {
				return err
			}
		}
	}

	return nil
}

// UpdateMyProfile - обновление профиля текущего пользователя
func (s *UserServiceLayer) UpdateMyProfile(ctx context.Context, userID string, fullName *string) error {
	log.Printf("📝 UpdateMyProfile: userID=%s", userID)

	// Подготавливаем обновления
	updates := make(map[string]interface{})
	if fullName != nil && *fullName != "" {
		updates["full_name"] = *fullName
	}

	// Если нет полей для обновления, просто возвращаем nil
	if len(updates) == 0 {
		log.Printf("UpdateMyProfile: нет полей для обновления")
		return nil
	}

	// Вызываем существующий UpdateUser
	req := &domain.UpdateUserRequest{
		UserID:      userID,
		RequesterID: userID, // Обновляем свой профиль, поэтому requester = target
		Updates:     updates,
	}

	return s.UpdateUser(ctx, req)
}

// DeleteUser - удаление пользователя
func (s *UserServiceLayer) DeleteUser(ctx context.Context, req *domain.DeleteUserRequest) error {
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
func (s *UserServiceLayer) ListUsers(ctx context.Context, req *domain.ListUsersRequest) (*domain.ListUsersResponse, error) {
	log.Printf("📝 Listing users: org_id=%s, offset=%d, limit=%d, filters=%v", req.OrganizationID, req.Pagination.Offset, req.Pagination.Limit, req.Filters)

	// 1. Извлекаем фильтры из map
	var roleFilter *domain.Role
	if roleStr, ok := req.Filters["role"]; ok && roleStr != "" {
		role := domain.Role(roleStr)
		roleFilter = &role
	}

	var statusFilter *domain.UserStatus
	if statusStr, ok := req.Filters["status"]; ok && statusStr != "" {
		status := domain.UserStatus(statusStr)
		statusFilter = &status
	}

	var searchQuery string
	if search, ok := req.Filters["search"]; ok {
		searchQuery = search
	}

	// 2. Получаем список пользователей из репозитория
	users, err := s.UserRepo.ListWithFilters(ctx, req.OrganizationID, roleFilter, statusFilter, searchQuery, req.Pagination.Offset, req.Pagination.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// 3. Получаем общее количество
	totalCount, err := s.UserRepo.CountWithFilters(ctx, req.OrganizationID, roleFilter, statusFilter, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// 4. Определяем, есть ли ещё записи
	hasMore := totalCount > req.Pagination.Offset+req.Pagination.Limit

	log.Printf("✅ Retrieved %d users (total: %d, hasMore: %v)", len(users), totalCount, hasMore)

	return &domain.ListUsersResponse{
		Users:      users,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

// ListUsersByOrganization - получение списка пользователей организации
func (s *UserServiceLayer) ListUsersByOrganization(ctx context.Context, organizationID string, offset, limit int) ([]*domain.User, error) {
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
func (s *UserServiceLayer) AuthenticateUser(ctx context.Context, email, password string) (*domain.User, error) {
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
func (s *UserServiceLayer) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
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
func (s *UserServiceLayer) TransferOwnership(ctx context.Context, currentOwnerID, newOwnerID, organizationID string) error {
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
func (s *UserServiceLayer) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.UserRepo.GetByEmail(ctx, email)
}

// CheckUserExists - проверка существования пользователя
func (s *UserServiceLayer) CheckUserExists(ctx context.Context, userID string) (bool, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return false, nil
	}
	return user != nil, nil
}

// GetUsersByRole - получение пользователей по роли
func (s *UserServiceLayer) GetUsersByRole(ctx context.Context, organizationID string, role domain.Role) ([]*domain.User, error) {
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

// BatchGetUsersByIDs - массовое получение пользователей по списку ID
// Возвращает map "ID → User" и список ID, которые не найдены (с использованием кэша)
func (s *UserServiceLayer) BatchGetUsersByIDs(ctx context.Context, userIDs []string) (map[string]*domain.User, []string, error) {
	// 1. Проверка входных данных
	if len(userIDs) == 0 {
		log.Printf("[WARN] BatchGetUsersByIDs: empty user IDs list")
		return make(map[string]*domain.User), []string{}, nil
	}

	// 2. Фильтрация пустых ID
	validIDs := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		if id != "" {
			validIDs = append(validIDs, id)
		}
	}

	if len(validIDs) == 0 {
		log.Printf("[WARN] BatchGetUsersByIDs: no valid IDs after filtering")
		return make(map[string]*domain.User), userIDs, nil
	}

	usersMap := make(map[string]*domain.User) // инициализируем мапу для результата
	notFoundIDs := make([]string, 0)          // инициализируем слайс для результата (ID, которые не найдены)
	missingFromCache := make([]string, 0)     // инициализируем слайс для необработанных ID по причине ошибок в кэшэ (чтобы получить их из базы)

	// 3. Сначала пробуем получить из кэша
	for _, id := range validIDs {
		user, err := s.getCachedUser(ctx, id)
		if err == nil {
			usersMap[id] = user
			log.Printf("📦 BatchGetUsersByIDs: user %s retrieved from cache", id)
		} else {
			missingFromCache = append(missingFromCache, id)
		}
	}

	// 4. Если все пользователи найдены в кэше
	if len(missingFromCache) == 0 {
		log.Printf("✅ BatchGetUsersByIDs: all %d users found in cache", len(usersMap))
		return usersMap, []string{}, nil
	}

	// 5. Получаем недостающих пользователей из БД
	dbUsers, err := s.UserRepo.BatchGetByIDs(ctx, missingFromCache)
	if err != nil {
		log.Printf("❌ BatchGetUsersByIDs: ошибка БД: %v", err)
		return nil, nil, fmt.Errorf("database error: %w", err)
	}

	// 6. Сохраняем полученных из БД пользователей в кэш и в map
	foundInDB := make(map[string]bool)
	for _, user := range dbUsers {
		usersMap[user.ID] = user
		foundInDB[user.ID] = true

		// Сохраняем в кэш асинхронно (чтобы не блокировать ответ)
		go func(u *domain.User) {
			if err := s.cacheUser(context.Background(), u); err != nil {
				log.Printf("⚠️ Failed to cache user %s: %v", u.ID, err)
			}
		}(user)
	}
	// 7. Определяем, какие ID не найдены (ни в кэше, ни в БД)
	for _, id := range missingFromCache {
		if !foundInDB[id] {
			notFoundIDs = append(notFoundIDs, id)
		}
	}

	log.Printf("✅ BatchGetUsersByIDs: total=%d, from_cache=%d, from_db=%d, not_found=%d",
		len(userIDs), len(usersMap)-len(dbUsers), len(dbUsers), len(notFoundIDs))

	return usersMap, notFoundIDs, nil
}
