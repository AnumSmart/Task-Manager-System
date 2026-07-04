package config

import (
	"fmt"
	"os"
	"pkg/configs"
	"pkg/logger"

	"github.com/joho/godotenv"
)

// путь к .env файлу
const (
	envPath = "c:\\Son_Alex\\GO_projects\\task_management_system\\apps\\telegram-bot\\.env"
)

// AppConfig объединяет все конфигурации сервиса
type AppConfig struct {
	Bot          *BotConfig
	BotServer    *BotHttpServerConfig
	GrpcClient   *GrpcClientConfig
	LoggerConfig *logger.LoggerConfig
}

// LoadAppConfig загружает все конфиги из указанного .env файла
func LoadAppConfig(envPath string) (*AppConfig, error) {
	config := &AppConfig{}

	// проверяем наличие пути до файла .env
	if envPath == "" {
		return nil, fmt.Errorf("Error during loading config: empty env file path")
	}

	err := godotenv.Load(envPath)
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	// получаем конфиг бота
	bot, err := LoadBotConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load bot config: %w", err)
	}

	config.Bot = bot

	// получаем конфиг http серверя для бота
	botServer, err := LoadBotHttpServerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load bot server config: %w", err)
	}

	config.BotServer = botServer

	// получаем конфиг для grpc клиента
	grpcClient, err := LoadGrpcClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load gRPC client config: %w", err)
	}

	config.GrpcClient = grpcClient

	logger, err := config.LoadLoggerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load logger config: %w", err)
	}

	config.LoggerConfig = logger

	return config, nil
}

// метод получения конфига логгера
func (a *AppConfig) LoadLoggerConfig() (*logger.LoggerConfig, error) {
	// проверка, что указан путь к .yml файлу
	loggerConfigPath := os.Getenv("LOGGER_CONFIG_ADDRESS_STRING")
	if loggerConfigPath == "" {
		// Если переменная окружения не задана - предупреждение, но можно продолжить
		fmt.Println("WARNING: LOGGER_ADDRESS_STRING is not set, using default config")
	}

	loggerConfig, err := configs.LoadYAMLConfig[logger.LoggerConfig](loggerConfigPath, logger.DefaultLoggerConfig)
	if err != nil {
		return nil, fmt.Errorf("Error during loading config [logger config]: %s\n", err.Error())
	}

	return loggerConfig, nil
}
