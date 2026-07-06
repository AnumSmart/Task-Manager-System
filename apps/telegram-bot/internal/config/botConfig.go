package config

import (
	"fmt"
	"os"
	"pkg/configs"
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

func LoadBotConfig() (config *BotConfig, err error) {
	// загружаем конфиг из .yml файла
	botConfig, err := configs.LoadYAMLConfig[BotConfig](os.Getenv("BOT_CONFIG_ADDRESS_STRING"), UseDefaultBotConfig)
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	return botConfig, nil
}

func UseDefaultBotConfig() *BotConfig {
	return &BotConfig{
		BotToken:    "123456:ABC",
		WebhookURL:  "https://example.com/webhook",
		WebhookPort: "8080",
		Environment: "production",
	}
}
