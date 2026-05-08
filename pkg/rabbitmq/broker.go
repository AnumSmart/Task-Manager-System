package rabbitmq

import (
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
