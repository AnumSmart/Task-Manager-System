package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type BotConfig struct {
	BotToken    string `yaml:"bot_token"`    // BotToken - это уникальный идентификатор бота в Telegram (Выдается @BotFather при создании бота)
	WebhookURL  string `yaml:"webhook_url"`  // Публичный HTTPS URL, на который Telegram будет отправлять обновления
	WebhookPort string `yaml:"webhook_port"` // Локальный порт, на котором бот слушает входящие вебхуки. Обычно 8080, 8443 или 443 (для HTTPS)
	// Стандартные значения:
	//   - "development" - локальная разработка (больше логов, debug режим)
	//   - "staging" - тестовый сервер (похоже на production, но с тестовыми данными)
	//   - "production" - продакшен (минимум логов, максимум производительности)
	Environment string `yaml:"environment"`
}

// ручной анмаршалинг, чтобы брать переменную токена из env
func LoadBotConfig() (config *BotConfig, err error) {
	data, err := os.ReadFile("c:\\Son_Alex\\GO_projects\\task_management_system\\apps\\telegram-bot\\yml-configs\\botConfig.yml")

	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg BotConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	cfg.BotToken = os.ExpandEnv(cfg.BotToken) // заменит ${BOT_TOKEN} на значение из окружения
	return &cfg, nil
}
