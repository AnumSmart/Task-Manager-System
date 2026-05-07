package config

import (
	"fmt"
	"os"
	configs "pkg/configs"

	"github.com/joho/godotenv"
)

// Конфигурация сервиса пользователей
type UserServiceConfig struct {
	GRPCServerConfig   *configs.GRPCServerConfig // конфиг для grpc сервера (загружается в шаблон из pkg, данные берутся из .yml файла)
	PostgresDBConfig   *configs.PostgresDBConfig // конфиг для экземпляра POSTGRES (загружается в шаблон из pkg, данные берутся из .env файла)
	RedisConf          *configs.RedisConfig      // конфиг для экземпляра REDIS (cache) (загружается в шаблон из pkg, данные берутся из .env файла)
	RedisBlackListConf *configs.RedisConfig      // конфиг для экземпляра REDIS (blacklist) (загружается в шаблон из pkg, данные берутся из .env файла)
	JWTConfig          *configs.JWTConfig        // конфиг для работы с JWT
	RabbitConfig       *configs.RabbitMQConfig   // конфиг для брокера сообщений RabbitMQ
}

// путь к .env файлу
const (
	envPath = "c:\\Son_Alex\\GO_projects\\task_management_system\\apps\\user-service\\.env"
)

// загружаем конфиг-данные из .env
func LoadConfig() (*UserServiceConfig, error) {
	err := godotenv.Load(envPath)
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	// загружаем конфиг для grpc сервера
	grpcServerConfig, err := configs.LoadYAMLConfig[configs.GRPCServerConfig](os.Getenv("GRPC_SERVER_CONFIG_ADDRESS_STRING"), configs.UseDefaultGRPCServerConfig)
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	// 🔑 Загружаем API Keys из переменных окружения (перезаписываем то, что было в YAML)
	if grpcServerConfig.AllowedServices == nil {
		grpcServerConfig.AllowedServices = make(map[string]string) // инициализируем мапу, если была не инициализирована
	}

	// Добавляем ключи из .env для каждого сервиса
	if key := os.Getenv("TASK_SERVICE_API_KEY"); key != "" {
		grpcServerConfig.AllowedServices["task-service"] = key
	}
	if key := os.Getenv("ANALYTICS_SERVICE_API_KEY"); key != "" {
		grpcServerConfig.AllowedServices["analytics-service"] = key
	}
	if key := os.Getenv("NOTIFICATION_SERVICE_API_KEY"); key != "" {
		grpcServerConfig.AllowedServices["notification-service"] = key
	}

	// загружаем данные из .env файла для postgresDBConfig
	postgresDBConfig, err := configs.NewPostgresDBConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	// загружаем данные из .env файла для redisConfig
	redisConfig, err := configs.NewRedisConfigFromEnv("REDIS_CACHE")
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	// загружаем данные из .env файла для blacklistConfig
	blacklistConfig, err := configs.NewRedisConfigFromEnv("REDIS_BLACKLIST")
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	// загружаем JWT конфигурацию из .env файла
	jwtConfig, err := configs.LoadJWTConfig()
	if err != nil {
		return nil, fmt.Errorf("Error during loading JWT config: %s\n", err.Error())
	}

	// валидируем JWT конфигурацию
	if err := jwtConfig.Validate(); err != nil {
		return nil, fmt.Errorf("Invalid JWT config: %s\n", err.Error())
	}

	// загружаем конфиг для RabbitMQ
	rabbitMQConfig, err := configs.LoadYAMLConfig[configs.RabbitMQConfig](os.Getenv("RABBITMQ_CONFIG_ADDRESS_STRING"), configs.NewRabbitMQConfig)
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	// Переопределяем URL и добавляем суффикс окружения из .env
	rabbitMQConfig.MergeWithEnv()

	// Валидируем конфиг
	if err := rabbitMQConfig.Validate(); err != nil {
		return nil, fmt.Errorf("Invalid RabbitMQ config: %s\n", err.Error())
	}

	return &UserServiceConfig{
		GRPCServerConfig:   grpcServerConfig,
		PostgresDBConfig:   postgresDBConfig,
		RedisConf:          redisConfig,
		RedisBlackListConf: blacklistConfig,
		JWTConfig:          jwtConfig,
		RabbitConfig:       rabbitMQConfig,
	}, nil
}
