package configs

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// JWTConfig хранит настройки JWT авторизации
type JWTConfig struct {
	// SecretKey - секретный ключ для подписи токенов (мин. 32 символа)
	SecretKey string

	// AccessTokenTTL - время жизни access токена
	AccessTokenTTL time.Duration

	// RefreshTokenTTL - время жизни refresh токена
	RefreshTokenTTL time.Duration

	// Issuer - издатель токена
	Issuer string
}

// LoadJWTConfig загружает JWT конфигурацию из переменных окружения
func LoadJWTConfig() (*JWTConfig, error) {
	cfg := &JWTConfig{
		Issuer: "task-management-system", // значение по умолчанию
	}

	// SecretKey (обязательный)
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(secretKey) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters, got %d", len(secretKey))
	}
	cfg.SecretKey = secretKey

	// AccessTokenTTL (часы)
	accessTTL := os.Getenv("JWT_ACCESS_TTL_HOURS")
	if accessTTL == "" {
		accessTTL = "24"
	}
	hours, err := strconv.Atoi(accessTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL_HOURS: %w", err)
	}
	cfg.AccessTokenTTL = time.Duration(hours) * time.Hour

	// RefreshTokenTTL (дни)
	refreshTTL := os.Getenv("JWT_REFRESH_TTL_DAYS")
	if refreshTTL == "" {
		refreshTTL = "7"
	}
	days, err := strconv.Atoi(refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL_DAYS: %w", err)
	}
	cfg.RefreshTokenTTL = time.Duration(days) * 24 * time.Hour

	// Issuer (опционально)
	if issuer := os.Getenv("JWT_ISSUER"); issuer != "" {
		cfg.Issuer = issuer
	}

	return cfg, nil
}

// MustLoadJWTConfig загружает конфигурацию или паникует при ошибке
func MustLoadJWTConfig() *JWTConfig {
	cfg, err := LoadJWTConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load JWT config: %v", err))
	}
	return cfg
}

// Validate проверяет корректность конфигурации
func (c *JWTConfig) Validate() error {
	if c.SecretKey == "" {
		return fmt.Errorf("JWT secret key is empty")
	}
	if len(c.SecretKey) < 32 {
		return fmt.Errorf("JWT secret key is too weak (min 32 characters)")
	}
	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("access token TTL must be positive")
	}
	if c.RefreshTokenTTL <= 0 {
		return fmt.Errorf("refresh token TTL must be positive")
	}
	return nil
}
