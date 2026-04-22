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

	return &UserServiceConfig{
		GRPCServerConfig:   grpcServerConfig,
		PostgresDBConfig:   postgresDBConfig,
		RedisConf:          redisConfig,
		RedisBlackListConf: blacklistConfig,
		JWTConfig:          jwtConfig,
	}, nil
}
