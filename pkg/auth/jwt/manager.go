package jwt

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"pkg/configs" // используем общий пакет конфигурации
)

// Manager управляет JWT операциями
type Manager struct {
	config *configs.JWTConfig
}

// NewManager создает новый JWT менеджер
func NewManager(cfg *configs.JWTConfig) (*Manager, error) {
	if cfg == nil {
		return nil, ErrInvalidConfig
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &Manager{
		config: cfg,
	}, nil
}

// GenerateTokenPair создает пару access и refresh токенов и sessionID, одинаковый для обоих токенов
func (m *Manager) GenerateTokenPair(userID, role, organizationID string) (*TokenPair, string, error) {
	// Генерируем ОДИН sessionID для всей пары токенов
	sessionID := uuid.New().String() // ← общий идентификатор сессии

	// Генерируем access token
	accessToken, err := m.generateToken(userID, role, organizationID, m.config.AccessTokenTTL, AccessToken, sessionID)
	if err != nil {
		return nil, "", fmt.Errorf("generate access token: %w", err)
	}

	// Генерируем refresh token
	refreshToken, err := m.generateToken(userID, role, organizationID, m.config.RefreshTokenTTL, RefreshToken, sessionID)
	if err != nil {
		return nil, "", fmt.Errorf("generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    int64(m.config.AccessTokenTTL.Seconds()),
	}, sessionID, nil
}

// generateToken создает один JWT токен
func (m *Manager) generateToken(userID, role, organizationID string, ttl time.Duration, tokenType TokenType, sessionID string) (string, error) {
	// Отладка: проверяем, что sessionID не пустой
	if sessionID == "" {
		return "", fmt.Errorf("sessionID is empty, cannot generate token")
	}

	log.Printf("🔐 Generating %s token with sessionID: %s for user: %s", tokenType, sessionID, userID)

	now := time.Now()
	claims := &CustomClaims{
		UserID:         userID,
		Role:           role,
		OrganizationID: organizationID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        sessionID, // ← используем переданный sessionID, НЕ новый uuid
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.config.Issuer,
			Subject:   userID,
			Audience:  []string{string(tokenType)},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(m.config.SecretKey))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken проверяет и парсит JWT токен
func (m *Manager) ValidateToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.SecretKey), nil
	})

	if err != nil {
		// Возвращаем кастомные ошибки для каждого случая
		switch {
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, ErrMalformedToken
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, ErrInvalidSignature
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrExpiredToken
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, ErrTokenNotValidYet
		case errors.Is(err, ErrUnexpectedSigningMethod):
			return nil, ErrUnexpectedSigningMethod
		default:
			// Для любых других ошибок парсинга (например, проблемы с claims)
			return nil, ErrParseToken
		}
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if err := claims.Validate(); err != nil {
		return nil, err
	}

	return claims, nil
}

// RefreshAccessToken создает новый access token из refresh token
func (m *Manager) RefreshAccessToken(refreshTokenString string) (*TokenPair, string, error) {
	claims, err := m.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, "", fmt.Errorf("validate refresh token: %w", err)
	}

	if !claims.IsRefreshToken() {
		return nil, "", ErrNotRefreshToken
	}

	// генерируем новую пару токенов с новым SessionID
	tokenPair, sessionID, err := m.GenerateTokenPair(claims.UserID, claims.Role, claims.OrganizationID)
	if err != nil {
		return nil, "", err
	}

	return tokenPair, sessionID, nil
}

// ExtractTokenFromBearer извлекает токен из строки "Bearer <token>"
func (m *Manager) ExtractTokenFromBearer(authHeader string) (string, error) {
	if authHeader == "" {
		return "", ErrMissingToken
	}

	const prefix = "Bearer "
	if len(authHeader) < len(prefix) || authHeader[:len(prefix)] != prefix {
		return "", ErrInvalidAuthHeader
	}

	token := authHeader[len(prefix):]
	if token == "" {
		return "", ErrEmptyToken
	}

	return token, nil
}
