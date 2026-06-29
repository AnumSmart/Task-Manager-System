package config

import (
	"fmt"
	"log"
	"os"
	configs "pkg/configs"
	"time"

	"github.com/joho/godotenv"
)

// структрура конфига для сервиса сервера бота
type BotHttpServerConfig struct {
	// Встраиваем базовый конфиг HTTP сервера
	// Это позволит использовать все поля host, port, timeout и т.д.
	configs.HttpServerConfig `yaml:",inline"` // inline чтобы поля были на одном уровне, а не вложены
	Mode                     string           `yaml:"mode"`    // "polling" или "webhook"
	Polling                  PollingConfig    `yaml:"polling"` // Настройки для polling режима
	Webhook                  WebhookConfig    `yaml:"webhook"` // Настройки для webhook режима
	Limits                   LimitsConfig     `yaml:"limits"`  // Лимиты и ограничения
}

// PollingConfig - настройки для режима Long Polling
type PollingConfig struct {
	Enabled        bool          `yaml:"enabled"`         // Активен ли polling режим
	Timeout        int           `yaml:"timeout"`         // Таймаут long polling в секундах
	Offset         int           `yaml:"offset"`          // Начальный offset обновлений
	AllowedUpdates []string      `yaml:"allowed_updates"` // Типы обновлений
	RetryDelay     time.Duration `yaml:"retry_delay"`     // Задержка при ошибке
}

// WebhookConfig - настройки для режима Webhook
type WebhookConfig struct {
	Enabled        bool     `yaml:"enabled"`         // Активен ли webhook режим
	URL            string   `yaml:"url"`             // Домен для вебхука
	Path           string   `yaml:"path"`            // Путь для вебхука
	MaxConnections int      `yaml:"max_connections"` // Максимум одновременных соединений
	AllowedUpdates []string `yaml:"allowed_updates"` // Типы обновлений
}

// LimitsConfig - лимиты и ограничения бота
type LimitsConfig struct {
	MaxMessageSize    int `yaml:"max_message_size"`   // Максимальный размер сообщения в символах
	RateLimit         int `yaml:"rate_limit"`         // Сообщений в минуту на пользователя
	ConcurrentWorkers int `yaml:"concurrent_workers"` // Количество одновременных обработчиков
}

// Вспомогательная структура для ошибок конфигурации
type ConfigError struct {
	Field string
	Msg   string
}

// путь к .env файлу
const (
	envPath = "c:\\Son_Alex\\GO_projects\\task_management_system\\apps\\telegram-bot\\.env"
)

// метод вспомогательной функции для формирования ошибок
func (e *ConfigError) Error() string {
	return "config error in field '" + e.Field + "': " + e.Msg
}

func LoadBotHttpServerConfig(envPath string) (*BotHttpServerConfig, error) {
	// проверяем наличие пути до файла .env
	if envPath == "" {
		return nil, fmt.Errorf("Error during loading config: empty env file path")
	}

	err := godotenv.Load(envPath)
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	// Загрузка конфига
	config, err := configs.LoadYAMLConfig[BotHttpServerConfig](os.Getenv("BOT_HTTP_SERVER_CONFIG_ADDRESS_STRING"), UseDefaultBotHttpServerConfig)
	if err != nil {
		// Обрабатываем ошибку (файл есть, но не читается или не парсится)
		log.Fatalf("Failed to load config: %v", err)
	}

	// Валидация конфига
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	return config, nil
}

// Конструктор по умолчанию для TelegramBotConfig
func UseDefaultBotHttpServerConfig() *BotHttpServerConfig {
	return &BotHttpServerConfig{
		HttpServerConfig: *configs.UseDefaultServerConfig(), // Берем дефолты из базового конфига

		Mode: "polling", // По умолчанию используем polling для разработки

		Polling: PollingConfig{
			Enabled:        true,
			Timeout:        60,
			Offset:         0,
			AllowedUpdates: []string{"message", "callback_query"},
			RetryDelay:     3 * time.Second,
		},

		Webhook: WebhookConfig{
			Enabled:        false,
			URL:            "",
			Path:           "/webhook",
			MaxConnections: 40,
			AllowedUpdates: []string{"message", "callback_query"},
		},

		Limits: LimitsConfig{
			MaxMessageSize:    4096,
			RateLimit:         30,
			ConcurrentWorkers: 10,
		},
	}
}

// Валидация конфига (опционально, но полезно)
func (c *BotHttpServerConfig) Validate() error {
	// Проверяем режим
	if c.Mode != "polling" && c.Mode != "webhook" {
		return &ConfigError{
			Field: "mode",
			Msg:   "must be either 'polling' or 'webhook'",
		}
	}

	// В зависимости от режима проверяем соответствующие настройки
	if c.Mode == "polling" {
		if !c.Polling.Enabled {
			return &ConfigError{
				Field: "polling.enabled",
				Msg:   "must be true when mode = 'polling'",
			}
		}
		if c.Polling.Timeout <= 0 {
			return &ConfigError{
				Field: "polling.timeout",
				Msg:   "must be positive",
			}
		}
	}

	if c.Mode == "webhook" {
		if !c.Webhook.Enabled {
			return &ConfigError{
				Field: "webhook.enabled",
				Msg:   "must be true when mode = 'webhook'",
			}
		}
		if c.Webhook.URL == "" {
			return &ConfigError{
				Field: "webhook.url",
				Msg:   "cannot be empty in webhook mode",
			}
		}
		if c.Webhook.MaxConnections <= 0 {
			return &ConfigError{
				Field: "webhook.max_connections",
				Msg:   "must be positive",
			}
		}
	}

	// Проверяем лимиты
	if c.Limits.MaxMessageSize <= 0 {
		return &ConfigError{
			Field: "limits.max_message_size",
			Msg:   "must be positive",
		}
	}

	return nil
}
