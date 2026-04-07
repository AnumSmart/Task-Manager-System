package config

import (
	"fmt"
	"os"
	configs "pkg/config"

	"github.com/joho/godotenv"
)

// Конфигурация сервиса пользователей
type UserServiceConfig struct {
	GRPCServerConfig *configs.GRPCServerConfig
	PostgresDBConfig *configs.PostgresDBConfig
	RedisConf        *configs.RedisConfig
}

// путь к .env файлу
const (
	envPath = "c:\\Users\\aliaksei.makarevich\\go\\task_management_system_v_1_20\\apps\\user-service\\.env"
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

	// загружаем данные из .env файла для postgresDBConfig
	postgresDBConfig, err := configs.NewPostgresDBConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	// загружаем данные из .env файла для redisConfig
	redisConfig, err := configs.NewRedisConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	return &UserServiceConfig{
		GRPCServerConfig: grpcServerConfig,
		PostgresDBConfig: postgresDBConfig,
		RedisConf:        redisConfig,
	}, nil
}
