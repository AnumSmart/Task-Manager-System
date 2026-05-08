package rabbitmq

import (
	"errors"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consume начинает потребление сообщений из очереди.
//
// Параметры:
//   - handler: функция обработки сообщения (возврат ошибки = nack, nil = ack)
//   - bindings: дополнительные binding'и (если нужно слушать несколько routing key)
//
// Почему handler принимает amqp.Delivery, а не []byte:
//   - Даёт доступ к заголовкам (нужны для retry, tracing)
//   - Позволяет управлять ack/nack на уровне обработчика
//   - Доступ к routing_key, timestamp и другим метаданным

func (b *Broker) Consume(handler func(amqp.Delivery) error, bindings ...string) error {
	if b.getState() != StateConnected {
		return ErrNotConnected
	}

	b.connMu.RLock()
	ch := b.channel
	b.connMu.RUnlock()

	if ch == nil {
		return ErrNotConnected
	}

	queueName := b.config.QueueName
	if queueName == "" {
		return errors.New("❌ queue name is required for consumption")
	}

	// Объявляем очередь (idempotent операция)
	//
	// Почему очередь объявляется здесь:
	//   - Consumer может быть запущен раньше, чем Publisher
	//   - Нужно гарантировать существование очереди перед Consumption
	//   - Параметры должны совпадать с существующей очередью
	q, err := b.declareQueue(ch)
	if err != nil {
		return err
	}

	// Привязываем очередь к exchange с routing key по умолчанию
	if err := b.bindQueue(ch, q.Name, bindings); err != nil {
		return err
	}

	// Начинаем потребление
	//
	// Почему auto-ack = false:
	//   - Позволяет контролировать подтверждение после успешной обработки
	//   - При падении сервиса сообщение не потеряется (вернётся в очередь)
	//   - Необходимо для реализации retry logic
	msgs, err := ch.Consume(
		q.Name,
		b.config.ConsumerTag,
		false, // auto-ack (false = ручное подтверждение)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("❌ consume failed: %w", err)
	}

	log.Printf("[RabbitMQ] Consuming from queue: %s, routing_key: %s, consumer_tag: %s",
		q.Name, b.config.RoutingKey, b.config.ConsumerTag)

	// Запускаем обработчик сообщений в отдельной горутине
	//
	// Почему горутина:
	//   - Не блокируем основной поток
	//   - Позволяет обрабатывать сообщения параллельно
	//   - Graceful shutdown дожидается завершения
	b.closeWg.Add(1)
	go b.messageProcessor(msgs, handler)

	return nil
}

// declareQueue объявляет очередь
func (b *Broker) declareQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		b.config.QueueName,
		b.config.Durable,
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
}

// bindQueue привязывает очередь к exchange
func (b *Broker) bindQueue(ch *amqp.Channel, queueName string, bindings []string) error {
	// Основной binding
	if err := ch.QueueBind(queueName, b.config.RoutingKey, b.config.ExchangeName, false, nil); err != nil {
		return fmt.Errorf("queue bind failed: %w", err)
	}

	// Дополнительные bindings
	for _, binding := range bindings {
		if err := ch.QueueBind(queueName, binding, b.config.ExchangeName, false, nil); err != nil {
			log.Printf("[RabbitMQ] Warning: failed to bind %s: %v", binding, err)
		}
	}

	return nil
}

// messageProcessor - внутренний обработчик сообщений.
//
// Отвечает за:
//  1. Вызов пользовательского handler'а
//  2. Ack/Nack в зависимости от результата
//  3. Подготовку к отправке в DLQ после исчерпания retry
func (b *Broker) messageProcessor(msgs <-chan amqp.Delivery, handler func(amqp.Delivery) error) {
	defer b.closeWg.Done()

	for {
		select {
		case <-b.closeCh:
			log.Printf("[RabbitMQ] Message processor stopping")
			return

		case msg, ok := <-msgs:
			if !ok {
				// Канал закрыт - выходим
				log.Printf("[RabbitMQ] Messages channel closed")
				return
			}

			// Обрабатываем сообщение
			err := handler(msg)

			if err != nil {
				// Обработка вернула ошибку - нужно повторить или отправить в DLQ
				b.handleError(msg, err)
			} else {
				// Успешная обработка - подтверждаем
				msg.Ack(false) // multiple=false - подтверждаем только это сообщение
			}
		}
	}
}
