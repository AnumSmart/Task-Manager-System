package handler

import (
	pb "api/gen/go/user/v1"
	"context"
	"log"
	"user-service/internal/converter"
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
	user, accessToken, refreshToken, expiresIn, err := s.UserServerService.Telegram.LinkTelegramServ(
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
		JwtToken:     accessToken,
		RefreshToken: refreshToken,
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

	return &pb.GetUserResponse{}, nil
}

// UpdateMyProfile - обновление своего профиля
func (s *UserServerHandler) UpdateMyProfile(ctx context.Context, req *pb.UpdateMyProfileRequest) (*pb.GetUserResponse, error) {
	select {
	case <-ctx.Done():
		log.Printf("❌ Контекст отменён: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	log.Printf("📝 UpdateMyProfile вызван: telegram_id=%s", req.GetRequestId())

	return &pb.GetUserResponse{}, nil
}
