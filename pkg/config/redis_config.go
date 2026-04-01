package configs

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// структура конфига для Redis
type RedisConfig struct {
	Host            string        // Хост, где расположен redis
	Port            string        // Порт для подключения
	Password        string        // Пароль
	DB              int32         // 16 пронумерованных баз данных (0-15 по умолчанию), загружаем номер
	PoolSize        int32         // Максимальное количество одновременных TCP-соединений, которые клиент может открыть к Redis
	MinIdleConns    int32         // Минимальное количество соединений, которое нужно держать открытыми
	MaxRetries      int32         // Количество повторных запросов при временных сетевых сбоях
	DialTimeout     time.Duration // Максимальное время, которое клиент ждет при установке нового TCP-соединения с Redis сервером
	ReadTimeout     time.Duration // Таймаут чтения ответа от Redis
	WriteTimeout    time.Duration // Таймаут отправки команды в Redis
	IdleTimeout     time.Duration // Таймаут, по истечении которого закрывается неиспользуемое соединение
	PoolTimeout     time.Duration // Таймаут ожидания свободного соединения
	MaxConnAge      time.Duration // Соединение живет максимум заданное время в пуле соединений
	MinRetryBackOff time.Duration // Нижняя граница интервала попыток
	MaxRetryBackOff time.Duration // Верхняя граница интервала попыток
}

// NewRedisConfigFromEnv создает конфиг Redis из переменных окружения
// Возвращает ошибку, если обязательные поля не заполнены или значения некорректны
func NewRedisConfigFromEnv() (*RedisConfig, error) {
	var errors []string

	// Получаем значени хоста (есть дефолтные значения)
	host, err := getRequiredEnv("REDIS_HOST")
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Получаем значени порта (есть дефолтные значения)
	port, err := getRequiredEnv("REDIS_PORT")
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Получаем значени пароля (есть дефолтные значения)
	pass, err := getRequiredEnv("REDIS_PASSWORD")
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Если есть ошибки в обязательных полях - возвращаем сразу
	if len(errors) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(errors, ", "))
	}

	// Валидируем DB (Redis поддерживает 0-15 обычно, используем дэфолт)
	dbCount, err := getEnvAsInt32WithValidation("REDIS_DB", 5, 0, 15)
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Валидируем PoolSize (разумные границы 1-1000)
	poolSize, err := getEnvAsInt32WithValidation("REDIS_POOL_SIZE", 100, 1, 1000)
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Валидируем MinIdleConns (не может быть больше PoolSize)
	minIdleConns, err := getEnvAsInt32WithValidation("REDIS_MIN_IDLE_CONNS", 100, 1, 1000)
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Дополнительная проверка: MinIdleConns <= PoolSize
	if minIdleConns > poolSize {
		errors = append(errors, fmt.Sprintf("REDIS_MIN_IDLE_CONNS (%d) cannot be greater than REDIS_POOL_SIZE (%d)", minIdleConns, poolSize))
	}

	// Загружаем таймауты с валидацией

	dialTimeout, err := getEnvAsDurationWithValidation("REDIS_DIAL_TIMEOUT", 5*time.Second, 1*time.Second, 30*time.Second)
	if err != nil {
		errors = append(errors, err.Error())
	}

	readTimeout, err := getEnvAsDurationWithValidation("REDIS_DIAL_TIMEOUT", 3*time.Second, 100*time.Millisecond, 30*time.Second)
	if err != nil {
		errors = append(errors, err.Error())
	}

	writeTimeout, err := getEnvAsDurationWithValidation("REDIS_WRITE_TIMEOUT", 3*time.Second, 100*time.Millisecond, 30*time.Second)
	if err != nil {
		errors = append(errors, err.Error())
	}

	idleTimeout, err := getEnvAsDurationWithValidation("REDIS_IDLE_TIMEOUT", 5*time.Minute, 1*time.Minute, 24*time.Hour)
	if err != nil {
		errors = append(errors, err.Error())
	}

	poolTimeout, err := getEnvAsDurationWithValidation("REDIS_POOL_TIMEOUT", 4*time.Minute, 1*time.Minute, 7*time.Minute)
	if err != nil {
		errors = append(errors, err.Error())
	}

	maxConnAge, err := getEnvAsDurationWithValidation("REDIS_MAX_CON_AGE", 4*time.Minute, 1*time.Minute, 50*time.Minute)
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Валидируем MinIdleConns (не может быть больше PoolSize)
	maxRetries, err := getEnvAsInt32WithValidation("REDIS_MAX_RETRIES", 2, 0, 3)
	if err != nil {
		errors = append(errors, err.Error())
	}

	minRetryBackoff, err := getEnvAsDurationWithValidation("REDIS_MIN_RETRY_BACKOFF_MS", 100*time.Millisecond, 50*time.Millisecond, 300*time.Millisecond)
	if err != nil {
		errors = append(errors, err.Error())
	}

	maxRetryBackoff, err := getEnvAsDurationWithValidation("REDIS_MAX_RETRY_BACKOFF_MS", 1000*time.Millisecond, 512*time.Millisecond, 2000*time.Millisecond)
	if err != nil {
		errors = append(errors, err.Error())
	}

	// Если есть ошибки валидации - возвращаем их
	if len(errors) > 0 {
		return nil, fmt.Errorf("configuration errors:\n%s", strings.Join(errors, "\n"))
	}

	return &RedisConfig{
		Host:            host,            // Хост, где расположен redis
		Port:            port,            // Порт для подключения
		Password:        pass,            // Пароль
		DB:              dbCount,         // 16 пронумерованных баз данных (0-15 по умолчанию), загружаем номер
		PoolSize:        poolSize,        // Максимальное количество одновременных TCP-соединений, которые клиент может открыть к Redis
		MinIdleConns:    minIdleConns,    // Минимальное количество соединений, которое нужно держать открытыми
		MaxRetries:      maxRetries,      // Количество повторных запросов при временных сетевых сбоях
		DialTimeout:     dialTimeout,     // Максимальное время, которое клиент ждет при установке нового TCP-соединения с Redis сервером
		ReadTimeout:     readTimeout,     // Таймаут чтения ответа от Redis
		WriteTimeout:    writeTimeout,    // Таймаут отправки команды в Redis
		IdleTimeout:     idleTimeout,     // Таймаут, по истечении которого закрывается неиспользуемое соединение
		PoolTimeout:     poolTimeout,     // Таймаут ожидания свободного соединения
		MaxConnAge:      maxConnAge,      // Соединение живет максимум заданное время в пуле соединений
		MinRetryBackOff: minRetryBackoff, // Нижняя граница интервала попыток
		MaxRetryBackOff: maxRetryBackoff, // Верхняя граница интервала попыток
	}, nil
}

// для создания клиента redis необходимо передать указатель на структуру опций: *redis.Options
func (r *RedisConfig) ToRedisOptions() *redis.Options {
	return &redis.Options{
		Addr:     r.Host + ":" + r.Port,
		Password: r.Password,
		DB:       int(r.DB),
		// Пул соединений (зависит от нагрузки)
		PoolSize:     int(r.PoolSize),     // 50-200 для большинства приложений
		MinIdleConns: int(r.MinIdleConns), // 20-30% от PoolSize
		IdleTimeout:  r.IdleTimeout,       // 5-10 минут
		PoolTimeout:  r.PoolTimeout,       // Много конкурентных запросов
		MaxConnAge:   r.MaxConnAge,        // Обычно около 30 мин

		// Таймауты
		DialTimeout:  r.DialTimeout,
		ReadTimeout:  r.ReadTimeout,
		WriteTimeout: r.WriteTimeout,

		// Повторы
		MaxRetries:      int(r.MaxRetries),
		MinRetryBackoff: r.MinRetryBackOff,
		MaxRetryBackoff: r.MaxRetryBackOff,
	}
}
