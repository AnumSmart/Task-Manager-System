// описание конфига для подключения к базе PostgresQL
package configs

import (
	"fmt"
	"strings"
	"time"
)

// структура конфига для базы
type PostgresDBConfig struct {
	DSN string

	// Configure connection pool settings
	MaxConns int32
	MinConns int32

	// Configure connection health checks
	HealthCheckPeriod time.Duration
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration

	// Configure connection timeouts
	ConnectTimeout time.Duration
}

// NewPostgresDBConfigFromEnv создает конфиг PostgreSQL из переменных окружения
// Возвращает ошибку, если обязательные поля не заполнены или значения некорректны
func NewPostgresDBConfigFromEnv() (*PostgresDBConfig, error) {
	var errors []string

	// Получаем обязательные поля с проверкой
	host, err := getRequiredEnv("DB_HOST")
	if err != nil {
		errors = append(errors, err.Error())
	}

	user, err := getRequiredEnv("DB_USER")
	if err != nil {
		errors = append(errors, err.Error())
	}

	password, err := getRequiredEnv("DB_PASSWORD")
	if err != nil {
		errors = append(errors, err.Error())
	}

	dbName, err := getRequiredEnv("DB_NAME")
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Если есть ошибки в обязательных полях - возвращаем сразу
	if len(errors) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(errors, ", "))
	}

	// Получаем опциональные поля со значениями по умолчанию
	port := getEnvWithDefault("DB_PORT", "5432")
	sslMode := getEnvWithDefault("DB_SSL_MODE", "disable")

	// Собираем DSN
	dsn := buildDSN(host, port, user, password, dbName, sslMode)

	// Получаем числовые значения с валидацией
	maxConns, err := getEnvAsInt32WithValidation("DB_MAX_CONNS", 10, 1, 100)
	if err != nil {
		errors = append(errors, err.Error())
	}

	minConns, err := getEnvAsInt32WithValidation("DB_MIN_CONNS", 2, 0, 50)
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Проверяем что minConns <= maxConns
	if minConns > maxConns {
		errors = append(errors, fmt.Sprintf("DB_MIN_CONNS (%d) cannot be greater than DB_MAX_CONNS (%d)", minConns, maxConns))
	}

	// Получаем значения duration с валидацией
	healthCheckPeriod, err := getEnvAsDurationWithValidation("DB_HEALTH_CHECK_PERIOD", 60*time.Second, 1*time.Second, 300*time.Second)
	if err != nil {
		errors = append(errors, err.Error())
	}

	maxConnLifetime, err := getEnvAsDurationWithValidation("DB_MAX_CONN_LIFETIME", 3600*time.Second, 1*time.Second, 24*time.Hour)
	if err != nil {
		errors = append(errors, err.Error())
	}

	maxConnIdleTime, err := getEnvAsDurationWithValidation("DB_MAX_CONN_IDLE_TIME", 1800*time.Second, 1*time.Second, 24*time.Hour)
	if err != nil {
		errors = append(errors, err.Error())
	}

	connectTimeout, err := getEnvAsDurationWithValidation("DB_CONNECT_TIMEOUT", 5*time.Second, 1*time.Second, 60*time.Second)
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Проверяем что maxConnIdleTime <= maxConnLifetime
	if maxConnIdleTime > maxConnLifetime {
		errors = append(errors, fmt.Sprintf("DB_MAX_CONN_IDLE_TIME (%v) cannot be greater than DB_MAX_CONN_LIFETIME (%v)", maxConnIdleTime, maxConnLifetime))
	}

	// Если есть ошибки валидации - возвращаем их
	if len(errors) > 0 {
		return nil, fmt.Errorf("configuration errors:\n%s", strings.Join(errors, "\n"))
	}

	return &PostgresDBConfig{
		DSN:               dsn,
		MaxConns:          maxConns,
		MinConns:          minConns,
		HealthCheckPeriod: healthCheckPeriod,
		MaxConnLifetime:   maxConnLifetime,
		MaxConnIdleTime:   maxConnIdleTime,
		ConnectTimeout:    connectTimeout,
	}, nil
}
