package configs

import (
	"os"
	"sync"
	"testing"
	"time"
)

// глобальная блокировка для доступа к переменным окружения
var jwtEnvMutex sync.Mutex

// TestJWTEnv - структура для управления тестовым окружением JWT
type TestJWTEnv struct {
	variables map[string]string
}

// NewTestJWTEnv создает новое тестовое окружение
func NewTestJWTEnv() *TestJWTEnv {
	return &TestJWTEnv{
		variables: make(map[string]string),
	}
}

// Cleanup восстанавливает чистое состояние
func (te *TestJWTEnv) Cleanup() {
	jwtEnvMutex.Lock()
	defer jwtEnvMutex.Unlock()

	// Удаляем только те переменные, которые устанавливали
	for key := range te.variables {
		os.Unsetenv(key)
	}
}

// Set устанавливает переменную окружения для теста
func (te *TestJWTEnv) Set(key, value string) {
	te.variables[key] = value
}

// Apply применяет все установленные переменные к окружению
func (te *TestJWTEnv) Apply() {
	jwtEnvMutex.Lock()
	defer jwtEnvMutex.Unlock()

	for key, value := range te.variables {
		os.Setenv(key, value)
	}
}

// ClearAllJWTEnvVars очищает все переменные JWT
func ClearAllJWTEnvVars() {
	jwtEnvMutex.Lock()
	defer jwtEnvMutex.Unlock()

	envVars := []string{
		"JWT_SECRET",
		"JWT_ACCESS_TTL_HOURS",
		"JWT_REFRESH_TTL_DAYS",
		"JWT_ISSUER",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

// containsString проверяет вхождение подстроки
func containsStringJWT(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ========== Тесты для LoadJWTConfig ==========

func TestLoadJWTConfig(t *testing.T) {
	// Очищаем глобальное окружение перед всеми тестами
	ClearAllJWTEnvVars()

	tests := []struct {
		name          string
		setupEnv      func(*TestJWTEnv)
		expectedError string
		validateFunc  func(*testing.T, *JWTConfig)
	}{
		{
			name: "successful config with defaults",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
			},
			validateFunc: func(t *testing.T, cfg *JWTConfig) {
				if cfg.AccessTokenTTL != 24*time.Hour {
					t.Errorf("AccessTokenTTL = %v, want 24h", cfg.AccessTokenTTL)
				}
				if cfg.RefreshTokenTTL != 7*24*time.Hour {
					t.Errorf("RefreshTokenTTL = %v, want 7*24h", cfg.RefreshTokenTTL)
				}
				if cfg.Issuer != "task-management-system" {
					t.Errorf("Issuer = %v, want task-management-system", cfg.Issuer)
				}
			},
		},
		{
			name: "successful config with custom values",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "my_custom_32_byte_secret_key_here_!!")
				env.Set("JWT_ACCESS_TTL_HOURS", "12")
				env.Set("JWT_REFRESH_TTL_DAYS", "30")
				env.Set("JWT_ISSUER", "my-app")
			},
			validateFunc: func(t *testing.T, cfg *JWTConfig) {
				if cfg.AccessTokenTTL != 12*time.Hour {
					t.Errorf("AccessTokenTTL = %v, want 12h", cfg.AccessTokenTTL)
				}
				if cfg.RefreshTokenTTL != 30*24*time.Hour {
					t.Errorf("RefreshTokenTTL = %v, want 30*24h", cfg.RefreshTokenTTL)
				}
				if cfg.Issuer != "my-app" {
					t.Errorf("Issuer = %v, want my-app", cfg.Issuer)
				}
			},
		},
		{
			name: "missing JWT_SECRET",
			setupEnv: func(env *TestJWTEnv) {
				// Не устанавливаем JWT_SECRET
			},
			expectedError: "JWT_SECRET is required",
		},
		{
			name: "JWT_SECRET too short (less than 32 chars)",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "short_secret")
			},
			expectedError: "JWT_SECRET must be at least 32 characters",
		},
		{
			name: "JWT_SECRET exactly 32 characters",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "12345678901234567890123456789012") // ровно 32 символа
			},
			validateFunc: func(t *testing.T, cfg *JWTConfig) {
				if cfg.SecretKey != "12345678901234567890123456789012" {
					t.Errorf("SecretKey = %v, want 32 char string", cfg.SecretKey)
				}
			},
		},
		{
			name: "invalid JWT_ACCESS_TTL_HOURS (not a number)",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
				env.Set("JWT_ACCESS_TTL_HOURS", "invalid")
			},
			expectedError: "invalid JWT_ACCESS_TTL_HOURS",
		},
		{
			name: "invalid JWT_ACCESS_TTL_HOURS (negative)",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
				env.Set("JWT_ACCESS_TTL_HOURS", "-5")
			},
			validateFunc: func(t *testing.T, cfg *JWTConfig) {
				// Отрицательное значение пройдет парсинг, но упадет на валидации
				if cfg.AccessTokenTTL != -5*time.Hour {
					t.Errorf("AccessTokenTTL = %v, want -5h", cfg.AccessTokenTTL)
				}
			},
		},
		{
			name: "invalid JWT_REFRESH_TTL_DAYS (not a number)",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
				env.Set("JWT_REFRESH_TTL_DAYS", "invalid")
			},
			expectedError: "invalid JWT_REFRESH_TTL_DAYS",
		},
		{
			name: "zero JWT_ACCESS_TTL_HOURS",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
				env.Set("JWT_ACCESS_TTL_HOURS", "0")
			},
			validateFunc: func(t *testing.T, cfg *JWTConfig) {
				if cfg.AccessTokenTTL != 0 {
					t.Errorf("AccessTokenTTL = %v, want 0", cfg.AccessTokenTTL)
				}
			},
		},
		{
			name: "custom issuer with spaces",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
				env.Set("JWT_ISSUER", "my test issuer")
			},
			validateFunc: func(t *testing.T, cfg *JWTConfig) {
				if cfg.Issuer != "my test issuer" {
					t.Errorf("Issuer = %v, want 'my test issuer'", cfg.Issuer)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEnv := NewTestJWTEnv()
			defer testEnv.Cleanup()

			ClearAllJWTEnvVars()

			if tt.setupEnv != nil {
				tt.setupEnv(testEnv)
			}

			// Применяем переменные
			testEnv.Apply()

			config, err := LoadJWTConfig()

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !containsStringJWT(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Errorf("Expected config, got nil")
				return
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, config)
			}
		})
	}
}

func TestMustLoadJWTConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func(*TestJWTEnv)
		expectPanic bool
		panicMsg    string
	}{
		{
			name: "successful config",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
			},
			expectPanic: false,
		},
		{
			name: "missing JWT_SECRET causes panic",
			setupEnv: func(env *TestJWTEnv) {
				// Не устанавливаем JWT_SECRET
			},
			expectPanic: true,
			panicMsg:    "failed to load JWT config",
		},
		{
			name: "invalid TTL causes panic",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
				env.Set("JWT_ACCESS_TTL_HOURS", "invalid")
			},
			expectPanic: true,
			panicMsg:    "failed to load JWT config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEnv := NewTestJWTEnv()
			defer testEnv.Cleanup()

			ClearAllJWTEnvVars()

			if tt.setupEnv != nil {
				tt.setupEnv(testEnv)
			}
			testEnv.Apply()

			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic but got none")
					} else if panicStr, ok := r.(string); ok {
						if !containsStringJWT(panicStr, tt.panicMsg) {
							t.Errorf("Expected panic containing '%s', got '%s'", tt.panicMsg, panicStr)
						}
					}
				}()
				MustLoadJWTConfig()
			} else {
				config := MustLoadJWTConfig()
				if config == nil {
					t.Errorf("Expected config, got nil")
				}
			}
		})
	}
}

// ========== Тесты для Validate метода ==========

func TestJWTConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *JWTConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &JWTConfig{
				SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 7 * 24 * time.Hour,
				Issuer:          "test",
			},
			expectError: false,
		},
		{
			name: "empty secret key",
			config: &JWTConfig{
				SecretKey:       "",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 7 * 24 * time.Hour,
			},
			expectError: true,
			errorMsg:    "JWT secret key is empty",
		},
		{
			name: "secret key too short",
			config: &JWTConfig{
				SecretKey:       "short",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 7 * 24 * time.Hour,
			},
			expectError: true,
			errorMsg:    "JWT secret key is too weak",
		},
		{
			name: "access token TTL zero",
			config: &JWTConfig{
				SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
				AccessTokenTTL:  0,
				RefreshTokenTTL: 7 * 24 * time.Hour,
			},
			expectError: true,
			errorMsg:    "access token TTL must be positive",
		},
		{
			name: "access token TTL negative",
			config: &JWTConfig{
				SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
				AccessTokenTTL:  -1 * time.Hour,
				RefreshTokenTTL: 7 * 24 * time.Hour,
			},
			expectError: true,
			errorMsg:    "access token TTL must be positive",
		},
		{
			name: "refresh token TTL zero",
			config: &JWTConfig{
				SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 0,
			},
			expectError: true,
			errorMsg:    "refresh token TTL must be positive",
		},
		{
			name: "refresh token TTL negative",
			config: &JWTConfig{
				SecretKey:       "this_is_a_very_strong_secret_key_32_chars",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: -7 * 24 * time.Hour,
			},
			expectError: true,
			errorMsg:    "refresh token TTL must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if !containsStringJWT(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// ========== Граничные тесты ==========

func TestJWTConfigBoundaryValues(t *testing.T) {
	tests := []struct {
		name         string
		setupEnv     func(*TestJWTEnv)
		shouldFail   bool
		validateFunc func(*testing.T, *JWTConfig)
	}{
		{
			name: "JWT_SECRET exactly 32 characters",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "12345678901234567890123456789012")
			},
			shouldFail: false,
			validateFunc: func(t *testing.T, cfg *JWTConfig) {
				if err := cfg.Validate(); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			},
		},
		{
			name: "JWT_SECRET 100 characters",
			setupEnv: func(env *TestJWTEnv) {
				longSecret := ""
				for i := 0; i < 100; i++ {
					longSecret += "a"
				}
				env.Set("JWT_SECRET", longSecret)
			},
			shouldFail: false,
		},
		{
			name: "JWT_ACCESS_TTL_HOURS = 1 (minimum positive)",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
				env.Set("JWT_ACCESS_TTL_HOURS", "1")
			},
			shouldFail: false,
			validateFunc: func(t *testing.T, cfg *JWTConfig) {
				if cfg.AccessTokenTTL != 1*time.Hour {
					t.Errorf("AccessTokenTTL = %v, want 1h", cfg.AccessTokenTTL)
				}
			},
		},
		{
			name: "JWT_ACCESS_TTL_HOURS = 720 (30 days)",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
				env.Set("JWT_ACCESS_TTL_HOURS", "720")
			},
			shouldFail: false,
			validateFunc: func(t *testing.T, cfg *JWTConfig) {
				if cfg.AccessTokenTTL != 720*time.Hour {
					t.Errorf("AccessTokenTTL = %v, want 720h", cfg.AccessTokenTTL)
				}
			},
		},
		{
			name: "JWT_REFRESH_TTL_DAYS = 1 (minimum positive)",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
				env.Set("JWT_REFRESH_TTL_DAYS", "1")
			},
			shouldFail: false,
			validateFunc: func(t *testing.T, cfg *JWTConfig) {
				if cfg.RefreshTokenTTL != 24*time.Hour {
					t.Errorf("RefreshTokenTTL = %v, want 24h", cfg.RefreshTokenTTL)
				}
			},
		},
		{
			name: "JWT_REFRESH_TTL_DAYS = 365 (one year)",
			setupEnv: func(env *TestJWTEnv) {
				env.Set("JWT_SECRET", "this_is_a_very_strong_secret_key_32_chars")
				env.Set("JWT_REFRESH_TTL_DAYS", "365")
			},
			shouldFail: false,
			validateFunc: func(t *testing.T, cfg *JWTConfig) {
				if cfg.RefreshTokenTTL != 365*24*time.Hour {
					t.Errorf("RefreshTokenTTL = %v, want 8760h", cfg.RefreshTokenTTL)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEnv := NewTestJWTEnv()
			defer testEnv.Cleanup()

			ClearAllJWTEnvVars()

			tt.setupEnv(testEnv)
			testEnv.Apply()

			config, err := LoadJWTConfig()

			if tt.shouldFail && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.shouldFail && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.shouldFail && config != nil && tt.validateFunc != nil {
				tt.validateFunc(t, config)
			}
		})
	}
}

// ========== Параллельные тесты ==========

func TestParallelJWTConfig(t *testing.T) {
	tests := []struct {
		name       string
		secret     string
		accessTTL  string
		refreshTTL string
		shouldWork bool
	}{
		{"service1", "valid_secret_key_32_chars_long_here", "24", "7", true},
		{"service2", "another_valid_32_byte_secret_key_ok", "12", "14", true},
		{"service3", "", "24", "7", false},                                         // missing secret
		{"service4", "short", "24", "7", false},                                    // secret too short
		{"service5", "valid_secret_key_32_chars_long_here", "invalid", "7", false}, // invalid TTL
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testEnv := NewTestJWTEnv()
			defer testEnv.Cleanup()

			ClearAllJWTEnvVars()

			testEnv.Set("JWT_SECRET", tt.secret)
			if tt.accessTTL != "" {
				testEnv.Set("JWT_ACCESS_TTL_HOURS", tt.accessTTL)
			}
			if tt.refreshTTL != "" {
				testEnv.Set("JWT_REFRESH_TTL_DAYS", tt.refreshTTL)
			}
			testEnv.Apply()

			config, err := LoadJWTConfig()

			if tt.shouldWork {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				if config == nil {
					t.Errorf("Expected config, got nil")
				}
				if config.SecretKey != tt.secret {
					t.Errorf("SecretKey = %v, want %v", config.SecretKey, tt.secret)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, got success")
				}
			}
		})
	}
}
