package jwt

import (
	"github.com/golang-jwt/jwt/v5"
)

// TokenType определяет тип токена
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// CustomClaims содержит пользовательские данные в JWT
type CustomClaims struct {
	UserID         string `json:"user_id"`
	Role           string `json:"role"`
	OrganizationID string `json:"organization_id"`
	jwt.RegisteredClaims
}

// TokenPair содержит пару access и refresh токенов
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"` // seconds
}

// Validate проверяет обязательные поля в claims
func (c *CustomClaims) Validate() error {
	if c.UserID == "" {
		return ErrMissingUserID
	}
	if c.Role == "" {
		return ErrMissingRole
	}
	if c.OrganizationID == "" {
		return ErrMissingOrganizationID
	}
	return nil
}

// GetTokenType возвращает тип токена из audience
func (c *CustomClaims) GetTokenType() TokenType {
	if len(c.Audience) == 0 {
		return ""
	}
	return TokenType(c.Audience[0])
}

// IsAccessToken проверяет, является ли токен access токеном
func (c *CustomClaims) IsAccessToken() bool {
	return c.GetTokenType() == AccessToken
}

// IsRefreshToken проверяет, является ли токен refresh токеном
func (c *CustomClaims) IsRefreshToken() bool {
	return c.GetTokenType() == RefreshToken
}
