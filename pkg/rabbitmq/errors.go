package rabbitmq

import "errors"

// Ошибки пакета rabbitmq

var (
	// ErrBrokerClosed возникает при попытке использовать закрытый брокер
	ErrBrokerClosed = errors.New("broker is closed")

	// ErrNotConnected возникает при попытке публикации без активного соединения
	ErrNotConnected = errors.New("not connected to RabbitMQ")

	// ErrNoConfirmMode возникает при попытке использовать ConfirmMode без его включения
	ErrNoConfirmMode = errors.New("confirm mode not enabled")
)
