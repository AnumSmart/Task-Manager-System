package jwt

import (
	"pkg/configs"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestNewManager тестирует создание менеджера
func TestNewManager(t *testing.T) {
	tests := []struct {
		name        string
		config      *configs.JWTConfig
		expectedErr error
	}{
		{
			name: "successful creation",
			config: &configs.JWTConfig{
				SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 7 * 24 * time.Hour,
				Issuer:          "test-system",
			},
			expectedErr: nil,
		},
		{
			name:        "nil config",
			config:      nil,
			expectedErr: ErrInvalidConfig,
		},
		{
			name: "invalid config - weak secret",
			config: &configs.JWTConfig{
				SecretKey:       "weak",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 7 * 24 * time.Hour,
				Issuer:          "test-system",
			},
			expectedErr: configs.ErrWeakSecretKey,
		},
		{
			name: "invalid config - zero access TTL",
			config: &configs.JWTConfig{
				SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
				AccessTokenTTL:  0,
				RefreshTokenTTL: 7 * 24 * time.Hour,
				Issuer:          "test-system",
			},
			expectedErr: configs.ErrInvalidAccessTTL,
		},
		{
			name: "invalid config - zero refresh TTL",
			config: &configs.JWTConfig{
				SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 0,
				Issuer:          "test-system",
			},
			expectedErr: configs.ErrInvalidRefreshTTL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config)

			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if err != tt.expectedErr {
					t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
				}
				if manager != nil {
					t.Errorf("Expected nil manager, got %v", manager)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if manager == nil {
					t.Errorf("Expected manager, got nil")
				}
				if manager.config != tt.config {
					t.Errorf("Config not set correctly")
				}
			}
		})
	}
}

// TestGenerateTokenPair тестирует генерацию пары токенов
func TestGenerateTokenPair(t *testing.T) {
	config := &configs.JWTConfig{
		SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
		AccessTokenTTL:  24 * time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "test-system",
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name           string
		userID         string
		role           string
		organizationID string
		shouldFail     bool
	}{
		{
			name:           "valid user",
			userID:         "user-123",
			role:           "admin",
			organizationID: "org-456",
			shouldFail:     false,
		},
		{
			name:           "empty user ID",
			userID:         "",
			role:           "user",
			organizationID: "org-789",
			shouldFail:     false, // Генерация должна пройти, но валидация позже может упасть
		},
		{
			name:           "empty role",
			userID:         "user-456",
			role:           "",
			organizationID: "org-789",
			shouldFail:     false,
		},
		{
			name:           "empty organization",
			userID:         "user-789",
			role:           "user",
			organizationID: "",
			shouldFail:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenPair, sessionID, err := manager.GenerateTokenPair(tt.userID, tt.role, tt.organizationID)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tokenPair == nil {
				t.Errorf("Expected token pair, got nil")
				return
			}

			if tokenPair.AccessToken == "" {
				t.Errorf("Access token is empty")
			}

			if tokenPair.RefreshToken == "" {
				t.Errorf("Refresh token is empty")
			}

			if tokenPair.ExpiresAt == 0 {
				t.Errorf("ExpiresAt is zero")
			}

			if sessionID == "" {
				t.Errorf("Session ID is empty")
			}

			// Проверяем, что sessionID это валидный UUID
			if _, err := uuid.Parse(sessionID); err != nil {
				t.Errorf("Session ID is not a valid UUID: %v", err)
			}

			// Проверяем, что access и refresh токены разные
			if tokenPair.AccessToken == tokenPair.RefreshToken {
				t.Errorf("Access and refresh tokens are the same")
			}
		})
	}
}

// TestValidateToken тестирует валидацию токенов
func TestValidateToken(t *testing.T) {
	config := &configs.JWTConfig{
		SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "test-system",
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Генерируем валидную пару токенов
	userID := "user-123"
	role := "admin"
	orgID := "org-456"
	tokenPair, sessionID, err := manager.GenerateTokenPair(userID, role, orgID)
	if err != nil {
		t.Fatalf("Failed to generate tokens: %v", err)
	}

	tests := []struct {
		name        string
		tokenString string
		expectedErr error
		validateFn  func(*testing.T, *CustomClaims)
	}{
		{
			name:        "valid access token",
			tokenString: tokenPair.AccessToken,
			expectedErr: nil,
			validateFn: func(t *testing.T, claims *CustomClaims) {
				if claims.UserID != userID {
					t.Errorf("UserID = %s, want %s", claims.UserID, userID)
				}
				if claims.Role != role {
					t.Errorf("Role = %s, want %s", claims.Role, role)
				}
				if claims.OrganizationID != orgID {
					t.Errorf("OrganizationID = %s, want %s", claims.OrganizationID, orgID)
				}
				if claims.Issuer != config.Issuer {
					t.Errorf("Issuer = %s, want %s", claims.Issuer, config.Issuer)
				}
				if claims.Subject != userID {
					t.Errorf("Subject = %s, want %s", claims.Subject, userID)
				}
				if claims.ID != sessionID {
					t.Errorf("Session ID = %s, want %s", claims.ID, sessionID)
				}
				if claims.GetTokenType() != AccessToken {
					t.Errorf("Token type = %s, want %s", claims.GetTokenType(), AccessToken)
				}
			},
		},
		{
			name:        "valid refresh token",
			tokenString: tokenPair.RefreshToken,
			expectedErr: nil,
			validateFn: func(t *testing.T, claims *CustomClaims) {
				if claims.GetTokenType() != RefreshToken {
					t.Errorf("Token type = %s, want %s", claims.GetTokenType(), RefreshToken)
				}
			},
		},
		{
			name:        "empty token",
			tokenString: "",
			expectedErr: ErrMalformedToken,
			validateFn:  nil,
		},
		{
			name:        "malformed token",
			tokenString: "invalid.token.string",
			expectedErr: ErrMalformedToken,
			validateFn:  nil,
		},
		{
			name:        "invalid signature",
			tokenString: tokenPair.AccessToken + "invalid",
			expectedErr: ErrInvalidSignature,
			validateFn:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateToken(tt.tokenString)

			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if err != tt.expectedErr {
					t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
				}
				if claims != nil {
					t.Errorf("Expected nil claims, got %v", claims)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if claims == nil {
				t.Errorf("Expected claims, got nil")
				return
			}

			if tt.validateFn != nil {
				tt.validateFn(t, claims)
			}
		})
	}
}

// TestExtractTokenFromBearer тестирует извлечение токена из Bearer заголовка
func TestExtractTokenFromBearer(t *testing.T) {
	config := &configs.JWTConfig{
		SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
		AccessTokenTTL:  24 * time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "test-system",
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	validToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	tests := []struct {
		name        string
		authHeader  string
		expected    string
		expectedErr error
	}{
		{
			name:        "valid Bearer token",
			authHeader:  "Bearer " + validToken,
			expected:    validToken,
			expectedErr: nil,
		},
		{
			name:        "empty header",
			authHeader:  "",
			expected:    "",
			expectedErr: ErrMissingToken,
		},
		{
			name:        "missing Bearer prefix",
			authHeader:  validToken,
			expected:    "",
			expectedErr: ErrInvalidAuthHeader,
		},
		{
			name:        "malformed Bearer - no space",
			authHeader:  "Bearer" + validToken,
			expected:    "",
			expectedErr: ErrInvalidAuthHeader,
		},
		{
			name:        "empty token after Bearer",
			authHeader:  "Bearer ",
			expected:    "",
			expectedErr: ErrEmptyToken,
		},
		{
			name:        "lowercase bearer",
			authHeader:  "bearer " + validToken,
			expected:    "",
			expectedErr: ErrInvalidAuthHeader,
		},
		{
			name:        "only Bearer prefix without token",
			authHeader:  "Bearer",
			expected:    "",
			expectedErr: ErrInvalidAuthHeader,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.ExtractTokenFromBearer(tt.authHeader)

			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if err != tt.expectedErr {
					t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if token != tt.expected {
				t.Errorf("Extracted token = %s, want %s", token, tt.expected)
			}
		})
	}
}

// TestExpiredToken тестирует поведение с просроченными токенами
func TestExpiredToken(t *testing.T) {
	// Создаем конфиг с очень маленьким TTL
	config := &configs.JWTConfig{
		SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
		AccessTokenTTL:  1 * time.Second, // Истекает через 1 секунду
		RefreshTokenTTL: 2 * time.Second, // Истекает через 2 секунды
		Issuer:          "test-system",
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Генерируем токены
	tokenPair, _, err := manager.GenerateTokenPair("user-123", "admin", "org-456")
	if err != nil {
		t.Fatalf("Failed to generate tokens: %v", err)
	}

	// Ждем истечения токенов
	time.Sleep(3 * time.Second)

	tests := []struct {
		name        string
		tokenString string
		expectedErr error
	}{
		{
			name:        "expired access token",
			tokenString: tokenPair.AccessToken,
			expectedErr: ErrExpiredToken,
		},
		{
			name:        "expired refresh token",
			tokenString: tokenPair.RefreshToken,
			expectedErr: ErrExpiredToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.ValidateToken(tt.tokenString)
			if err != tt.expectedErr {
				t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

// TestSameSessionIDInTokens тестирует, что access и refresh токены имеют одинаковый sessionID
func TestSameSessionIDInTokens(t *testing.T) {
	config := &configs.JWTConfig{
		SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
		AccessTokenTTL:  24 * time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "test-system",
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tokenPair, sessionID, err := manager.GenerateTokenPair("user-123", "admin", "org-456")
	if err != nil {
		t.Fatalf("Failed to generate tokens: %v", err)
	}

	// Проверяем claims access токена
	accessClaims, err := manager.ValidateToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate access token: %v", err)
	}

	// Проверяем claims refresh токена
	refreshClaims, err := manager.ValidateToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to validate refresh token: %v", err)
	}

	// Проверяем, что sessionID одинаковый
	if accessClaims.ID != refreshClaims.ID {
		t.Errorf("Access token session ID (%s) != Refresh token session ID (%s)",
			accessClaims.ID, refreshClaims.ID)
	}

	// Проверяем, что sessionID соответствует возвращенному
	if accessClaims.ID != sessionID {
		t.Errorf("Access token session ID (%s) != returned session ID (%s)",
			accessClaims.ID, sessionID)
	}

	if refreshClaims.ID != sessionID {
		t.Errorf("Refresh token session ID (%s) != returned session ID (%s)",
			refreshClaims.ID, sessionID)
	}
}

// TestTokenType тестирует правильность установки типа токена
func TestTokenType(t *testing.T) {
	config := &configs.JWTConfig{
		SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
		AccessTokenTTL:  24 * time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "test-system",
	}

	manager, err := NewManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tokenPair, _, err := manager.GenerateTokenPair("user-123", "admin", "org-456")
	if err != nil {
		t.Fatalf("Failed to generate tokens: %v", err)
	}

	tests := []struct {
		name        string
		tokenString string
		expected    TokenType
	}{
		{
			name:        "access token type",
			tokenString: tokenPair.AccessToken,
			expected:    AccessToken,
		},
		{
			name:        "refresh token type",
			tokenString: tokenPair.RefreshToken,
			expected:    RefreshToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateToken(tt.tokenString)
			if err != nil {
				t.Fatalf("Failed to validate token: %v", err)
			}

			if claims.GetTokenType() != tt.expected {
				t.Errorf("Token type = %s, want %s", claims.GetTokenType(), tt.expected)
			}
		})
	}
}
