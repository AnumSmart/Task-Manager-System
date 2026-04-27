package service

import (
	"context"
	"fmt"
	"pkg/auth"
	"user-service/internal/domain"
	"user-service/internal/server/repository"
)

// структура части сервисного слоя, которая отвечает за работу телеграмм
type TelegramLayer struct {
	TeleRepo repository.UserDBRepository
	Auth     auth.AuthInterface
}

// констркутор для части сервисного слоя (телеграмм)
// в конструктор передаём составной репозиторий (на будущее)
func NewTelegramLayer(repo *repository.UserServiceRepository, auth auth.AuthInterface) *TelegramLayer {
	return &TelegramLayer{
		TeleRepo: repo.DBRepo,
		Auth:     auth,
	}
}

// LinkTelegramServ привязывает Telegram ID и возвращает accessToken, refreshToken и время жизни
func (t *TelegramLayer) LinkTelegramServ(ctx context.Context, email string, telegramID int64, telegramUsername string) (*domain.User, string, string, int64, error) {
	// 1. Поиск пользователя по email
	user, err := t.TeleRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("пользователь с email %s не найден: %w", email, err)
	}

	// 2. Проверка активности пользователя
	if !user.IsActive() {
		return nil, "", "", 0, fmt.Errorf("пользователь %s не активен", email)
	}

	// 3. Проверка привязки Telegram
	if user.IsTelegramLinked() {
		if user.GetTelegramID() == telegramID {
			// Уже привязан к этому же Telegram — просто выдаём токены
			tokenPair, err := t.Auth.GenerateTokenPair(user.ID, string(user.Role), user.OrganizationID)
			if err != nil {
				return nil, "", "", 0, fmt.Errorf("не удалось сгенерировать JWT: %w", err)
			}
			return user, tokenPair.AccessToken, tokenPair.RefreshToken, tokenPair.ExpiresIn, nil
		}
		return nil, "", "", 0, fmt.Errorf("к этому email уже привязан другой Telegram аккаунт")
	}

	// 4. Привязываем Telegram (используем доменный метод)
	user.LinkTelegram(telegramID, telegramUsername)

	// 5. Сохраняем обновления в БД
	if err := t.TeleRepo.Update(ctx, user); err != nil {
		return nil, "", "", 0, fmt.Errorf("не удалось обновить пользователя: %w", err)
	}

	// 6. Генерируем пару JWT токенов
	tokenPair, err := t.Auth.GenerateTokenPair(user.ID, string(user.Role), user.OrganizationID)
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("не удалось сгенерировать JWT: %w", err)
	}

	return user, tokenPair.AccessToken, tokenPair.RefreshToken, tokenPair.ExpiresIn, nil
}
