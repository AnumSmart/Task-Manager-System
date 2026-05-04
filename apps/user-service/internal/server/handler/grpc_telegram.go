package handler

import (
	pb "api/gen/go/user/v1"
	"context"
	"errors"
	"log"
	"user-service/internal/converter"
	"user-service/internal/domain"
	"user-service/internal/server/interceptors"
)

// LinkTelegram - привязка Telegram аккаунта к существующему пользователю
// Пользователь вводит свой email в боте, после чего происходит привязка
func (s *UserServerHandler) LinkTelegram(ctx context.Context, req *pb.LinkTelegramRequest) (*pb.LinkTelegramResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 LinkTelegram вызван: telegram_id=%d, email=%s", req.GetTelegramId(), req.GetEmail())

	// Валидация email
	if req.GetEmail() == "" {
		return &pb.LinkTelegramResponse{
			Success:      false,
			ErrorMessage: "email не может быть пустым",
		}, nil
	}

	// Валидация telegram_id
	if req.GetTelegramId() == 0 {
		return &pb.LinkTelegramResponse{
			Success:      false,
			ErrorMessage: "telegram_id не может быть 0",
		}, nil
	}

	// Обработка optional поля telegram_username
	telegramUsername := ""
	if req.TelegramUsername != nil {
		telegramUsername = *req.TelegramUsername
	}

	// Вызов сервисного слоя
	user, tokens, expiresIn, err := s.UserServerService.Telegram.LinkTelegramServ(
		ctx,
		req.GetEmail(),
		req.GetTelegramId(),
		telegramUsername,
	)

	if err != nil {
		log.Printf("❌ LinkTelegram ошибка: %v", err)
		return &pb.LinkTelegramResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Формируем успешный ответ
	return &pb.LinkTelegramResponse{
		Success:      true,
		Message:      "Telegram успешно привязан",
		User:         converter.ToProtoUser(user),
		JwtToken:     tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// GetUserByTelegram - поиск пользователя по Telegram ID
// Используется ботом для идентификации пользователя при каждом запросе
func (s *UserServerHandler) GetUserByTelegram(ctx context.Context, req *pb.GetUserByTelegramRequest) (*pb.GetUserResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 GetUserByTelegram вызван: telegram_id=%d", req.GetTelegramId())

	return &pb.GetUserResponse{}, nil
}

// GetMyProfile - получение своего профиля по Telegram ID
// Удобный метод для бота, чтобы не хранить user_id на клиенте
func (s *UserServerHandler) GetMyProfile(ctx context.Context, req *pb.GetMyProfileRequest) (*pb.GetUserResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 GetMyProfile вызван: telegram_id=%s", req.GetRequestId())

	// Извлечение user_id из контекста (добавлен интерсептором)
	userID, ok := ctx.Value(interceptors.ContextKeyUserID).(string)
	if !ok || userID == "" {
		return &pb.GetUserResponse{
			Success:      false,
			ErrorMessage: "не авторизован",
		}, nil
	}

	// Вызов сервисного слоя
	user, err := s.UserServerService.User.GetUserByID(ctx, userID)
	if err != nil {
		log.Printf("❌ GetMyProfile: ошибка получения пользователя: %v", err)
		return &pb.GetUserResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Конвертация в protobuf
	pbUser := converter.ToProtoUser(user)

	return &pb.GetUserResponse{
		Success: true,
		User:    pbUser,
	}, nil
}

// UpdateMyProfile - обновление своего профиля
func (s *UserServerHandler) UpdateMyProfile(ctx context.Context, req *pb.UpdateMyProfileRequest) (*pb.GetUserResponse, error) {
	// 1. Проверка контекста (graceful shutdown)
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 UpdateMyProfile вызван: telegram_id=%s", req.GetRequestId())

	// 2. Извлечение user_id из контекста (добавлен интерсептором)
	userID, ok := ctx.Value(interceptors.ContextKeyUserID).(string)
	if !ok || userID == "" {
		log.Printf("❌ UpdateMyProfile: user_id не найден в контексте")
		return &pb.GetUserResponse{
			Success:      false,
			ErrorMessage: "не авторизован",
		}, nil
	}

	// 3. Получаем full_name из запроса (если передан)
	var fullName *string
	if req.FullName != nil {
		fullName = req.FullName
	}

	// 4. Вызов сервисного слоя
	err := s.UserServerService.User.UpdateMyProfile(ctx, userID, fullName)
	if err != nil {
		log.Printf("❌ UpdateMyProfile: ошибка обновления пользователя user_id=%s: %v", userID, err)

		// Маппинг ошибок
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			return &pb.GetUserResponse{
				Success:      false,
				ErrorMessage: "пользователь не найден",
			}, nil
		case errors.Is(err, domain.ErrPermissionDenied):
			return &pb.GetUserResponse{
				Success:      false,
				ErrorMessage: "нет прав для обновления профиля",
			}, nil
		default:
			return &pb.GetUserResponse{
				Success:      false,
				ErrorMessage: "не удалось обновить профиль",
			}, nil
		}
	}

	// 5. После успешного обновления получаем актуальные данные пользователя
	updatedUser, err := s.UserServerService.User.GetUserByID(ctx, userID)
	if err != nil {
		log.Printf("❌ UpdateMyProfile: ошибка получения обновлённого пользователя: %v", err)
		return &pb.GetUserResponse{
			Success:      false,
			ErrorMessage: "профиль обновлён, но не удалось получить актуальные данные",
		}, nil
	}

	// 6. Конвертация в protobuf
	pbUser := converter.ToProtoUser(updatedUser)

	log.Printf("✅ UpdateMyProfile успешно: user_id=%s, full_name=%s", userID, updatedUser.FullName)
	return &pb.GetUserResponse{
		Success: true,
		User:    pbUser,
	}, nil
}

// Logout - - выход из системы (отзыв JWT токена)
func (s *UserServerHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	// 1. Извлекаем userID из контекста (из JWT)
	sessionID := ctx.Value("sessionID").(string)

	// 2. Вызываем сервис
	err := s.UserServerService.Telegram.LogOut(ctx, sessionID)
	if err != nil {
		return &pb.LogoutResponse{Success: false, ErrorMessage: err.Error()}, nil
	}

	return &pb.LogoutResponse{Success: true}, nil
}
