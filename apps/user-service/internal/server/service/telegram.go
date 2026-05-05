package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"pkg/auth"
	"pkg/auth/jwt"
	"time"
	"user-service/internal/domain"
	"user-service/internal/server/repository"
)

// структура части сервисного слоя, которая отвечает за работу телеграмм
type TelegramLayer struct {
	TeleRepo repository.UserDBRepository
	Cache    repository.UserCacheRepository
	Auth     auth.AuthInterface
}

// констркутор для части сервисного слоя (телеграмм)
// в конструктор передаём составной репозиторий (на будущее)
func NewTelegramLayer(repo *repository.UserServiceRepository, auth auth.AuthInterface) *TelegramLayer {
	return &TelegramLayer{
		TeleRepo: repo.DBRepo,
		Cache:    repo.CacheRepo,
		Auth:     auth,
	}
}

// getCacheKey - генерация ключа для кэша по ID пользователя
func (t *TelegramLayer) getCacheKey(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

// getTelegramCacheKey - генерация ключа для кэша по Telegram ID
func (t *TelegramLayer) getTelegramCacheKey(telegramID int64) string {
	return fmt.Sprintf("user:telegram:%d", telegramID)
}

// cacheUser - сохранение пользователя в кэш
func (t *TelegramLayer) cacheUser(ctx context.Context, user *domain.User) error {
	key := t.getCacheKey(user.ID)
	if err := t.Cache.Set(ctx, key, user, 3600); err != nil {
		log.Printf("⚠️ Failed to cache user %s: %v", user.ID, err)
		return err
	}
	log.Printf("✅ User cached: %s", user.ID)
	return nil
}

// cacheUserByTelegram - сохранение пользователя в кэш по Telegram ID
func (t *TelegramLayer) cacheUserByTelegram(ctx context.Context, user *domain.User) error {
	if !user.IsTelegramLinked() {
		return nil
	}

	telegramID := user.GetTelegramID()
	key := t.getTelegramCacheKey(telegramID)

	if err := t.Cache.Set(ctx, key, user, 3600); err != nil {
		log.Printf("⚠️ Failed to cache user by telegram_id %d: %v", telegramID, err)
		return err
	}
	log.Printf("✅ User cached by telegram_id: %d -> user_id: %s", telegramID, user.ID)
	return nil
}

// invalidateUserCache - инвалидация кэша пользователя
func (t *TelegramLayer) invalidateUserCache(ctx context.Context, userID string) error {
	key := t.getCacheKey(userID)
	if err := t.Cache.Delete(ctx, key); err != nil {
		log.Printf("⚠️ Failed to invalidate cache for user %s: %v", userID, err)
		return err
	}
	log.Printf("🗑️ User cache invalidated: %s", userID)
	return nil
}

// invalidateTelegramCache - инвалидация кэша по Telegram ID
func (t *TelegramLayer) invalidateTelegramCache(ctx context.Context, telegramID int64) error {
	key := t.getTelegramCacheKey(telegramID)
	if err := t.Cache.Delete(ctx, key); err != nil {
		log.Printf("⚠️ Failed to invalidate telegram cache for %d: %v", telegramID, err)
		return err
	}
	log.Printf("🗑️ Telegram cache invalidated: %d", telegramID)
	return nil
}

// LinkTelegramServ привязывает Telegram ID и возвращает accessToken, refreshToken и время жизни
func (t *TelegramLayer) LinkTelegramServ(ctx context.Context, email string, telegramID int64, telegramUsername string) (*domain.User, *jwt.TokenPair, int64, error) {
	// 1. Поиск пользователя по email
	user, err := t.TeleRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("пользователь с email %s не найден: %w", email, err)
	}

	// 2. Проверка активности пользователя
	if !user.IsActive() {
		return nil, nil, 0, fmt.Errorf("пользователь %s не активен", email)
	}

	// 3. Проверка привязки Telegram
	if user.IsTelegramLinked() {
		if user.GetTelegramID() == telegramID {
			// Уже привязан к этому же Telegram — просто выдаём токены
			tokenPair, _, err := t.Auth.GenerateTokenPair(ctx, user.ID, string(user.Role), user.OrganizationID)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("не удалось сгенерировать JWT: %w", err)
			}
			return user, tokenPair, tokenPair.ExpiresAt, nil
		}
		return nil, nil, 0, fmt.Errorf("к этому email уже привязан другой Telegram аккаунт")
	}

	// 4. Привязываем Telegram (используем доменный метод)
	user.LinkTelegram(telegramID, telegramUsername)

	// 5. Сохраняем обновления в БД
	if err := t.TeleRepo.Update(ctx, user); err != nil {
		return nil, nil, 0, fmt.Errorf("не удалось обновить пользователя: %w", err)
	}

	// 🔥 6. Обновляем кэш (оба ключа)
	if err := t.cacheUser(ctx, user); err != nil {
		log.Printf("⚠️ Failed to update user cache: %v", err)
	}

	if err := t.cacheUserByTelegram(ctx, user); err != nil {
		log.Printf("⚠️ Failed to update telegram cache: %v", err)
	}

	// 7. Генерируем пару JWT токенов
	tokenPair, _, err := t.Auth.GenerateTokenPair(ctx, user.ID, string(user.Role), user.OrganizationID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("не удалось сгенерировать JWT: %w", err)
	}

	return user, tokenPair, tokenPair.ExpiresAt, nil
}

// UnlinkTelegramServ - отвязка Telegram от пользователя
func (t *TelegramLayer) UnlinkTelegramServ(ctx context.Context, userID string) error {
	// 1. Получаем пользователя
	user, err := t.TeleRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("пользователь не найден: %w", err)
	}

	// 2. Проверяем, привязан ли Telegram
	if !user.IsTelegramLinked() {
		return fmt.Errorf("telegram не привязан к этому аккаунту")
	}

	oldTelegramID := user.GetTelegramID()

	// 3. Отвязываем Telegram
	user.UnlinkTelegram()

	// 4. Сохраняем в БД
	if err := t.TeleRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("не удалось обновить пользователя: %w", err)
	}

	// 🔥 5. Инвалидируем кэш
	if err := t.invalidateTelegramCache(ctx, oldTelegramID); err != nil {
		log.Printf("⚠️ Failed to invalidate telegram cache: %v", err)
	}

	if err := t.cacheUser(ctx, user); err != nil {
		log.Printf("⚠️ Failed to update user cache: %v", err)
	}

	log.Printf("✅ Telegram unlinked: user_id=%s, old_telegram_id=%d", userID, oldTelegramID)

	return nil
}

// метод сервисного слоя LogOut
func (t *TelegramLayer) LogOut(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}

	err := t.Auth.Logout(ctx, sessionID)
	if err != nil {
		return err
	}
	return nil
}

// метод получения пользователя на основе телеграмм ID
func (t *TelegramLayer) GetUserByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	startTime := time.Now()

	log.Printf("🔍 GetUserByTelegramID: searching for telegram_id=%d", telegramID)

	// 1. Валидация
	if telegramID == 0 {
		return nil, domain.ErrInvalidInput
	}

	// 2. Пробуем получить из кэша
	cacheKey := t.getTelegramCacheKey(telegramID)

	// создаём переменную-указатель
	var user *domain.User

	if err := t.Cache.Get(ctx, cacheKey, user); err == nil {
		log.Printf("✅ GetUserByTelegramID: cache HIT for telegram_id=%d", telegramID)
		return user, nil
	}

	log.Printf("📦 GetUserByTelegramID: cache MISS for telegram_id=%d", telegramID)

	// 3. Поиск в БД
	dbStartTime := time.Now()
	user, err := t.TeleRepo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			log.Printf("❌ GetUserByTelegramID: user with telegram_id=%d not found in DB", telegramID)
			return nil, domain.ErrUserNotFound
		}
		log.Printf("❌ GetUserByTelegramID: database error: %v", err)
		return nil, fmt.Errorf("database error: %w", err)
	}

	log.Printf("✅ GetUserByTelegramID: database query completed in %v, found user_id=%s",
		time.Since(dbStartTime), user.ID)

	// 4. Сохраняем в кэш (оба ключа)
	if err := t.cacheUserByTelegram(ctx, user); err != nil {
		log.Printf("⚠️ GetUserByTelegramID: failed to cache by telegram: %v", err)
	}

	if err := t.cacheUser(ctx, user); err != nil {
		log.Printf("⚠️ GetUserByTelegramID: failed to cache by user_id: %v", err)
	}

	log.Printf("🎉 GetUserByTelegramID: completed in %v", time.Since(startTime))

	return user, nil
}
