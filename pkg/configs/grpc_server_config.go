package configs

import (
	"fmt"
	"time"
)

// структура конфига grpc сервера
type GRPCServerConfig struct {
	Host                  string        `yaml:"host"`
	Port                  string        `yaml:"port"`
	MaxConnectionIdle     time.Duration `yaml:"max_connection_idle"`      // Если клиент молчит 15 минут - можно закрыть соединение
	MaxConnectionAge      time.Duration `yaml:"max_connection_age"`       // Максимальное время жизни соединения - 30 минут
	MaxConnectionAgeGrace time.Duration `yaml:"max_connection_age_grace"` // Даем 5 минут на завершение текущих дел перед закрытием
	KeepaliveTime         time.Duration `yaml:"keepalive_time"`           // Каждые 5 минут проверяем, жив ли клиент
	KeepaliveTimeout      time.Duration `yaml:"keepalive_timeout"`        // Ждем ответ 20 секунд, если не отвечает - считаем отключившимся
	MaxRecvMsgSize        int           `yaml:"max_recv_msg_size"`        // Максимальный размер принимаемого сообщения - 10 МБ
	MaxSendMsgSize        int           `yaml:"max_send_msg_size"`        // Максимальный размер отправляемого сообщения - тоже 10 МБ
}

// метод получения адреса
func (c *GRPCServerConfig) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// дэфолтный конфиг
func UseDefaultGRPCServerConfig() *GRPCServerConfig {
	return &GRPCServerConfig{
		Host:                  "0.0.0.0",
		Port:                  "50051",
		MaxConnectionIdle:     15 * time.Minute,
		MaxConnectionAge:      30 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Minute,
		KeepaliveTime:         5 * time.Minute,
		KeepaliveTimeout:      20 * time.Second,
		MaxRecvMsgSize:        10485760,
		MaxSendMsgSize:        10485760,
	}
}
