package config

import (
	"fmt"
	"log"
	"os"
	"pkg/configs"
	"time"

	"github.com/joho/godotenv"
)

// конфиг для подключения к grpc клиенту
type GrpcClientConfig struct {
	Host        string        `mapstructure:"host"`
	Port        string        `mapstructure:"port"`
	ServiceName string        `mapstructure:"service_name"`
	ServiceKey  string        `mapstructure:"service_key"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

// загружаем конфиг
func LoadGrpcClientConfig(envPath string) (*GrpcClientConfig, error) {
	if envPath == "" {
		return nil, fmt.Errorf("Error during loading config: empty env file path")
	}

	err := godotenv.Load(envPath)
	if err != nil {
		return nil, fmt.Errorf("Error during loading config: %s\n", err.Error())
	}

	config, err := configs.LoadYAMLConfig[GrpcClientConfig](os.Getenv("USER_GRPC_CLIENT_CONFIG_ADDRESS_STRING"), LoadDefaultGrpcClientCOnfig)
	if err != nil {
		// Обрабатываем ошибку (файл есть, но не читается или не парсится)
		log.Fatalf("Failed to load config: %v", err)
	}

	err = config.validate()
	if err != nil {
		log.Fatalf("Invalid grpc client config: %v", err)
	}

	return config, nil
}

func LoadDefaultGrpcClientCOnfig() *GrpcClientConfig {
	return &GrpcClientConfig{
		Host:        "localhost",
		Port:        "50051",
		ServiceName: "default",
		ServiceKey:  "some key",
		Timeout:     10,
	}
}

// валидация конфига
func (c *GrpcClientConfig) validate() error {
	if c.Host == "" {
		return fmt.Errorf("config validation: empty host")
	}

	if c.Port == "" {
		return fmt.Errorf("config validation: empty port")
	}

	if c.ServiceKey == "some key" {
		return fmt.Errorf("config validation: default service key")
	}
	return nil
}

// метод получения полного адреса сервера
func (c *GrpcClientConfig) GetAddress() string {
	address := c.Host + ":" + c.Port
	return address
}
