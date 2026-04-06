package handler

import (
	pb "api/gen/go/user/v1"
	"context"
	"log"
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

	return &pb.LinkTelegramResponse{}, nil
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

	log.Printf("📝 GetMyProfile вызван: telegram_id=%d", req.GetTelegramId())

	return &pb.GetUserResponse{}, nil
}
