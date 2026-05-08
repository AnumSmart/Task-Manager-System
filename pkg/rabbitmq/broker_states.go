package rabbitmq

import "time"

const (
	// defaultReconnectDelay используется, если в конфиге не указан ReconnectDelay
	defaultReconnectDelay = 5 * time.Second

	// heartbeatInterval интервал проверки alive соединения
	heartbeatInterval = 30 * time.Second

	// maxReconnectDelay максимальная задержка перед переподключением
	maxReconnectDelay = 60 * time.Second

	// maxRetryDelay максимальная задержка перед повторной обработкой
	maxRetryDelay = 5 * time.Minute

	// confirmsBufferSize размер буфера для канала подтверждений
	confirmsBufferSize = 100
)

// ConnectionState представляет состояние соединения с RabbitMQ
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota // соединение отсутствует
	StateConnecting                          // попытка подключения
	StateConnected                           // соединение установлено
	StateClosing                             // закрывается (graceful shutdown)
)

func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateClosing:
		return "closing"
	default:
		return "unknown"
	}
}
