package configs

import (
	"testing"
	"time"
)

func TestGRPCServerConfig_Addr(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     string
		expected string
	}{
		{
			name:     "стандартные значения",
			host:     "localhost",
			port:     "8080",
			expected: "localhost:8080",
		},
		{
			name:     "пустой хост",
			host:     "",
			port:     "50051",
			expected: ":50051",
		},
		{
			name:     "пустой порт",
			host:     "127.0.0.1",
			port:     "",
			expected: "127.0.0.1:",
		},
		{
			name:     "IPv6 адрес",
			host:     "::1",
			port:     "9090",
			expected: "::1:9090",
		},
		{
			name:     "domain name",
			host:     "grpc-server.local",
			port:     "50051",
			expected: "grpc-server.local:50051",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &GRPCServerConfig{
				Host: tt.host,
				Port: tt.port,
			}
			addr := config.Addr()
			if addr != tt.expected {
				t.Errorf("Addr() = %v, expected %v", addr, tt.expected)
			}
		})
	}
}

func TestUseDefaultGRPCServerConfig(t *testing.T) {
	config := UseDefaultGRPCServerConfig()

	// Проверяем, что конфиг не nil
	if config == nil {
		t.Fatal("UseDefaultGRPCServerConfig() вернула nil")
	}

	// Проверяем хост
	if config.Host != "0.0.0.0" {
		t.Errorf("Host = %v, expected 0.0.0.0", config.Host)
	}

	// Проверяем порт
	if config.Port != "50051" {
		t.Errorf("Port = %v, expected 50051", config.Port)
	}

	// Проверяем MaxConnectionIdle
	expectedMaxConnectionIdle := 15 * time.Minute
	if config.MaxConnectionIdle != expectedMaxConnectionIdle {
		t.Errorf("MaxConnectionIdle = %v, expected %v", config.MaxConnectionIdle, expectedMaxConnectionIdle)
	}

	// Проверяем MaxConnectionAge
	expectedMaxConnectionAge := 30 * time.Minute
	if config.MaxConnectionAge != expectedMaxConnectionAge {
		t.Errorf("MaxConnectionAge = %v, expected %v", config.MaxConnectionAge, expectedMaxConnectionAge)
	}

	// Проверяем MaxConnectionAgeGrace
	expectedMaxConnectionAgeGrace := 5 * time.Minute
	if config.MaxConnectionAgeGrace != expectedMaxConnectionAgeGrace {
		t.Errorf("MaxConnectionAgeGrace = %v, expected %v", config.MaxConnectionAgeGrace, expectedMaxConnectionAgeGrace)
	}

	// Проверяем KeepaliveTime
	expectedKeepaliveTime := 5 * time.Minute
	if config.KeepaliveTime != expectedKeepaliveTime {
		t.Errorf("KeepaliveTime = %v, expected %v", config.KeepaliveTime, expectedKeepaliveTime)
	}

	// Проверяем KeepaliveTimeout
	expectedKeepaliveTimeout := 20 * time.Second
	if config.KeepaliveTimeout != expectedKeepaliveTimeout {
		t.Errorf("KeepaliveTimeout = %v, expected %v", config.KeepaliveTimeout, expectedKeepaliveTimeout)
	}

	// Проверяем MaxRecvMsgSize (10 MB)
	expectedMaxRecvMsgSize := 10485760
	if config.MaxRecvMsgSize != expectedMaxRecvMsgSize {
		t.Errorf("MaxRecvMsgSize = %v, expected %v", config.MaxRecvMsgSize, expectedMaxRecvMsgSize)
	}

	// Проверяем MaxSendMsgSize (10 MB)
	expectedMaxSendMsgSize := 10485760
	if config.MaxSendMsgSize != expectedMaxSendMsgSize {
		t.Errorf("MaxSendMsgSize = %v, expected %v", config.MaxSendMsgSize, expectedMaxSendMsgSize)
	}

	// Проверяем AllowedServices
	if config.AllowedServices == nil {
		t.Error("AllowedServices не должен быть nil")
	}
	if len(config.AllowedServices) != 1 {
		t.Errorf("len(AllowedServices) = %v, expected 1", len(config.AllowedServices))
	}
	if key, exists := config.AllowedServices["default"]; !exists || key != "default" {
		t.Errorf("AllowedServices['default'] = %v, expected 'default'", key)
	}
}

func TestUseDefaultGRPCServerConfig_UniqueInstance(t *testing.T) {
	// Проверяем, что каждый вызов возвращает новый экземпляр
	config1 := UseDefaultGRPCServerConfig()
	config2 := UseDefaultGRPCServerConfig()

	if config1 == config2 {
		t.Error("Каждый вызов UseDefaultGRPCServerConfig() должен возвращать новый экземпляр")
	}

	// Изменяем первый конфиг
	config1.Host = "127.0.0.1"

	// Проверяем, что второй конфиг не изменился
	if config2.Host == "127.0.0.1" {
		t.Error("Изменение одного конфига не должно влиять на другой")
	}
	if config2.Host != "0.0.0.0" {
		t.Errorf("config2.Host = %v, expected 0.0.0.0", config2.Host)
	}
}

func TestGRPCServerConfig_AllowedServices_Modification(t *testing.T) {
	// Проверяем, что AllowedServices можно изменять
	config := UseDefaultGRPCServerConfig()

	// Добавляем новый сервис
	config.AllowedServices["test_service"] = "test_key"

	if len(config.AllowedServices) != 2 {
		t.Errorf("После добавления len(AllowedServices) = %v, expected 2", len(config.AllowedServices))
	}

	key, exists := config.AllowedServices["test_service"]
	if !exists || key != "test_key" {
		t.Errorf("test_service должен существовать с ключом test_key")
	}

	// Удаляем сервис
	delete(config.AllowedServices, "default")

	if len(config.AllowedServices) != 1 {
		t.Errorf("После удаления len(AllowedServices) = %v, expected 1", len(config.AllowedServices))
	}

	_, exists = config.AllowedServices["default"]
	if exists {
		t.Error("Сервис default должен быть удален")
	}
}

func TestGRPCServerConfig_Addr_Format(t *testing.T) {
	// Тестируем корректность форматирования адреса
	testCases := []struct {
		name   string
		config GRPCServerConfig
	}{
		{
			name: "полный конфиг",
			config: GRPCServerConfig{
				Host: "0.0.0.0",
				Port: "50051",
			},
		},
		{
			name: "пустой конфиг",
			config: GRPCServerConfig{
				Host: "",
				Port: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expected := tc.config.Host + ":" + tc.config.Port
			actual := tc.config.Addr()
			if actual != expected {
				t.Errorf("Addr() = %v, expected %v", actual, expected)
			}
		})
	}
}
