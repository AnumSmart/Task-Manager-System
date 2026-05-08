package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type BrokerInterface interface {
	Publish(routingKey string, body []byte, headers amqp.Table) error
	PublishWithConfirm(ctx context.Context, routingKey string, body []byte, headers amqp.Table) error
	Consume(handler func(amqp.Delivery) error, bindings ...string) error
	IsConnected() bool
	HealthCheck() error
	Close() error
}
