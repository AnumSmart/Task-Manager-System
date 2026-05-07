package configs

import (
	"os"
	"time"
)

// Ошибки валидации конфигурации.
// Используются в Validate() и могут быть проверены через errors.Is().
var (
	// ErrEmptyURL возникает, если URL подключения к RabbitMQ не указан
	ErrEmptyURL = &RabbitMQConfigError{Field: "URL", Message: "RabbitMQ URL cannot be empty"}

	// ErrEmptyExchangeName возникает, если имя exchange не указано
	ErrEmptyExchangeName = &RabbitMQConfigError{Field: "ExchangeName", Message: "exchange name cannot be empty"}

	// ErrInvalidExchangeType возникает, если тип exchange не соответствует допустимым значениям
	ErrInvalidExchangeType = &RabbitMQConfigError{
		Field:   "ExchangeType",
		Message: "exchange type must be one of: direct, topic, fanout, headers",
	}

	// ErrInvalidMaxRetries возникает, если MaxRetries отрицательный
	ErrInvalidMaxRetries = &RabbitMQConfigError{Field: "MaxRetries", Message: "MaxRetries cannot be negative"}

	// ErrInvalidPrefetchCount возникает, если PrefetchCount отрицательный
	ErrInvalidPrefetchCount = &RabbitMQConfigError{Field: "PrefetchCount", Message: "PrefetchCount cannot be negative"}

	// ErrInvalidRetryDelay возникает, если RetryDelay отрицательный
	ErrInvalidRetryDelay = &RabbitMQConfigError{Field: "RetryDelay", Message: "RetryDelay cannot be negative"}
)

// ConfigError - структура для детализированных ошибок конфигурации.
// Реализует интерфейс error и содержит дополнительную информацию о причине.
//
// Преимущества использования:
//   - Явное указание поля, вызвавшего ошибку
//   - Возможность программно обработать конкретную ошибку (через errors.Is)
//   - Человекочитаемое сообщение для логов
//
// Пример обработки:
//
//	if errors.Is(err, rabbit.ErrEmptyURL) {
//	    log.Fatal("Please set RABBITMQ_URL environment variable")
//	}

type RabbitMQConfigError struct {
	Field   string // имя поля конфигурации, вызвавшего ошибку
	Message string // человекочитаемое описание ошибки
}

// Error возвращает строковое представление ошибки (реализация error interface).
// Формат: "config validation failed: {Field} - {Message}"
func (e *RabbitMQConfigError) Error() string {
	return "config validation failed: " + e.Field + " - " + e.Message
}

// Config - структура конфигурации для подключения к RabbitMQ и настройки поведения брокера.
// Все поля имеют yaml-теги для загрузки из YAML-файла через функцию LoadYAMLConfig.

type RabbitMQConfig struct {
	// ========================================================================
	// ОСНОВНЫЕ НАСТРОЙКИ ПОДКЛЮЧЕНИЯ
	// ========================================================================

	// URL - строка подключения к RabbitMQ в формате AMQP URI.
	// Поддерживаются следующие схемы:
	//   - amqp://      → обычное соединение (порт 5672)
	//   - amqps://     → TLS защищённое соединение (порт 5671)
	//
	// Формат: amqp://[user]:[password]@[host]:[port][/vhost][?query]
	//
	// Примеры:
	//   - Локальный без аутентификации:      "amqp://guest:guest@localhost:5672/"
	//   - Локальный с указанием vhost:       "amqp://guest:guest@localhost:5672/production"
	//   - В Docker Compose:                  "amqp://guest:guest@rabbitmq:5672/"
	//   - С параметрами (heartbeat=30s):     "amqp://guest:guest@localhost:5672/?heartbeat=30"
	//   - TLS соединение:                    "amqps://user:pass@rabbitmq:5671/"
	//
	// Обязательное поле. Без него соединение не будет установлено.
	// При использовании в production рекомендуется указывать через переменную окружения.
	URL string `yaml:"url"`

	// ExchangeName - имя обмена (exchange), в который будут публиковаться сообщения.
	// Exchange - это компонент RabbitMQ, отвечающий за маршрутизацию сообщений в очереди.
	//
	// Рекомендации по именованию:
	//   - Используйте формат: [приложение].[домен].[тип]
	//   - Примеры: "taskms.events", "taskms.commands", "analytics.metrics"
	//   - Для всех микросервисов используйте один exchange для событий
	//
	// Если exchange с таким именем не существует, он будет автоматически создан
	// при инициализации Broker с параметрами из ExchangeType и Durable.
	//
	// Обязательное поле.
	ExchangeName string `yaml:"exchange_name"`

	// ExchangeType - тип обмена, определяющий логику маршрутизации сообщений.
	// Поддерживаемые типы:
	//
	//   "direct" - точное совпадение routing_key.
	//       Сообщение доставляется только в очереди, у которых routing_key
	//       полностью совпадает с ключом сообщения.
	//       Пример: routing_key="task.created" → только подписчики на "task.created"
	//       Применение: команды, точечные уведомления.
	//
	//   "topic" - шаблонное совпадение с wildcard.
	//       '*' - заменяет ровно одно слово (разделённое точкой)
	//       '#' - заменяет ноль или более слов
	//       Пример: routing_key="task.*" → "task.created", "task.updated"
	//               routing_key="task.#"  → "task.created", "task.comment.added"
	//       Применение: события предметной области (рекомендуется).
	//
	//   "fanout" - широковещательная рассылка (игнорирует routing_key).
	//       Сообщение отправляется во все очереди, привязанные к exchange.
	//       Применение: уведомление всех сервисов (инвалидация кэша).
	//
	//   "headers" - маршрутизация по заголовкам сообщения.
	//       Сравниваются заголовки из сообщения с аргументами привязки.
	//       Применение: сложная маршрутизация (редко используется).
	//
	// Рекомендация: используйте "topic" для событийной архитектуры,
	// это даёт максимальную гибкость при минимальной сложности.
	ExchangeType string `yaml:"exchange_type"`

	// Durable - определяет, должны ли exchange и очередь переживать перезапуск RabbitMQ.
	//
	//   true  → метаданные exchange/очереди сохраняются на диске.
	//           После рестарта RabbitMQ exchange и очередь восстанавливаются автоматически.
	//           Сообщения, помеченные как persistent, также сохраняются.
	//
	//   false → exchange/очередь являются временными (non-durable).
	//           Пропадают при перезапуске RabbitMQ. Только для разработки/тестов.
	//
	// Критически важно для production: всегда должен быть true.
	// При изменении этого параметра для уже существующего exchange/очереди RabbitMQ
	// выбросит ошибку (нельзя изменить durable у существующего ресурса).
	//
	// Влияет на: exchange, очередь (queue). Не влияет на сами сообщения,
	// у каждого сообщения есть отдельный флаг persistent.
	Durable bool `yaml:"durable"`

	// QueueName - имя очереди для потребления сообщений.
	// Каждый микросервис (или даже экземпляр сервиса) может иметь свою очередь.
	//
	// Стратегии именования:
	//
	//   1. Одна очередь на сервис (для горизонтального масштабирования):
	//      queue_name: "notification-service"
	//      Все экземпляры notification-service конкурируют за сообщения из одной очереди.
	//      RabbitMQ распределяет сообщения между ними round-robin.
	//      Преимущество: балансировка нагрузки.
	//
	//   2. Отдельная очередь на экземпляр (fanout для каждого):
	//      queue_name: "notification-service-1", "notification-service-2"
	//      Каждый экземпляр получает копию каждого сообщения.
	//      Преимущество: каждый сервис получает все события.
	//
	//   3. С суффиксом окружения (рекомендуется для dev/prod):
	//      queue_name: "task-service-development"
	//      queue_name: "task-service-production"
	//
	// Формат рекомендации: "{service-name}-{environment}[ -{instance}]"
	// Примеры: "task-service-prod", "notifications-dev-1"
	//
	// Обязательное поле для Consumer (может быть пустым для Publisher).
	QueueName string `yaml:"queue_name"`

	// RoutingKey - ключ маршрутизации по умолчанию.
	// Используется в двух сценариях:
	//
	//   - При публикации: если вызов Publish() не указал routingKey, используется этот.
	//   - При подписке: очередь привязывается к exchange с этим routingKey.
	//
	// Формат и семантика зависят от ExchangeType:
	//
	//   direct → точное совпадение, например "task.created"
	//   topic  → паттерн, например "task.*" или "analytics.#"
	//   fanout → игнорируется (можно оставить пустым)
	//
	// Примеры topic-ключей:
	//   "task.created"       → только создание задач
	//   "task.*"             → все события задач (created, updated, deleted, assigned)
	//   "task.#"             → все события задач + подресурсов (task.comment.added)
	//   "*.created"          → все события создания в любом контексте
	//   "#"                  → все события в exchange (осторожно, много сообщений!)
	//
	// Для Consumer можно указать несколько вызовов с разными binding-ключами
	// (ConsumeWithBinding).
	RoutingKey string `yaml:"routing_key"`

	// ========================================================================
	// НАСТРОЙКИ НАДЁЖНОСТИ И RETRY (ПОВТОРНЫЕ ПОПЫТКИ)
	// ========================================================================

	// MaxRetries - максимальное количество повторных попыток обработки сообщения.
	// При исчерпании попыток сообщение отправляется в Dead Letter Queue (DLQ)
	// или отбрасывается (если DLQName не указан).
	//
	// Алгоритм работы:
	//   1. Consumer получает сообщение и вызывает handler.
	//   2. Если handler вернул ошибку → consumer читает заголовок "x-retries".
	//   3. Если retries < MaxRetries → увеличивает счётчик на 1, публикует
	//      сообщение заново через RetryDelay.
	//   4. Если retries >= MaxRetries → публикует в DLQ (или nack с requeue=false).
	//
	// Рекомендуемые значения:
	//   0 → не повторять (сразу в DLQ при ошибке)
	//   1-2 → для быстрых операций (валидация, кэш)
	//   3-5 → для стандартных операций с БД (рекомендуется)
	//   5-10 → для внешних API с возможными временными сбоями
	//
	// Важно: повторные попытки не бесконечны — это защита от "вечных" ошибок
	// (например, баг в коде, который всегда падает).
	//
	// Внимание: хрупкие операции (например, списание денег) должны быть
	// идемпотентными, так как повторная попытка может привести к дублю.
	MaxRetries int `yaml:"max_retries"`

	// RetryDelay - задержка перед повторной попыткой обработки сообщения.
	// Между неудачной попыткой и следующей выдерживается пауза.
	//
	// Зачем нужна задержка:
	//   - Даёт время восстановиться временно недоступному ресурсу (БД, API)
	//   - Предотвращает "лавину" повторных запросов при массовом сбое
	//   - Снижает нагрузку на систему в проблемные периоды
	//
	// Рекомендации:
	//   - Для быстрых retry: 1-2 секунды
	//   - Стандартное значение: 5 секунд
	//   - Для внешних API: можно увеличить до 10-30 секунд
	//
	// Для более сложной логики (экспоненциальная задержка) используйте
	// кастомный handler с time.Sleep().
	//
	// Не рекомендуется ставить менее 1 секунды — это может привести к
	// бесконечному циклу быстрых retry и перегрузке системы.
	RetryDelay time.Duration `yaml:"retry_delay"`

	// ReconnectDelay - задержка перед попыткой переподключения при потере соединения.
	// Когда соединение с RabbitMQ разорвано, брокер ждёт указанное время,
	// затем пытается восстановить соединение и канал.
	//
	// Почему нельзя переподключаться мгновенно:
	//   - RabbitMQ может быть временно недоступен (рестарт, сетевая проблема)
	//   - Мгновенные попытки создают избыточную нагрузку и спам в логах
	//
	// Алгоритм восстановления:
	//   1. Обнаружено разрыве соединения (notify close)
	//   2. Ждём ReconnectDelay
	//   3. Пытаемся переподключиться (с тем же Config)
	//   4. При успехе восстанавливаем канал, exchange, очередь, consumers
	//   5. При ошибке повторяем шаги 2-4
	//
	// Рекомендуемые значения:
	//   - Разработка/тесты: 2 секунды
	//   - Production: 5 секунд
	//
	// Внимание: во время ожидания переподключения все вызовы Publish/Consume
	// возвращают ошибку "connection lost".
	ReconnectDelay time.Duration `yaml:"reconnect_delay"`

	// DLQName - имя Dead Letter Queue, куда попадают сообщения после исчерпания всех
	// повторных попыток (MaxRetries). Отдельная очередь "мёртвых" сообщений позволяет:
	//
	//   1. Анализировать причины ошибок (логи, мониторинг)
	//   2. Повторно обработать сообщения вручную после исправления бага
	//   3. Настроить алертинг при росте DLQ
	//   4. Изучить проблемные сообщения через админку RabbitMQ
	//
	// Рекомендуемое именование: "{queue_name}.dlq" или "{service-name}.dead"
	// Примеры:
	//   - "task-service.dlq"
	//   - "notification-service-dev.dlq"
	//   - "analytics.dead"
	//
	// Стратегия работы с DLQ в production:
	//   1. Настроить мониторинг размера DLQ (Prometheus + Grafana)
	//   2. При росте DLQ → алерт для команды разработки
	//   3. После исправления бага → скрипт републикации из DLQ в основной exchange
	//
	// Если DLQName пустая строка, сообщения после всех retry отбрасываются
	// (nack с requeue=false). Используйте для неважных событий (логи, метрики).
	//
	// Для критичных событий (создание задачи, оплата) DLQ обязателен.
	DLQName string `yaml:"dlq_name"`

	// ========================================================================
	// ДОПОЛНИТЕЛЬНЫЕ НАСТРОЙКИ (PERFORMANCE & MONITORING)
	// ========================================================================

	// EnableConfirmMode - включает режим подтверждения публикации (publisher confirms).
	// При включении брокер будет ждать подтверждения от RabbitMQ, что сообщение
	// принято всеми репликами (в кластере) или сохранено на диск (для durable очередей).
	//
	// Механизм работы:
	//   1. Клиент публикует сообщение
	//   2. RabbitMQ сохраняет сообщение (при durable=true)
	//   3. RabbitMQ отправляет confirmation клиенту
	//   4. Только после confirmation считается, что сообщение доставлено надёжно
	//
	// Плюсы:
	//   + Гарантия доставки "at-least-once" (сообщение не потеряется при краше)
	//   + Возможность обработки ошибок публикации (канал закрыт, нет места на диске)
	//   + Соответствие требованиям для критичных данных
	//
	// Минусы:
	//   - Снижение пропускной способности на 10-20% (задержка на RTT)
	//   - Дополнительная нагрузка на RabbitMQ (ждёт fsync)
	//
	// Рекомендации:
	//   true → для критичных событий (task.created, payment.processed, user.registered)
	//   false → для логов, метрик, аналитики, неважных уведомлений
	//
	// Внимание: даже с confirm mode возможно дублирование сообщений при сетевых сбоях.
	// Для гарантии exactly-once нужна идемпотентность на стороне consumer.
	EnableConfirmMode bool `yaml:"enable_confirm_mode"`

	// PrefetchCount - количество сообщений, которое consumer может обрабатывать параллельно
	// без подтверждения (QoS - Quality of Service prefetch).
	// Контролирует "окно" неподтверждённых сообщений.
	//
	// Как это работает:
	//   1. Consumer получает PrefetchCount сообщений от RabbitMQ
	//   2. Обрабатывает их асинхронно
	//   3. После успешной обработки каждого отправляет ack
	//   4. Когда количество неподтверждённых становится < PrefetchCount,
	//      RabbitMQ высылает следующую порцию
	//
	// Влияние на производительность:
	//
	//   PrefetchCount=1:
	//     - Строгая последовательная обработка (одно сообщение за раз)
	//     - Минимальный риск дублей и перегрузки
	//     - Низкая пропускная способность
	//
	//   PrefetchCount=10-50 (рекомендуется):
	//     - Хороший баланс скорость/безопасность
	//     - Позволяет обрабатывать сообщения параллельно
	//     - Достаточно для большинства микросервисов
	//
	//   PrefetchCount=0 или большое число:
	//     - Без ограничений, RabbitMQ высылает все сообщения сразу
	//     - Опасно для памяти (при медленном consumer'е очередь в памяти растёт)
	//     - Только если обработка очень быстрая (микросекунды)
	//
	// Формула подбора: PrefetchCount = (желаемое кол-во параллельных задач) * 1.2
	//
	// Важно: работает только при ручных подтверждениях (auto-ack=false).
	// При auto-ack=true (не рекомендуется) этот параметр игнорируется.
	//
	// Рекомендуемые значения:
	//   - Обработка с БД или HTTP: 10-50
	//   - Тяжёлые вычисления: 1-5
	//   - Очень быстрая обработка (in-memory): 100-500
	PrefetchCount int `yaml:"prefetch_count"`

	// ConsumerTag - уникальный идентификатор потребителя в рамках канала.
	// Используется для мониторинга и отладки. Отображается в админке RabbitMQ
	// во вкладке "Connections" и "Consumers".
	//
	// Рекомендации по формированию:
	//   - В Kubernetes: consumer_tag = "{service-name}-{pod-name}"
	//   - В Docker Compose: consumer_tag = "{service-name}-{container-id}"
	//   - Для локальной разработки: consumer_tag = "{service-name}-dev"
	//
	// Примеры:
	//   - "task-service-pod-abc123"
	//   - "notification-service-container-1"
	//   - "analytics-dev"
	//
	// Если ConsumerTag пустая строка, RabbitMQ автоматически сгенерирует тег
	// (например, "amq.ctag-12345"). Это не рекомендуется для production,
	// так как усложняет мониторинг.
	//
	// ConsumerTag используется при graceful shutdown для отмены consumer
	// перед закрытием канала.
	ConsumerTag string `yaml:"consumer_tag"`
}

// NewRabbitMQConfig - конструктор для Config, возвращает значения по умолчанию.
func NewRabbitMQConfig() *RabbitMQConfig {
	return &RabbitMQConfig{
		// Основные настройки (для локальной разработки)
		URL:          "amqp://guest:guest@localhost:5672/", // локальный RabbitMQ
		ExchangeName: "taskms.events",                      // общий exchange для всех сервисов
		ExchangeType: "topic",                              // гибкая маршрутизация
		Durable:      true,                                 // сохранять метаданные на диск
		QueueName:    "",                                   // требуется явно указать для consumer
		RoutingKey:   "#",                                  // по умолчанию слушаем все события

		// Настройки надёжности (щадящие для разработки)
		MaxRetries:     3,               // достаточно для транзиентных ошибок
		RetryDelay:     2 * time.Second, // небольшая задержка для быстрых retry
		ReconnectDelay: 3 * time.Second, // умеренное ожидание переподключения
		DLQName:        "",              // в dev пусть падает сразу (легче отлаживать)

		// Дополнительные настройки (минимальное влияние на производительность)
		EnableConfirmMode: false, // отключаем для скорости в разработке
		PrefetchCount:     10,    // баланс для локального запуска
		ConsumerTag:       "",    // авто-генерация (для простоты)
	}
}

// Validate проверяет корректность конфигурации и возвращает ошибку,
// если обязательные параметры пропущены или их значения некорректны.
func (c *RabbitMQConfig) Validate() error {
	// Проверка обязательного поля URL
	if c.URL == "" {
		return ErrEmptyURL
	}

	// Проверка обязательного поля ExchangeName
	if c.ExchangeName == "" {
		return ErrEmptyExchangeName
	}

	// Проверка корректности типа exchange
	switch c.ExchangeType {
	case "direct", "topic", "fanout", "headers":
		// допустимые значения
	default:
		return ErrInvalidExchangeType
	}

	// MaxRetries не может быть отрицательным (0 допустимо)
	if c.MaxRetries < 0 {
		return ErrInvalidMaxRetries
	}

	// PrefetchCount не может быть отрицательным (0 допустимо = без ограничений)
	if c.PrefetchCount < 0 {
		return ErrInvalidPrefetchCount
	}

	// RetryDelay не может быть отрицательным (0 допустимо = без задержки)
	if c.RetryDelay < 0 {
		return ErrInvalidRetryDelay
	}

	return nil
}

// MergeWithEnv переопределяет поля Config из переменных окружения.
// Приоритет: переменные окружения имеют более высокий приоритет,
// чем значения, загруженные из YAML или дефолтные.
//
// Поддерживаемые переменные окружения:
//
//	RABBITMQ_URL            → c.URL
//	ENVIRONMENT             → добавляется как суффикс к QueueName и DLQName
//
// Специальная логика для ENVIRONMENT:
//
//	Если переменная ENVIRONMENT установлена и не пустая, она автоматически
//	добавляется как суффикс к c.QueueName и c.DLQName (если те не равны "-").
//	Это позволяет легко разделять очереди разных окружений:
//	  QueueName = "task-service" + ENVIRONMENT → "task-service-production"
//	  DLQName   = "task-service.dlq" + ENVIRONMENT → "task-service.dlq-production"
//
// Пример использования:
//
//	cfg := rabbit.NewConfig()
//	cfg.MergeWithEnv()
//	// После этого cfg.URL может быть переопределён из RABBITMQ_URL
//
// Рекомендуется вызывать после LoadYAMLConfig и перед Validate().
func (c *RabbitMQConfig) MergeWithEnv() *RabbitMQConfig {
	// Только URL подставляем из ENV (самое важное)
	if v := os.Getenv("RABBITMQ_URL"); v != "" {
		c.URL = v
	}

	// Добавляем суффикс окружения к именам очередей
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		if c.QueueName != "" && c.QueueName != "-" {
			c.QueueName = c.QueueName + "-" + env
		}
		if c.DLQName != "" && c.DLQName != "-" {
			c.DLQName = c.DLQName + "-" + env
		}
	}

	return c
}
