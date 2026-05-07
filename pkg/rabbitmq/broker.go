package rabbitmq

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"pkg/configs"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Package rabbitmq предоставляет клиент для работы с RabbitMQ.
// Реализует подключение, публикацию и потребление сообщений с поддержкой:
//   - Автоматического восстановления соединения
//   - Publisher Confirms (гарантия доставки)
//   - Graceful shutdown
//   - Health checks

const (
	defaultReconnectDelay = 5 * time.Second  // defaultReconnectDelay используется, если в конфиге не указан ReconnectDelay
	heartbeatInterval     = 30 * time.Second // heartbeatInterval интервал проверки alive соединения
)

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

// ============================================================================
// ОШИБКИ
// ============================================================================

var (
	// ErrBrokerClosed возникает при попытке использовать закрытый брокер
	ErrBrokerClosed = errors.New("broker is closed")

	// ErrNotConnected возникает при попытке публикации без активного соединения
	ErrNotConnected = errors.New("not connected to RabbitMQ")

	// ErrNoConfirmMode возникает при попытке использовать ConfirmMode без его включения
	ErrNoConfirmMode = errors.New("confirm mode not enabled")
)

// ============================================================================
// ОСНОВНАЯ СТРУКТУРА BROKER
// ============================================================================

// Broker - основной клиент для работы с RabbitMQ.
// Инкапсулирует соединение, канал и управление их жизненным циклом.
//
// Почему структура спроектирована именно так:
//   1. Отдельные мьютексы для разных целей (подключение vs публикация) - меньше блокировок
//   2. Атомарные переменные для состояния - быстрые проверки без блокировок
//   3. Канал confirms отдельно от основного - подтверждения не блокируют публикацию
//   4. sync.Once для закрытия - защита от двойного закрытия

type Broker struct {
	config configs.RabbitMQConfig // конфиг для брокера

	// Основные компоненты RabbitMQ
	conn    *amqp.Connection // соединение с RabbitMQ
	channel *amqp.Channel    // канал (легковесное соединение внутри conn)

	// Канал для подтверждений (publisher confirms)
	// Создаётся отдельно, т.к. требует специального режима confirm.Select()
	confirms chan amqp.Confirmation

	// Управление состоянием
	state     atomic.Int32   // текущее состояние (ConnectionState)
	closeOnce sync.Once      // гарантирует однократное закрытие
	closeCh   chan struct{}  // сигнал для остановки всех горутин
	closeWg   sync.WaitGroup // ожидание завершения всех горутин

	// Мьютексы для разных целей (fine-grained locking)
	connMu    sync.RWMutex // защищает conn и channel
	publishMu sync.RWMutex // защищает публикацию (confirm режим)

	// Канал ошибок (небуферизированный - важные ошибки не должны теряться)
	// Почему небуферизированный: ошибка соединения критична, её нельзя пропустить
	notifyClose chan *amqp.Error // канал уведомлений о закрытии соединения

	// Здоровье и мониторинг
	lastReconnectAttempt time.Time // время последней попытки переподключения
	reconnectAttempts    int       // счётчик попыток подряд
}

// New создаёт новый экземпляр Broker и устанавливает соединение с RabbitMQ.
//
// Почему New делает подключение синхронно:
//   - Пользователь ожидает готовности брокера или получает ошибку сразу
//   - Упрощает отладку на старте (ошибка видна сразу, а не через N секунд)
//
// Для асинхронного подключения есть отдельный метод ConnectAsync (если понадобится)
func New(config *configs.RabbitMQConfig) (*Broker, error) {
	// Валидируем конфиг перед использованием
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("❌ invalid config: %w", err)
	}

	// Устанавливаем значения по умолчанию для отсутствующих параметров
	if config.ReconnectDelay <= 0 {
		config.ReconnectDelay = defaultReconnectDelay
	}

	// проводим базовую инициализацию брокера
	broker := &Broker{
		config:      *config,                // передаём конфиг из параметров
		closeCh:     make(chan struct{}),    // инициализируем канал закрытия
		notifyClose: make(chan *amqp.Error), // инициализируем канал ошибок
	}

	// Инициализируем состояние как disconnected
	broker.setState(StateDisconnected)

	// Устанавливаем соединение (синхронно)
	if err := broker.connect(); err != nil {
		return nil, fmt.Errorf("❌ failed to connect: %w", err)
	}

	// Запускаем горутину для автоматического восстановления соединения
	// Делаем это только после успешного первого подключения
	broker.closeWg.Add(1)
	go broker.reconnectionLoop()

	log.Printf("✅ [RabbitMQ] Broker initialized successfully, exchange: %s, queue: %s",
		config.ExchangeName, config.QueueName)

	return broker, nil
}

// ============================================================================
// ВНУТРЕННИЕ МЕТОДЫ (СОЕДИНЕНИЕ)
// ============================================================================

// connect устанавливает соединение с RabbitMQ и создаёт канал.
//
// Почему соединение и канал создаются вместе:
//   - Канал привязан к соединению и не может существовать без него
//   - При разрыве соединения канал тоже становится невалидным
//   - Восстановление всегда создаёт новую пару (connection + channel)

func (b *Broker) connect() error {
	b.connMu.Lock()         // ставим мьютекс на всю логику
	defer b.connMu.Unlock() // закрываем мьютекс по окончанию действия метода

	// Если уже в процессе закрытия, не пытаемся подключиться
	if b.getState() == StateClosing {
		return ErrBrokerClosed
	}

	// изменяем статус на - устанавливаем соединение
	b.setState(StateConnecting)
	log.Printf("[RabbitMQ] Connecting to %s", b.config.URL)

	// Устанавливаем соединение
	// DialConfig позволяет настроить heartbeat и другие параметры
	conn, err := amqp.Dial(b.config.URL)
	if err != nil {
		b.setState(StateDisconnected)
		return fmt.Errorf("❌ dial failed: %w", err)
	}

	// Создаём канал
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		b.setState(StateDisconnected)
		return fmt.Errorf("❌ channel creation failed: %w", err)
	}

	// Настраиваем QoS (Quality of Service) - сколько сообщений можно получать без подтверждения
	//
	// Почему это важно:
	//   - Без QoS сервер может отправить тысячи сообщений, переполнив память consumer'а
	//   - С QoS мы контролируем "окно" неподтверждённых сообщений
	//   - Значение должно быть оптимизировано под нагрузку сервиса
	if b.config.PrefetchCount > 0 {
		err = ch.Qos(
			b.config.PrefetchCount, // prefetch count
			0,                      // prefetch size (0 = без ограничений)
			false,                  // global (false = применяется к этому consumer'у)
		)
		if err != nil {
			ch.Close()
			conn.Close()
			b.setState(StateDisconnected)
			return fmt.Errorf("❌ QoS setup failed: %w", err)
		}
		log.Printf("✅ [RabbitMQ] QoS configured: prefetch_count=%d", b.config.PrefetchCount)
	}

	// Объявляем exchange (обменник)
	//
	// Почему exchange объявляется здесь:
	//   - Гарантирует, что exchange существует до первой публикации
	//   - Idempotent операция - если exchange уже есть, ничего не меняет
	//   - Параметры должны совпадать с существующими, иначе будет ошибка
	err = ch.ExchangeDeclare(
		b.config.ExchangeName, // имя
		b.config.ExchangeType, // тип
		b.config.Durable,      // durable
		false,                 // auto-deleted (не удалять при отвязке всех очередей)
		false,                 // internal (не для прямых публикаций)
		false,                 // no-wait
		nil,                   // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		b.setState(StateDisconnected)
		return fmt.Errorf("❌ exchange declare failed: %w", err)
	}

	// Включаем режим подтверждений (publisher confirms), если нужно
	//
	// Почему этот режим не включён по умолчанию:
	//   - Добавляет задержку на каждую публикацию (ждёт подтверждения от сервера)
	//   - Увеличивает надёжность, но снижает пропускную способность
	if b.config.EnableConfirmMode {
		if err := ch.Confirm(false); err != nil {
			ch.Close()
			conn.Close()
			b.setState(StateDisconnected)
			return fmt.Errorf("❌ confirm mode setup failed: %w", err)
		}

		// Канал для подтверждений (буферизированный, чтобы не блокировать публикацию)
		// Буфер 100 - компромисс: не теряем подтверждения, но и не переполняем память
		b.publishMu.Lock()
		b.confirms = ch.NotifyPublish(make(chan amqp.Confirmation, 100))
		b.publishMu.Unlock()

		log.Printf("✅ [RabbitMQ] Publisher confirms enabled")
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

// closeOldConnection закрывает существующее соединение (при переподключении)
func (b *Broker) closeOldConnection() {
	if b.conn != nil {
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
		}

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
			delay := b.config.ReconnectDelay
			for attempt := 1; attempt <= b.reconnectAttempts && attempt < 10; attempt++ {
				delay = delay * time.Duration(attempt)
				if delay > 60*time.Second {
					delay = 60 * time.Second // максимум 60 секунд
				}
			}

			log.Printf("[RabbitMQ] Reconnecting in %v (attempt %d)", delay, b.reconnectAttempts+1)

			select {
			case <-time.After(delay):
			case <-b.closeCh:
				return
			}

			// Пытаемся переподключиться
			b.reconnectAttempts++
			if err := b.connect(); err != nil {
				log.Printf("[RabbitMQ] Reconnection failed: %v, will retry", err)
				// Продолжаем цикл, notifyClose уже новый (создан в connect)
				continue
			}

			log.Printf("✅ [RabbitMQ] Reconnected successfully")
		}
	}
}

// ============================================================================
// ПУБЛИЧНЫЕ МЕТОДЫ (ПУБЛИКАЦИЯ)
// ============================================================================

// Publish публикует сообщение в RabbitMQ.
//
// Параметры:
//   - routingKey: ключ маршрутизации (зависит от типа exchange)
//   - body: тело сообщения (обычно JSON)
//   - headers: дополнительные заголовки (например, для retry)
//
// Почему метод не возвращает ошибку при включённом confirm mode:
//   - Подтверждения приходят асинхронно (см. WaitForConfirmation)
//   - Синхронное ожидание каждого подтверждения сильно снизит производительность
//   - Для гарантии доставки используйте PublishWithConfirm
func (b *Broker) Publish(routingKey string, body []byte, headers amqp.Table) error {
	return b.PublishWithContext(context.Background(), routingKey, body, headers)
}

// PublishWithContext публикует сообщение с поддержкой контекста.
//
// Поддержка контекста позволяет:
//   - Отменить публикацию при превышении timeout'а
//   - Передать trace-id для распределённой трассировки
func (b *Broker) PublishWithContext(ctx context.Context, routingKey string, body []byte, headers amqp.Table) error {
	// Проверяем состояние брокера
	if b.getState() != StateConnected {
		return ErrNotConnected
	}

	b.connMu.RLock()
	ch := b.channel
	b.connMu.RUnlock()

	if ch == nil {
		return ErrNotConnected
	}

	// Если routingKey не указан, используем из конфига
	if routingKey == "" {
		routingKey = b.config.RoutingKey
	}

	// Публикуем сообщение
	//
	// Почему publishing содержит DeliveryMode=Persistent:
	//   - Сообщение будет сохранено на диск (если очередь durable)
	//   - При перезапуске RabbitMQ сообщение не потеряется
	//   - Persistent сообщения немного медленнее, но надёжнее
	err := ch.PublishWithContext(
		ctx,
		b.config.ExchangeName,
		routingKey,
		false, // mandatory (если true - вернёт сообщение, если нет очередей)
		false, // immediate (deprecated, не используем)
		amqp.Publishing{
			ContentType:  "application/json",
			Headers:      headers,
			Body:         body,
			DeliveryMode: amqp.Persistent, // сохранять на диск
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}

	return nil
}

// PublishWithConfirm публикует сообщение и синхронно ждёт подтверждения.
// Используйте для критически важных сообщений, где недопустима потеря.
//
// Почему это медленнее:
//   - Делает round-trip к RabbitMQ и обратно
//   - Блокирует горутину до получения подтверждения
//   - Рекомендуется только для важных событий (< 1% всех сообщений)
func (b *Broker) PublishWithConfirm(ctx context.Context, routingKey string, body []byte, headers amqp.Table) error {
	// Confirm mode должен быть включён
	if !b.config.EnableConfirmMode {
		return ErrNoConfirmMode
	}

	// Публикуем сообщение
	if err := b.PublishWithContext(ctx, routingKey, body, headers); err != nil {
		return err
	}

	// Ждём подтверждение
	return b.WaitForConfirmation(ctx)
}

// WaitForConfirmation ожидает подтверждение от RabbitMQ (после публикации).
func (b *Broker) WaitForConfirmation(ctx context.Context) error {
	b.publishMu.RLock()
	confirmsCh := b.confirms
	b.publishMu.RUnlock()

	if confirmsCh == nil {
		return ErrNoConfirmMode
	}

	select {
	case confirm := <-confirmsCh:
		if !confirm.Ack {
			return fmt.Errorf("message not confirmed (nack)")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ============================================================================
// ПУБЛИЧНЫЕ МЕТОДЫ (ПОТРЕБЛЕНИЕ)
// ============================================================================

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
	q, err := ch.QueueDeclare(
		queueName,
		b.config.Durable, // durable
		false,            // auto-delete (не удалять, когда отвязались все consumers)
		false,            // exclusive (не эксклюзивная - другие сервисы могут к ней обращаться)
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		return fmt.Errorf("❌ queue declare failed: %w", err)
	}

	// Привязываем очередь к exchange с routing key по умолчанию
	err = ch.QueueBind(
		q.Name,
		b.config.RoutingKey,
		b.config.ExchangeName,
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("❌ queue bind failed: %w", err)
	}

	// Привязываем дополнительные routing key (если указаны)
	for _, binding := range bindings {
		err = ch.QueueBind(
			q.Name,
			binding,
			b.config.ExchangeName,
			false,
			nil,
		)
		if err != nil {
			log.Printf("[RabbitMQ] Warning: failed to bind %s: %v", binding, err)
		}
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

// handleError обрабатывает ошибку при обработке сообщения.
// Реализует логику повторных попыток (retry) и отправки в DLQ.
func (b *Broker) handleError(msg amqp.Delivery, handlerErr error) {
	// Получаем счётчик попыток из заголовков
	retries := 0
	if msg.Headers != nil {
		if val, ok := msg.Headers["x-retries"]; ok {
			if retriesInt, ok := val.(int32); ok {
				retries = int(retriesInt)
			} else if retriesInt, ok := val.(int64); ok {
				retries = int(retriesInt)
			} else if retriesInt, ok := val.(int); ok {
				retries = retriesInt
			}
		}
	}

	// Проверяем, не превышен ли лимит попыток
	if retries >= b.config.MaxRetries {
		log.Printf("[RabbitMQ] Max retries (%d) exceeded for message, sending to DLQ: %v",
			b.config.MaxRetries, handlerErr)

		// Отправляем в Dead Letter Queue
		if err := b.sendToDLQ(msg, handlerErr); err != nil {
			log.Printf("[RabbitMQ] Failed to send to DLQ: %v", err)
			// В случае ошибки просто nack без requeue (теряем сообщение)
			msg.Nack(false, false)
		} else {
			// Успешно отправили в DLQ - подтверждаем исходное сообщение
			msg.Ack(false)
		}
		return
	}

	// Увеличиваем счётчик попыток и отправляем на повторную обработку
	log.Printf("[RabbitMQ] Message processing failed (attempt %d/%d): %v",
		retries+1, b.config.MaxRetries, handlerErr)

	// Копируем заголовки (если есть)
	headers := make(amqp.Table)
	if msg.Headers != nil {
		for k, v := range msg.Headers {
			headers[k] = v
		}
	}
	headers["x-retries"] = retries + 1
	headers["x-last-error"] = handlerErr.Error()

	// Публикуем сообщение снова с задержкой
	//
	// Почему publish, а не nack с requeue=true:
	//   - nack с requeue не даёт возможности сделать задержку
	//   - Сообщение вернётся в начало очереди мгновенно
	//   - Наша реализация позволяет задать RetryDelay
	//   - Можно добавить заголовки (счётчик попыток, ошибку)
	err := b.Publish(
		msg.RoutingKey,
		msg.Body,
		headers,
	)

	if err != nil {
		log.Printf("❌ [RabbitMQ] Failed to republish message for retry: %v", err)
		// Если не удалось переопубликовать, nack с requeue (попробуем позже)
		msg.Nack(false, true)
	} else {
		// Успешно переопубликовали - подтверждаем исходное сообщение
		msg.Ack(false)

		// Немного ждём перед следующей попыткой
		//
		// Почему sleep здесь:
		//   - Если делать без задержки, retry цикл будет слишком быстрым
		//   - Даём время на восстановление проблемного ресурса (БД, API)
		//   - В реальном проекте лучше использовать TTL + Dead Letter
		if b.config.RetryDelay > 0 {
			time.Sleep(b.config.RetryDelay)
		}
	}
}

// sendToDLQ отправляет сообщение в Dead Letter Queue.
func (b *Broker) sendToDLQ(msg amqp.Delivery, handlerErr error) error {
	if b.config.DLQName == "" {
		// DLQ не настроен - просто отбрасываем
		return nil
	}

	// Формируем заголовки с информацией об ошибке
	headers := make(amqp.Table)
	if msg.Headers != nil {
		for k, v := range msg.Headers {
			headers[k] = v
		}
	}
	headers["x-dlq-reason"] = handlerErr.Error()
	headers["x-dlq-timestamp"] = time.Now().UTC()

	// Публикуем в DLQ
	return b.Publish(
		b.config.DLQName, // используем DLQ как routing key
		msg.Body,
		headers,
	)
}

// ============================================================================
// МЕТОДЫ УПРАВЛЕНИЯ (STATE, HEALTH, CLOSE)
// ============================================================================

// getState возвращает текущее состояние соединения (атомарно)
func (b *Broker) getState() ConnectionState {
	return ConnectionState(b.state.Load())
}

// setState устанавливает новое состояние (атомарно)
func (b *Broker) setState(state ConnectionState) {
	b.state.Store(int32(state))
}

// IsConnected проверяет, активно ли соединение с RabbitMQ.
func (b *Broker) IsConnected() bool {
	return b.getState() == StateConnected
}

// HealthCheck возвращает статус здоровья брокера.
// Используется для health check эндпоинтов.
func (b *Broker) HealthCheck() error {
	if b.getState() != StateConnected {
		return errors.New("❌ rabbitmq not connected")
	}

	b.connMu.RLock()
	defer b.connMu.RUnlock()

	if b.conn == nil || b.conn.IsClosed() {
		return errors.New("❌ rabbitmq connection is closed")
	}

	return nil
}

// Close gracefully закрывает брокер и все соединения.
//
// Почему graceful:
//  1. Перестаём принимать новые сообщения
//  2. Дожидаемся завершения обработки текущих (closeWg)
//  3. Закрываем канал confirms
//  4. Закрываем канал и соединение
func (b *Broker) Close() error {
	var closeErr error

	b.closeOnce.Do(func() {
		log.Printf("[RabbitMQ] Closing broker...")
		b.setState(StateClosing)

		// Сигналим всем горутинам о завершении
		close(b.closeCh)

		// Ждём завершения всех горутин (максимум 30 секунд)
		done := make(chan struct{})
		go func() {
			b.closeWg.Wait()
			close(done)
		}()

		select {
		case <-done:
			log.Printf("[RabbitMQ] All goroutines finished")
		case <-time.After(30 * time.Second):
			log.Printf("[RabbitMQ] Timeout waiting for goroutines to finish")
		}

		// Закрываем канал confirms
		b.publishMu.Lock()
		if b.confirms != nil {
			close(b.confirms)
		}
		b.publishMu.Unlock()

		// Закрываем канал и соединение
		b.connMu.Lock()
		defer b.connMu.Unlock()

		if b.channel != nil {
			if err := b.channel.Close(); err != nil {
				log.Printf("❌ [RabbitMQ] Error closing channel: %v", err)
				closeErr = err
			}
		}

		if b.conn != nil {
			if err := b.conn.Close(); err != nil {
				log.Printf("❌ [RabbitMQ] Error closing connection: %v", err)
				closeErr = err
			}
		}

		log.Printf("[RabbitMQ] Broker closed")
	})

	return closeErr
}

// ============================================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ============================================================================

// generateConsumerTag генерирует уникальный consumer tag.
// Используется, если в конфиге не указан.
//
// Почему нужен уникальный tag:
//   - Для отладки (видно в админке, какой инстанс потребитель)
//   - Для graceful shutdown (можно отменить конкретного consumer'а)
func generateConsumerTag() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "consumer-" + hex.EncodeToString(b)
}
