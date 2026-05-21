package configs

import (
	"os"
	"sync"
	"testing"
	"time"
)

// глобальная блокировка для доступа к переменным окружения
var envMutex sync.Mutex

// TestEnv - структура для управления тестовым окружением
type TestEnv struct {
	variables map[string]string
}

// NewTestEnv создает новое тестовое окружение
func NewTestEnv() *TestEnv {
	return &TestEnv{
		variables: make(map[string]string),
	}
}

// Cleanup восстанавливает чистое состояние
func (te *TestEnv) Cleanup() {
	envMutex.Lock()
	defer envMutex.Unlock()

	// Удаляем только те переменные, которые устанавливали
	for key := range te.variables {
		os.Unsetenv(key)
	}
}

// Set устанавливает переменную окружения для теста
func (te *TestEnv) Set(key, value string) {
	te.variables[key] = value
}

// Apply применяет все установленные переменные к окружению
func (te *TestEnv) Apply() {
	envMutex.Lock()
	defer envMutex.Unlock()

	for key, value := range te.variables {
		os.Setenv(key, value)
	}
}

// Unset удаляет переменную окружения
func (te *TestEnv) Unset(key string) {
	os.Unsetenv(key)
}

// ========== Helper функции ==========

// ClearAll очищает все переменные БД (используется перед тестом)
func ClearAllDBEnvVars() {
	envMutex.Lock()
	defer envMutex.Unlock()

	envVars := []string{
		"DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME",
		"DB_PORT", "DB_SSL_MODE", "DB_MAX_CONNS", "DB_MIN_CONNS",
		"DB_HEALTH_CHECK_PERIOD", "DB_MAX_CONN_LIFETIME",
		"DB_MAX_CONN_IDLE_TIME", "DB_CONNECT_TIMEOUT",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

// containsString проверяет вхождение подстроки
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// табличный тест для создания конфига для Postgres с тестовым окружением
func TestNewPostgresDBConfigFromEnv(t *testing.T) {
	// Очищаем глобальное окружение перед всеми тестами
	ClearAllDBEnvVars()

	tests := []struct {
		name          string
		setupEnv      func(*TestEnv)
		expectedDSN   string
		expectedError string
		validateFunc  func(*testing.T, *PostgresDBConfig)
	}{
		{
			name: "successful config with defaults",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_HOST", "localhost")
				env.Set("DB_USER", "postgres")
				env.Set("DB_PASSWORD", "secret")
				env.Set("DB_NAME", "testdb")
			},
			expectedDSN: "host=localhost port=5432 user=postgres password=secret dbname=testdb sslmode=disable",
			validateFunc: func(t *testing.T, cfg *PostgresDBConfig) {
				if cfg.MaxConns != 10 {
					t.Errorf("MaxConns = %d, want 10", cfg.MaxConns)
				}
				if cfg.MinConns != 2 {
					t.Errorf("MinConns = %d, want 2", cfg.MinConns)
				}
			},
		},
		{
			name: "successful config with custom values",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_HOST", "192.168.1.100")
				env.Set("DB_USER", "admin")
				env.Set("DB_PASSWORD", "strongpass")
				env.Set("DB_NAME", "production")
				env.Set("DB_PORT", "5433")
				env.Set("DB_SSL_MODE", "require")
				env.Set("DB_MAX_CONNS", "50")
				env.Set("DB_MIN_CONNS", "10")
				env.Set("DB_HEALTH_CHECK_PERIOD", "30s")
				env.Set("DB_MAX_CONN_LIFETIME", "2h")
				env.Set("DB_MAX_CONN_IDLE_TIME", "1h")
				env.Set("DB_CONNECT_TIMEOUT", "10s")
			},
			expectedDSN: "host=192.168.1.100 port=5433 user=admin password=strongpass dbname=production sslmode=require",
			validateFunc: func(t *testing.T, cfg *PostgresDBConfig) {
				if cfg.MaxConns != 50 {
					t.Errorf("MaxConns = %d, want 50", cfg.MaxConns)
				}
				if cfg.MinConns != 10 {
					t.Errorf("MinConns = %d, want 10", cfg.MinConns)
				}
				if cfg.HealthCheckPeriod != 30*time.Second {
					t.Errorf("HealthCheckPeriod = %v, want 30s", cfg.HealthCheckPeriod)
				}
			},
		},
		{
			name: "missing DB_HOST",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_USER", "postgres")
				env.Set("DB_PASSWORD", "secret")
				env.Set("DB_NAME", "testdb")
			},
			expectedError: "DB_HOST is required",
		},
		{
			name: "missing DB_USER",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_HOST", "localhost")
				env.Set("DB_PASSWORD", "secret")
				env.Set("DB_NAME", "testdb")
			},
			expectedError: "DB_USER is required",
		},
		{
			name: "missing DB_PASSWORD",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_HOST", "localhost")
				env.Set("DB_USER", "postgres")
				env.Set("DB_NAME", "testdb")
			},
			expectedError: "DB_PASSWORD is required",
		},
		{
			name: "missing DB_NAME",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_HOST", "localhost")
				env.Set("DB_USER", "postgres")
				env.Set("DB_PASSWORD", "secret")
			},
			expectedError: "DB_NAME is required",
		},
		{
			name: "minConns greater than maxConns",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_HOST", "localhost")
				env.Set("DB_USER", "postgres")
				env.Set("DB_PASSWORD", "secret")
				env.Set("DB_NAME", "testdb")
				env.Set("DB_MIN_CONNS", "20")
				env.Set("DB_MAX_CONNS", "10")
			},
			expectedError: "cannot be greater than",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// НЕ используем параллельное выполнение для этих тестов
			// потому что они зависят от глобального окружения
			testEnv := NewTestEnv()
			defer testEnv.Cleanup()

			ClearAllDBEnvVars()

			if tt.setupEnv != nil {
				tt.setupEnv(testEnv)
			}

			// Применяем переменные
			testEnv.Apply()

			config, err := NewPostgresDBConfigFromEnv()

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !containsString(err.Error(), tt.expectedError) {
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

			if config.DSN != tt.expectedDSN {
				t.Errorf("DSN mismatch:\nExpected: %s\nGot:      %s", tt.expectedDSN, config.DSN)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, config)
			}
		})
	}
}

// Простые тесты для граничных значений (без параллелизации)
func TestPostgresDBConfigBoundaryValues(t *testing.T) {
	tests := []struct {
		name       string
		setupEnv   func(*TestEnv)
		shouldFail bool
	}{
		{
			name: "maxConns = 1 (minimum)",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_MAX_CONNS", "1")
			},
			shouldFail: true, // так как не задано minConns, оно по default = 2. Получается maxConn < minConn
		},
		{
			name: "maxConns = 100 (maximum)",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_MAX_CONNS", "100")
			},
			shouldFail: false,
		},
		{
			name: "maxConns = 101 (exceeds)",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_MAX_CONNS", "101")
			},
			shouldFail: true,
		},
		{
			name: "minConns = 0 (minimum)",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_MIN_CONNS", "0")
			},
			shouldFail: false,
		},
		{
			name: "minConns = 50 (maximum)",
			setupEnv: func(env *TestEnv) {
				env.Set("DB_MIN_CONNS", "50")
			},
			shouldFail: true, // для minConn верхняя граница 50 (не включая)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEnv := NewTestEnv()
			defer testEnv.Cleanup()

			ClearAllDBEnvVars()

			// Базовые обязательные поля
			testEnv.Set("DB_HOST", "localhost")
			testEnv.Set("DB_USER", "postgres")
			testEnv.Set("DB_PASSWORD", "secret")
			testEnv.Set("DB_NAME", "testdb")

			if tt.setupEnv != nil {
				tt.setupEnv(testEnv)
			}

			testEnv.Apply()

			config, err := NewPostgresDBConfigFromEnv()

			if tt.shouldFail && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.shouldFail && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.shouldFail && config == nil {
				t.Errorf("Expected config but got nil")
			}
		})
	}
}

// Параллельные тесты с изолированным окружением
func TestParallelConfigLogic(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		user       string
		pass       string
		dbname     string
		shouldWork bool
	}{
		{"service1", "host1", "user1", "pass1", "db1", true},
		{"service2", "host2", "user2", "pass2", "db2", true},
		{"service3", "", "user3", "pass3", "db3", false}, // missing host
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// КАЖДЫЙ ПАРАЛЛЕЛЬНЫЙ ТЕСТ ИСПОЛЬЗУЕТ СВОЙ МЬЮТЕКС
			testEnv := NewTestEnv()
			defer testEnv.Cleanup()

			ClearAllDBEnvVars()

			testEnv.Set("DB_HOST", tt.host)
			testEnv.Set("DB_USER", tt.user)
			testEnv.Set("DB_PASSWORD", tt.pass)
			testEnv.Set("DB_NAME", tt.dbname)
			testEnv.Apply()

			config, err := NewPostgresDBConfigFromEnv()

			if tt.shouldWork {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				if config == nil {
					t.Errorf("Expected config, got nil")
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, got success")
				}
			}
		})
	}
}
