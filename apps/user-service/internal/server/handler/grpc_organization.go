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

// SetupInitialOrganization - вызывается при первом запуске для создания организации и владельца
func (s *UserServerHandler) SetupInitialOrganization(ctx context.Context, req *pb.SetupInitialOrganizationRequest) (*pb.SetupInitialOrganizationResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 SetupInitialOrganization вызван: org_name=%s, email=%s",
		req.GetOrganizationName(), req.GetOwnerEmail())

	// 1. Валидация входных данных
	if req.GetOrganizationName() == "" {
		return &pb.SetupInitialOrganizationResponse{
			Success:      false,
			ErrorMessage: "organization_name is required",
		}, nil
	}
	if req.GetOwnerEmail() == "" {
		return &pb.SetupInitialOrganizationResponse{
			Success:      false,
			ErrorMessage: "owner_email is required",
		}, nil
	}
	if req.GetOwnerPassword() == "" {
		return &pb.SetupInitialOrganizationResponse{
			Success:      false,
			ErrorMessage: "owner_password is required",
		}, nil
	}
	if req.GetOwnerFullName() == "" {
		return &pb.SetupInitialOrganizationResponse{
			Success:      false,
			ErrorMessage: "owner_full_name is required",
		}, nil
	}

	// 2. Проверка, что система ещё не инициализирована
	exists, err := s.UserServerService.Organization.IsInitialized(ctx)
	if err != nil {
		log.Printf("❌ SetupInitialOrganization error checking initialization: %v", err)
		return nil, status.Error(codes.Internal, "failed to check system state")
	}
	if exists {
		return &pb.SetupInitialOrganizationResponse{
			Success:      false,
			ErrorMessage: "system already initialized",
		}, nil
	}

	// 3. Создаём доменную модель организации
	domainOrg := domain.NewOrganization(req.GetOrganizationName(), "") // OwnerID пока пустой, будет обновлён после создания пользователя

	// 4. Сохраняем организацию через сервисный слой
	org, err := s.UserServerService.Organization.SaveOrganization(ctx, domainOrg)
	if err != nil {
		log.Printf("❌ SetupInitialOrganization error saving organization: %v", err)
		return nil, status.Error(codes.Internal, "failed to create organization")
	}

	// 5. Создаём запрос на создание пользователя (используем существующий CreateUserRequest)
	createUserReq := &domain.CreateUserRequest{
		OrganizationID: org.ID,
		Email:          req.GetOwnerEmail(),
		FullName:       req.GetOwnerFullName(),
		Role:           domain.RoleOwner,
	}

	// 6. Создаём пользователя через существующий метод CreateUser
	var user *domain.User
	// ВНИМАНИЕ: для первого пользователя нет createdBy, поэтому передаём special system user
	// Временно создаём пользователя без проверки прав (можно добавить флаг systemInit)
	user, err = s.UserServerService.User.CreateUserSystem(ctx, createUserReq, req.GetOwnerPassword())
	if err != nil {
		// Проверяем, может пользователь уже существует?
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			// Пользователь с таким email уже есть в БД
			existingUser, getUserErr := s.UserServerService.User.GetUserByEmail(ctx, req.GetOwnerEmail())
			if getUserErr != nil {
				log.Printf("❌ SetupInitialOrganization: user exists but failed to fetch: %v", getUserErr)
				_ = s.UserServerService.Organization.DeleteOrganization(ctx, org.ID)
				return nil, status.Error(codes.Internal, "failed to verify existing user")
			}

			// Проверяем, что пользователь имеет роль OWNER
			if existingUser.Role != domain.RoleOwner {
				_ = s.UserServerService.Organization.DeleteOrganization(ctx, org.ID)
				return &pb.SetupInitialOrganizationResponse{
					Success:      false,
					ErrorMessage: "user with this email already exists but is not an owner",
				}, nil
			}

			// Проверяем, что пользователь принадлежит этой организации
			if existingUser.OrganizationID != org.ID {
				_ = s.UserServerService.Organization.DeleteOrganization(ctx, org.ID)
				return &pb.SetupInitialOrganizationResponse{
					Success:      false,
					ErrorMessage: "user with this email belongs to another organization",
				}, nil
			}

			// Всё хорошо — используем существующего пользователя
			user = existingUser
			log.Printf("📝 SetupInitialOrganization: using existing owner user: %s", user.ID)
		} else {
			// Другая ошибка при создании пользователя
			log.Printf("❌ SetupInitialOrganization error creating user: %v", err)
			_ = s.UserServerService.Organization.DeleteOrganization(ctx, org.ID)
			return nil, status.Error(codes.Internal, "failed to create owner user")
		}
	} else {
		// Пользователь успешно создан
		log.Printf("📝 SetupInitialOrganization: created new owner user: %s", user.ID)
	}

	// 7. Обновляем организацию: устанавливаем OwnerID
	err = s.UserServerService.Organization.UpdateOwner(ctx, org.ID, user.ID)
	if err != nil {
		log.Printf("⚠️ SetupInitialOrganization: failed to update organization owner: %v", err)
		// Не критично, но логируем
	}

	// 8. Активируем организацию
	err = s.UserServerService.Organization.ActivateOrganization(ctx, org.ID, user)
	if err != nil {
		log.Printf("⚠️ SetupInitialOrganization: organization created but failed to activate: %v", err)
		// Не критично, организация создана, но не активна
	}

	log.Printf("✅ SetupInitialOrganization успешен: org_id=%s, user_id=%s", org.ID, user.ID)

	return &pb.SetupInitialOrganizationResponse{
		Success:      true,
		Organization: converter.ToProtoOrganization(org),
		Owner:        converter.ToProtoUser(user),
		Message:      "system initialized successfully",
	}, nil
}

// GetOrganization - получение информации об организации текущего пользователя
// Organization ID извлекается из JWT токена в metadata
func (s *UserServerHandler) GetOrganization(ctx context.Context, req *pb.GetOrganizationRequest) (*pb.GetOrganizationResponse, error) {
	// 1. Проверка контекста (graceful shutdown)
	select {
	case <-ctx.Done():
		log.Printf("❌ GetOrganization: контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 GetOrganization вызван: request_id=%s", req.GetRequestId())

	// 2. Извлечение organization_id из контекста (добавлен интерсептором)
	orgID, ok := ctx.Value(interceptors.ContextKeyOrganizationID).(string)
	if !ok || orgID == "" {
		log.Printf("❌ GetOrganization: organization_id не найден в контексте")
		return &pb.GetOrganizationResponse{
			Success:      false,
			ErrorMessage: "не авторизован",
		}, nil
	}

	// 3. Вызов сервисного слоя для получения организации
	org, err := s.UserServerService.Organization.GetOrganizationByID(ctx, orgID)
	if err != nil {
		log.Printf("❌ GetOrganization: ошибка получения организации org_id=%s: %v", orgID, err)

		// Маппинг ошибок
		if errors.Is(err, domain.ErrOrganizationNotFound) {
			return &pb.GetOrganizationResponse{
				Success:      false,
				ErrorMessage: "организация не найдена",
			}, nil
		}

		return &pb.GetOrganizationResponse{
			Success:      false,
			ErrorMessage: "не удалось получить информацию об организации",
		}, nil
	}

	// 4. Конвертация доменной модели в protobuf
	pbOrg := converter.ToProtoOrganization(org)

	log.Printf("✅ GetOrganization успешно: org_id=%s, name=%s", org.ID, org.Name)
	return &pb.GetOrganizationResponse{
		Success:      true,
		Organization: pbOrg,
	}, nil
}
