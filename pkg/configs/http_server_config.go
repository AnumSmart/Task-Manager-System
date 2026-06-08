package configs

import "time"

// структура для конфига сервера
type HttpServerConfig struct {
	Host           string        `yaml:"host"`
	Port           string        `yaml:"port"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	IdleTimeout    time.Duration `yaml:"idle_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
}

// функция для создания конфига сервера по - дефолту
func UseDefaultServerConfig() *HttpServerConfig {
	return &HttpServerConfig{
		Host:           "localhost",
		Port:           "8080",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

// метод конфига сервера для формирования адреса
func (c *HttpServerConfig) Addr() string {
	return c.Host + ":" + c.Port
}

// Вспомогательная структура для ошибок конфигурации
type ConfigError struct {
	Field string
	Msg   string
}

// метод вспомогательной функции для формирования ошибок
func (e *ConfigError) Error() string {
	return "config error in field '" + e.Field + "': " + e.Msg
}
