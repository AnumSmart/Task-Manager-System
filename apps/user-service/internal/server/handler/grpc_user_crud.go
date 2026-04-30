package handler

import (
	pb "api/gen/go/user/v1"
	"context"
	"errors"
	"fmt"
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

	// 1. Извлекаем данные текущего пользователя из контекста (добавлены интерсептором)
	currentUserID, ok := ctx.Value(interceptors.ContextKeyUserID).(string)
	if !ok || currentUserID == "" {
		return nil, status.Error(codes.Unauthenticated, "не удалось получить ID текущего пользователя")
	}

	currentUserRole, _ := ctx.Value(interceptors.ContextKeyRole).(string)
	currentUserOrgID, _ := ctx.Value(interceptors.ContextKeyOrganizationID).(string)

	log.Printf("🔐 Текущий пользователь: id=%s, role=%s, org_id=%s", currentUserID, currentUserRole, currentUserOrgID)

	// 2. Валидация входных данных
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id обязателен")
	}

	// 3. Получаем запрашиваемого пользователя через сервисный слой
	targetUser, err := s.UserServerService.User.GetUserByID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			log.Printf("❌ Пользователь не найден: %s", req.GetUserId())
			return nil, status.Error(codes.NotFound, "пользователь не найден")
		}
		log.Printf("❌ Ошибка получения пользователя: %v", err)
		return nil, status.Error(codes.Internal, "внутренняя ошибка сервера")
	}

	// 4. Получаем полные данные текущего пользователя (для проверки прав)
	//    Нужно получить из БД, т.к. в контексте только ID, роль и org_id
	currentUser, err := s.UserServerService.User.GetUserByID(ctx, currentUserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			log.Printf("❌ Текущий пользователь не найден: %s", currentUserID)
			return nil, status.Error(codes.Unauthenticated, "пользователь не найден")
		}
		log.Printf("❌ Ошибка получения текущего пользователя: %v", err)
		return nil, status.Error(codes.Internal, "внутренняя ошибка сервера")
	}

	// 5. Проверяем права доступа
	if !s.canAccessUser(currentUser, targetUser) {
		log.Printf("⚠️ Доступ запрещён: пользователь %s (role=%s, org=%s) пытается получить данные пользователя %s (role=%s, org=%s)",
			currentUser.ID, currentUser.Role, currentUser.OrganizationID,
			targetUser.ID, targetUser.Role, targetUser.OrganizationID)
		return nil, status.Error(codes.PermissionDenied, "доступ запрещён")
	}
	// Используем конвертер с правами
	response := &pb.GetUserResponse{
		User: converter.ToProtoUserWithPermissions(targetUser, currentUser),
	}

	return response, nil
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

	// 1. Извлекаем ID текущего пользователя из контекста
	currentUserID, ok := ctx.Value(interceptors.ContextKeyUserID).(string)
	if !ok || currentUserID == "" {
		return nil, status.Error(codes.Unauthenticated, "не удалось получить ID текущего пользователя")
	}

	// 2. Валидация входных данных
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id обязателен")
	}

	// 3. Формируем map обновлений из запроса (используем конвертер)
	updates := converter.ToDomainUserUpdates(req)

	// 4. Если нет полей для обновления, возвращаем текущего пользователя
	if len(updates) == 0 {
		log.Printf("⚠️ Нет полей для обновления пользователя %s", req.GetUserId())

		// Получаем текущие данные пользователя для ответа
		targetUser, err := s.UserServerService.User.GetUserByID(ctx, req.GetUserId())
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				return nil, status.Error(codes.NotFound, "пользователь не найден")
			}
			return nil, status.Error(codes.Internal, "внутренняя ошибка сервера")
		}

		// Получаем текущего пользователя (для прав в ответе)
		currentUser, err := s.UserServerService.User.GetUserByID(ctx, currentUserID)
		if err != nil {
			return nil, status.Error(codes.Internal, "внутренняя ошибка сервера")
		}

		return &pb.UpdateUserResponse{
			User: converter.ToProtoUserWithPermissions(targetUser, currentUser),
		}, nil
	}

	// 5. Вызываем сервисный слой
	err := s.UserServerService.User.UpdateUser(ctx, &domain.UpdateUserRequest{
		UserID:      req.GetUserId(),
		RequesterID: currentUserID,
		Updates:     updates,
	})

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			log.Printf("❌ Пользователь не найден: %s", req.GetUserId())
			return nil, status.Error(codes.NotFound, "пользователь не найден")

		case errors.Is(err, domain.ErrUserAlreadyExists):
			log.Printf("❌ Email уже используется: %s", req.GetEmail())
			return nil, status.Error(codes.AlreadyExists, "пользователь с таким email уже существует")

		case errors.Is(err, domain.ErrPermissionDenied):
			log.Printf("❌ Доступ запрещён: пользователь %s пытается обновить %s", currentUserID, req.GetUserId())
			return nil, status.Error(codes.PermissionDenied, "доступ запрещён")

		case errors.Is(err, domain.ErrInvalidEmail):
			log.Printf("❌ Неверный формат email: %s", req.GetEmail())
			return nil, status.Error(codes.InvalidArgument, "неверный формат email")

		case errors.Is(err, domain.ErrInvalidRoleTransition):
			log.Printf("❌ Недопустимое изменение роли: %v", err)
			return nil, status.Error(codes.PermissionDenied, "недопустимое изменение роли")

		default:
			log.Printf("❌ Ошибка обновления пользователя: %v", err)
			return nil, status.Error(codes.Internal, "внутренняя ошибка сервера")
		}
	}

	log.Printf("✅ UpdateUser успешно выполнен: user_id=%s", req.GetUserId())

	// 6. Получаем обновлённые данные для ответа
	updatedUser, err := s.UserServerService.User.GetUserByID(ctx, req.GetUserId())
	if err != nil {
		// Даже если не получили обновлённые данные, возвращаем успех
		log.Printf("⚠️ Не удалось получить обновлённые данные пользователя: %v", err)
		return &pb.UpdateUserResponse{
			Success:      true,
			ErrorMessage: fmt.Sprintf("⚠️ Не удалось получить обновлённые данные пользователя"),
		}, nil
	}

	// 7. Получаем текущего пользователя (для прав в ответе)
	currentUser, err := s.UserServerService.User.GetUserByID(ctx, currentUserID)
	if err != nil {
		log.Printf("⚠️ Не удалось получить данные текущего пользователя: %v", err)
		return &pb.UpdateUserResponse{
			User: converter.ToProtoUser(updatedUser),
		}, nil
	}

	return &pb.UpdateUserResponse{
		User: converter.ToProtoUserWithPermissions(updatedUser, currentUser),
	}, nil
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

	// 1. Извлекаем ID текущего пользователя из контекста
	currentUserID, ok := ctx.Value(interceptors.ContextKeyUserID).(string)
	if !ok || currentUserID == "" {
		return nil, status.Error(codes.Unauthenticated, "не удалось получить ID текущего пользователя")
	}

	// 2. Валидация входных данных
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id обязателен")
	}

	// 3. Нельзя удалить самого себя (дополнительная проверка на уровне хэндлера)
	if currentUserID == req.GetUserId() {
		log.Printf("⚠️ Пользователь %s попытался удалить сам себя", currentUserID)
		return nil, status.Error(codes.PermissionDenied, "нельзя удалить свой собственный аккаунт")
	}

	// 4. Преобразуем soft_delete в hard_delete (инвертируем логику)
	hardDelete := !req.GetSoftDelete()

	// 4. Вызываем сервисный слой
	err := s.UserServerService.User.DeleteUser(ctx, &domain.DeleteUserRequest{
		UserID:      req.GetUserId(),
		RequesterID: currentUserID,
		HardDelete:  hardDelete, // false по умолчанию (soft delete)
	})

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			log.Printf("❌ Пользователь не найден: %s", req.GetUserId())
			return nil, status.Error(codes.NotFound, "пользователь не найден")

		case errors.Is(err, domain.ErrPermissionDenied):
			log.Printf("❌ Доступ запрещён: пользователь %s пытается удалить %s", currentUserID, req.GetUserId())
			return nil, status.Error(codes.PermissionDenied, "только владелец организации может удалять пользователей")

		default:
			log.Printf("❌ Ошибка удаления пользователя: %v", err)
			return nil, status.Error(codes.Internal, "внутренняя ошибка сервера")
		}
	}

	log.Printf("✅ DeleteUser успешно выполнен: user_id=%s, soft_delete=%v", req.GetUserId(), req.GetSoftDelete())

	return &pb.DeleteUserResponse{
		Success: true,
	}, nil
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

	// 1. Извлекаем данные текущего пользователя из контекста
	currentUserID, ok := ctx.Value(interceptors.ContextKeyUserID).(string)
	if !ok || currentUserID == "" {
		return &pb.ListUsersResponse{
			Success:      false,
			ErrorMessage: "не авторизован",
		}, nil
	}

	// 2. Получаем полные данные текущего пользователя
	currentUser, err := s.UserServerService.User.GetUserByID(ctx, currentUserID)
	if err != nil {
		log.Printf("❌ Ошибка получения текущего пользователя: %v", err)
		return &pb.ListUsersResponse{
			Success:      false,
			ErrorMessage: "внутренняя ошибка сервера",
		}, nil
	}

	// 3. Обработка Employee (только себя)
	if currentUser.Role == domain.RoleEmployee {
		log.Printf("🔐 Employee %s запрашивает только свой профиль", currentUserID)
		return converter.ToProtoListUsersResponseForSingleUser(currentUser), nil
	}

	// 4. Конвертируем запрос в доменную структуру
	listReq := converter.ToDomainListUsersRequest(currentUserID, currentUser.OrganizationID, req)

	// 5. Вызываем сервисный слой
	listResp, err := s.UserServerService.User.ListUsers(ctx, listReq)
	if err != nil {
		log.Printf("❌ Ошибка получения списка пользователей: %v", err)
		return &pb.ListUsersResponse{
			Success:      false,
			ErrorMessage: "внутренняя ошибка сервера",
		}, nil
	}

	// 6. Конвертируем ответ в protobuf
	return converter.ToProtoListUsersResponse(listResp, currentUser, req.GetPage(), req.GetPageSize()), nil
}

// canAccessUser (вспомогательная функция) - проверяет, может ли currentUser получить данные targetUser
func (s *UserServerHandler) canAccessUser(currentUser, targetUser *domain.User) bool {
	// Свои данные всегда можно посмотреть
	if currentUser.ID == targetUser.ID {
		return true
	}

	// Пользователи должны быть из одной организации
	if currentUser.OrganizationID != targetUser.OrganizationID {
		return false
	}

	// Owner и Manager могут видеть всех в своей организации
	switch currentUser.Role {
	case domain.RoleOwner, domain.RoleManager:
		return true
	default:
		return false
	}
}
