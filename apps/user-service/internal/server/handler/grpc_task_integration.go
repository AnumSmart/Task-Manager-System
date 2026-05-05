package handler

import (
	pb "api/gen/go/user/v1"
	"context"
	"errors"
	"fmt"
	"log"
	"user-service/internal/converter"
	"user-service/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ValidateUser - комплексная проверка пользователя перед назначением задачи
// Проверяет существование, активность и соответствие ролям
// 🔒 API Key авторизация
func (s *UserServerHandler) ValidateUser(ctx context.Context, req *pb.ValidateUserRequest) (*pb.ValidateUserResponse, error) {
	// 1. Проверка контекста
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	requestID := req.GetRequestId()
	userID := req.GetUserId()
	allowedRoles := req.GetAllowedRoles()

	log.Printf("📝 ValidateUser: request_id=%s, user_id=%s, allowed_roles=%v",
		requestID, userID, allowedRoles)

	// 2. Валидация входных параметров
	if userID == "" {
		log.Printf("⚠️ ValidateUser [%s]: user_id is required", requestID)
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// 3. Получение пользователя через сервисный слой (используем существующий метод)
	user, err := s.UserServerService.User.GetUserByID(ctx, userID)
	if err != nil {
		log.Printf("❌ ValidateUser [%s]: ошибка получения пользователя %s: %v",
			requestID, userID, err)

		// Определяем тип ошибки для validation_error
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			return &pb.ValidateUserResponse{
				Success:         true,
				IsValid:         false,
				User:            nil,
				ValidationError: "user not found",
			}, nil

		case errors.Is(err, domain.ErrInvalidUserID):
			return &pb.ValidateUserResponse{
				Success:         true,
				IsValid:         false,
				User:            nil,
				ValidationError: "invalid user id format",
			}, nil

		default:
			// Внутренняя ошибка (проблема с БД и т.д.)
			log.Printf("❌ ValidateUser [%s]: внутренняя ошибка: %v", requestID, err)
			return nil, status.Error(codes.Internal, "failed to get user")
		}
	}

	// 4. Проверка, что пользователь не удалён (soft delete)
	if user.DeletedAt != nil {
		log.Printf("⚠️ ValidateUser [%s]: пользователь %s удалён", requestID, userID)
		return &pb.ValidateUserResponse{
			Success:         true,
			IsValid:         false,
			User:            nil,
			ValidationError: "user is deleted",
		}, nil
	}

	// 5. Проверка статуса пользователя (используем доменный метод)
	if user.IsSuspended() {
		log.Printf("⚠️ ValidateUser [%s]: пользователь %s заблокирован", requestID, userID)
		return &pb.ValidateUserResponse{
			Success:         true,
			IsValid:         false,
			User:            nil,
			ValidationError: "user is suspended",
		}, nil
	}

	// 6. Проверка, что пользователь активен
	if !user.IsActive() {
		log.Printf("⚠️ ValidateUser [%s]: пользователь %s неактивен (status=%s)",
			requestID, userID, user.Status)
		return &pb.ValidateUserResponse{
			Success:         true,
			IsValid:         false,
			User:            nil,
			ValidationError: "user is not active",
		}, nil
	}

	// 7. Проверка роли (если указаны allowed_roles)
	if len(allowedRoles) > 0 {
		roleAllowed := false
		for _, allowedRole := range allowedRoles {
			// Конвертируем proto.Role в domain.Role и сравниваем
			if user.Role == converter.ToDomainRole(allowedRole) {
				roleAllowed = true
				break
			}
		}

		if !roleAllowed {
			log.Printf("⚠️ ValidateUser [%s]: роль %s не входит в разрешённые: %v",
				requestID, user.Role, allowedRoles)
			return &pb.ValidateUserResponse{
				Success:         true,
				IsValid:         false,
				User:            nil,
				ValidationError: "user role is not allowed for this operation",
			}, nil
		}
	}

	// 8. Успешная валидация - конвертируем пользователя через существующий конвертер
	protoUser := converter.ToProtoUser(user)

	log.Printf("✅ ValidateUser [%s]: пользователь %s (%s) валиден, роль: %s",
		requestID, user.ID, user.Email, user.Role)

	return &pb.ValidateUserResponse{
		Success:         true,
		IsValid:         true,
		User:            protoUser,
		ValidationError: "",
	}, nil
}

// CheckUserExists - быстрая проверка существования пользователя
// Используется для валидации перед созданием задачи
// 🔒 API Key авторизация
func (s *UserServerHandler) CheckUserExists(ctx context.Context, req *pb.CheckUserExistsRequest) (*pb.CheckUserExistsResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	requestID := req.GetRequestId()
	userID := req.GetUserId()

	log.Printf("📝 CheckUserExists: request_id=%s, user_id=%s", requestID, userID)

	// 2. Валидация входных параметров
	if userID == "" {
		log.Printf("⚠️ CheckUserExists [%s]: user_id is required", requestID)
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// 3. Получение пользователя через сервисный слой (используем существующий метод)
	user, err := s.UserServerService.User.GetUserByID(ctx, userID)
	if err != nil {
		// Пользователь не найден - это не ошибка, а ожидаемый бизнес-кейс
		if errors.Is(err, domain.ErrUserNotFound) {
			log.Printf("📝 CheckUserExists [%s]: пользователь %s не найден", requestID, userID)
			return &pb.CheckUserExistsResponse{
				Success:      true,
				Exists:       false,
				IsActive:     false,
				ErrorMessage: "",
			}, nil
		}

		// Неверный формат ID
		if errors.Is(err, domain.ErrInvalidUserID) {
			log.Printf("⚠️ CheckUserExists [%s]: неверный формат user_id: %s", requestID, userID)
			return &pb.CheckUserExistsResponse{
				Success:      true,
				Exists:       false,
				IsActive:     false,
				ErrorMessage: "invalid user id format",
			}, nil
		}

		// Внутренняя ошибка (проблема с БД)
		log.Printf("❌ CheckUserExists [%s]: внутренняя ошибка: %v", requestID, err)
		return nil, status.Error(codes.Internal, "failed to check user existence")
	}

	// 4. Проверка soft delete (пользователь удалён)
	if user.DeletedAt != nil {
		log.Printf("📝 CheckUserExists [%s]: пользователь %s удалён (soft delete)", requestID, userID)
		return &pb.CheckUserExistsResponse{
			Success:      true,
			Exists:       true,  // Пользователь существует в БД
			IsActive:     false, // Но не активен (удалён)
			ErrorMessage: "user is deleted",
		}, nil
	}

	// 5. Проверка активности пользователя
	isActive := user.IsActive() && !user.IsSuspended()

	// Дополнительное логирование статуса
	if !isActive {
		log.Printf("📝 CheckUserExists [%s]: пользователь %s существует, но неактивен (status=%s)",
			requestID, userID, user.Status)
	} else {
		log.Printf("✅ CheckUserExists [%s]: пользователь %s существует и активен", requestID, userID)
	}

	// 6. Успешный ответ
	return &pb.CheckUserExistsResponse{
		Success:      true,
		Exists:       true,
		IsActive:     isActive,
		ErrorMessage: "",
	}, nil
}

// GetUserByID - получение пользователя по ID (алиас для GetUser)
// Выделен в отдельный метод для task-service для чёткого разделения ответственности
// 🔒 API Key авторизация
func (s *UserServerHandler) GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.GetUserResponse, error) {
	// 1. Проверка контекста
	select {
	case <-ctx.Done():
		log.Printf("❌ GetUserByID: контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	requestID := req.GetRequestId()
	userID := req.GetUserId()

	log.Printf("📝 GetUserByID: request_id=%s, user_id=%s", requestID, userID)

	// 2. Валидация входных параметров
	if userID == "" {
		log.Printf("⚠️ GetUserByID [%s]: user_id is required", requestID)
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// 3. Получение пользователя через сервисный слой
	user, err := s.UserServerService.User.GetUserByID(ctx, userID)
	if err != nil {
		log.Printf("❌ GetUserByID [%s]: ошибка получения пользователя %s: %v",
			requestID, userID, err)

		// Определяем тип ошибки для понятного ответа
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			return &pb.GetUserResponse{
				Success:      false,
				User:         nil,
				ErrorMessage: "user not found",
			}, nil

		case errors.Is(err, domain.ErrInvalidUserID):
			return &pb.GetUserResponse{
				Success:      false,
				User:         nil,
				ErrorMessage: "invalid user id format",
			}, nil

		default:
			// Внутренняя ошибка (проблема с БД)
			log.Printf("❌ GetUserByID [%s]: внутренняя ошибка: %v", requestID, err)
			return nil, status.Error(codes.Internal, "failed to get user")
		}
	}

	// 4. Проверка soft delete
	if user.DeletedAt != nil {
		log.Printf("⚠️ GetUserByID [%s]: пользователь %s удалён (soft delete)", requestID, userID)
		return &pb.GetUserResponse{
			Success:      false,
			User:         nil,
			ErrorMessage: "user is deleted",
		}, nil
	}

	// 5. Конвертация пользователя в protobuf
	protoUser := converter.ToProtoUser(user)
	if protoUser == nil {
		log.Printf("❌ GetUserByID [%s]: не удалось сконвертировать пользователя %s", requestID, userID)
		return nil, status.Error(codes.Internal, "failed to convert user")
	}

	// Убедимся, что Telegram ID не скрыт
	if user.TelegramID != nil {
		protoUser.TelegramId = user.TelegramID
	}

	if user.TelegramUsername != nil {
		protoUser.TelegramUsername = user.TelegramUsername
	}

	log.Printf("✅ GetUserByID [%s]: успешно получен пользователь %s (%s), роль: %s, статус: %s",
		requestID, user.ID, user.Email, user.Role, user.Status)

	return &pb.GetUserResponse{
		Success:      true,
		User:         protoUser,
		ErrorMessage: "",
	}, nil
}

// BatchGetUsers - массовое получение пользователей
// Оптимизация: вместо N вызовов GetUser - один вызов BatchGetUsers
// 🔒 API Key авторизация
func (s *UserServerHandler) BatchGetUsers(ctx context.Context, req *pb.BatchGetUsersRequest) (*pb.BatchGetUsersResponse, error) {
	// 1. Проверка контекста
	select {
	case <-ctx.Done():
		log.Printf("❌ BatchGetUsers: контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	requestID := req.GetRequestId()
	userIDs := req.GetUserIds()

	log.Printf("📝 BatchGetUsers: request_id=%s, count=%d", requestID, len(userIDs))

	// 2. Валидация входных параметров
	if len(userIDs) == 0 {
		log.Printf("⚠️ BatchGetUsers [%s]: user_ids is empty", requestID)
		return &pb.BatchGetUsersResponse{
			Success:     true,
			NotFoundIds: []string{},
		}, nil
	}

	// 3. Ограничение на количество запрашиваемых пользователей (защита от DoS)
	maxBatchSize := 100
	if len(userIDs) > maxBatchSize {
		log.Printf("⚠️ BatchGetUsers [%s]: слишком много запросов: %d (макс: %d)",
			requestID, len(userIDs), maxBatchSize)
		return nil, status.Error(codes.InvalidArgument,
			fmt.Sprintf("too many user ids: max %d", maxBatchSize))
	}

	// 4. Удаляем дубликаты
	uniqueIDs := make([]string, 0, len(userIDs))
	seen := make(map[string]bool)
	for _, id := range userIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	if len(uniqueIDs) != len(userIDs) {
		log.Printf("📝 BatchGetUsers [%s]: удалено дубликатов: %d",
			requestID, len(userIDs)-len(uniqueIDs))
	}

	// 5. Массовое получение пользователей через сервисный слой
	usersMap, notFoundIDs, err := s.UserServerService.User.BatchGetUsersByIDs(ctx, uniqueIDs)
	if err != nil {
		log.Printf("❌ BatchGetUsers [%s]: ошибка получения пользователей: %v", requestID, err)
		return nil, status.Error(codes.Internal, "failed to get users")
	}

	// 6. Конвертация пользователей в protobuf (используем новую функцию конвертера)
	pbUsers := converter.ToProtoUserMap(usersMap)

	log.Printf("✅ BatchGetUsers [%s]: получено %d пользователей, не найдено %d",
		requestID, len(pbUsers), len(notFoundIDs))

	return &pb.BatchGetUsersResponse{
		Success:      true,
		Users:        pbUsers,
		NotFoundIds:  notFoundIDs,
		ErrorMessage: "",
	}, nil
}
