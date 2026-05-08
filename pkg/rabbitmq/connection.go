package rabbitmq

import (
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// connect устанавливает соединение с RabbitMQ и создаёт канал.
//
// Почему соединение и канал создаются вместе:
//   - Канал привязан к соединению и не может существовать без него
//   - При разрыве соединения канал тоже становится невалидным
//   - Восстановление всегда создаёт новую пару (connection + channel)

// connect устанавливает соединение с RabbitMQ
func (b *Broker) connect() error {
	b.connMu.Lock()
	defer b.connMu.Unlock()

	if b.getState() == StateClosing {
		return ErrBrokerClosed
	}

	b.setState(StateConnecting)
	log.Printf("[RabbitMQ] Connecting to %s", b.config.URL)

	// Устанавливаем соединение
	// DialConfig позволяет настроить heartbeat и другие параметры
	conn, err := amqp.Dial(b.config.URL)
	if err != nil {
		b.setState(StateDisconnected)
		return fmt.Errorf("dial failed: %w", err)
	}

	// Создаём канал
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		b.setState(StateDisconnected)
		return fmt.Errorf("channel creation failed: %w", err)
	}

	// Настраиваем QoS (Quality of Service) - сколько сообщений можно получать без подтверждения
	//
	// Почему это важно:
	//   - Без QoS сервер может отправить тысячи сообщений, переполнив память consumer'а
	//   - С QoS мы контролируем "окно" неподтверждённых сообщений
	//   - Значение должно быть оптимизировано под нагрузку сервиса
	if err := b.setupQoS(ch); err != nil {
		ch.Close()
		conn.Close()
		b.setState(StateDisconnected)
		return err
	}

	// Объявляем exchange (обменник)
	//
	// Почему exchange объявляется здесь:
	//   - Гарантирует, что exchange существует до первой публикации
	//   - Idempotent операция - если exchange уже есть, ничего не меняет
	//   - Параметры должны совпадать с существующими, иначе будет ошибка
	if err := b.setupExchange(ch); err != nil {
		ch.Close()
		conn.Close()
		b.setState(StateDisconnected)
		return err
	}

	// Включаем режим подтверждений (publisher confirms), если нужно
	//
	// Почему этот режим не включён по умолчанию:
	//   - Добавляет задержку на каждую публикацию (ждёт подтверждения от сервера)
	//   - Увеличивает надёжность, но снижает пропускную способность
	if err := b.setupConfirmMode(ch); err != nil {
		ch.Close()
		conn.Close()
		b.setState(StateDisconnected)
		return err
	}

	// Заменяем старые соединение и канал на новые
	b.closeOldConnection()
	b.conn = conn
	b.channel = ch

	// Настраиваем уведомления о закрытии соединения
	//
	// Почему это важно:
	//   - Канал notifyClose закрывается при разрыве соединения
	//   - Получаем уведомление для запуска процедуры переподключения
	//   - Используем make(chan) для создания нового канала каждый раз
	b.notifyClose = make(chan *amqp.Error)
	b.conn.NotifyClose(b.notifyClose)

	b.setState(StateConnected)
	b.reconnectAttempts = 0 // сбрасываем счётчик при успешном подключении
	b.lastReconnectAttempt = time.Now()

	log.Printf("✅ [RabbitMQ] Connected successfully")
	return nil
}

// setupQoS настраивает Quality of Service
func (b *Broker) setupQoS(ch *amqp.Channel) error {
	if b.config.PrefetchCount <= 0 {
		return nil
	}

	if err := ch.Qos(b.config.PrefetchCount, 0, false); err != nil {
		return fmt.Errorf("QoS setup failed: %w", err)
	}

	log.Printf("✅ [RabbitMQ] QoS configured: prefetch_count=%d", b.config.PrefetchCount)
	return nil
}

// setupExchange объявляет exchange
func (b *Broker) setupExchange(ch *amqp.Channel) error {
	err := ch.ExchangeDeclare(
		b.config.ExchangeName,
		b.config.ExchangeType,
		b.config.Durable,
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("exchange declare failed: %w", err)
	}
	return nil
}

// setupConfirmMode включает publisher confirms
func (b *Broker) setupConfirmMode(ch *amqp.Channel) error {
	if !b.config.EnableConfirmMode {
		return nil
	}

	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("confirm mode setup failed: %w", err)
	}

	b.publishMu.Lock()
	b.confirms = ch.NotifyPublish(make(chan amqp.Confirmation, confirmsBufferSize))
	b.publishMu.Unlock()

	log.Printf("✅ [RabbitMQ] Publisher confirms enabled")
	return nil
}

// closeOldConnection закрывает существующее соединение (при переподключении)
func (b *Broker) closeOldConnection() {
	if b.conn == nil {
		return
	}

	// Закрываем канал confirms, чтобы не блокировать публикации
	b.publishMu.Lock()
	if b.confirms != nil {
		close(b.confirms)
		b.confirms = nil
	}
	b.publishMu.Unlock()

	// Закрываем канал
	if b.channel != nil {
		b.channel.Close()

		// Закрываем соединение
		b.conn.Close()
	}
}

// reconnectionLoop - горутина для автоматического восстановления соединения.
//
// Почему нужен отдельный цикл:
//   - RabbitMQ может упасть или сетевое соединение может прерваться
//   - Без переподключения сервис перестанет отправлять/получать сообщения
//   - Бесконечные попытки с задержкой позволяют восстановиться после сбоя
func (b *Broker) reconnectionLoop() {
	defer b.closeWg.Done()

	for {
		select {
		case <-b.closeCh:
			// Брокер закрывается - выходим из цикла
			log.Printf("[RabbitMQ] Reconnection loop stopping")
			return

		case err := <-b.notifyClose:
			// Соединение разорвано (или закрыто вручную)
			if b.getState() == StateClosing {
				return // грациозное закрытие, не переподключаемся
			}

			log.Printf("[RabbitMQ] Connection lost: %v, attempting to reconnect...", err)
			b.setState(StateDisconnected)

			// Экспоненциальная задержка перед переподключением
			//
			// Почему экспоненциальная:
			//   - При первом сбое ждём короткое время (шанс, что быстро восстановится)
			//   - При повторных сбоях ждём всё дольше (даём время админу починить)
			//   - Не создаём "шторм" переподключений
			delay := b.calculateReconnectDelay()
			select {
			case <-time.After(delay):
			case <-b.closeCh:
				return
			}

			b.reconnectAttempts++
			if err := b.connect(); err != nil {
				log.Printf("[RabbitMQ] Reconnection failed: %v", err)
				continue
			}

			log.Printf("✅ [RabbitMQ] Reconnected successfully")
		}
	}
}

// calculateReconnectDelay рассчитывает задержку перед переподключением
func (b *Broker) calculateReconnectDelay() time.Duration {
	delay := b.config.ReconnectDelay
	for attempt := 1; attempt <= b.reconnectAttempts && attempt < 10; attempt++ {
		delay = delay * time.Duration(attempt)
		if delay > maxReconnectDelay {
			delay = maxReconnectDelay
		}
	}
	return delay
}
