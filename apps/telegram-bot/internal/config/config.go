package config

import (
	"fmt"
	"os"
	configs "pkg/configs"

	"github.com/joho/godotenv"
)

// структрура конфига для сервиса сервера бота
type BotServiceConfig struct {
	HTTPServerConfig *configs.HttpServerConfig
}

// путь к .env файлу
const (
	envPath = "c:\\Son_Alex\\GO_projects\\task_management_system\\apps\\telegram-bot\\.env"
)

// загружаем конфиг-данные из .env
func LoadConfig() (*BotServiceConfig, error) {
	err := godotenv.Load(envPath)
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	httpServerConf, err := configs.LoadYAMLConfig[configs.HttpServerConfig](os.Getenv("BOT_HTTP_SERVER_CONFIG_ADDRESS_STRING"), configs.UseDefaultServerConfig)
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	return &BotServiceConfig{
		HTTPServerConfig: httpServerConf,
	}, nil
}
