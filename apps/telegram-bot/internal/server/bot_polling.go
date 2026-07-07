package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"telegram-bot/internal/config"
	"telegram-bot/internal/server/handlers"
	"time"

	tele "gopkg.in/telebot.v4"
)

// PollingBotManager - управление long polling ботом.
type PollingBotManager struct {
	telegramBot *tele.Bot                // экземпляр бота
	handler     *handlers.BotHttpHandler // хэндлер для бизнес-логики
	config      *config.BotConfig        // конфиг бота

	botWg     sync.WaitGroup     // для ожидания завершения бота
	botCtx    context.Context    // контекст для управления ботом
	botCancel context.CancelFunc // функция отмены
	stopChan  chan struct{}      // канал для остановки

	isRunning atomic.Bool
	mu        sync.RWMutex

	//============== элементы workerpool ===================
	workChan   chan tele.Context // канал для задач
	errChan    chan error        // канал для ошибок при обработке обновлений
	numWorkers int               // количество воркеров
}

// NewPollingBotManager - конструктор менеджера бота.
func NewPollingBotManager(
	botConfig *config.BotConfig,
	handler *handlers.BotHttpHandler,
	numWorkers int,
	taskBuff int,
	errBuff int,
) (*PollingBotManager, error) {
	// Настройки бота
	pref := tele.Settings{
		Token: botConfig.BotToken,
		Poller: &tele.LongPoller{
			Timeout: 30 * time.Second, // интервал запросов к Telegram
		},
	}

	// Создаём бота
	bot, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	return &PollingBotManager{
		telegramBot: bot,
		handler:     handler,
		config:      botConfig,
		stopChan:    make(chan struct{}),
		workChan:    make(chan tele.Context, taskBuff), // буферизированный канал задач
		errChan:     make(chan error, errBuff),         // буферизированный канал для ошибок
		numWorkers:  numWorkers,
	}, nil
}

// Start - запуск бота.
func (m *PollingBotManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning.Load() {
		return errors.New("bot already running")
	}

	// Создаём контекст для управления ботом
	m.botCtx, m.botCancel = context.WithCancel(context.Background())

	// запускаем воркеры, которые будут обрабатывать поступающие сообщения
	for i := 0; i < m.numWorkers; i++ {
		m.botWg.Add(1)
		go m.worker()
	}

	// Регистрируем обработчики
	m.registerHandlers()

	// Запускаем бота в горутине
	m.botWg.Add(1)
	go m.run()

	// Запускаем обработчик ошибок
	m.botWg.Add(1)
	go m.errorHandler()

	m.isRunning.CompareAndSwap(false, true)

	log.Println("✅ Long polling бот запущен")

	return nil
}

// воркер для обаботки задачи
func (m *PollingBotManager) worker() {
	defer m.botWg.Done()

	for {
		select {
		case <-m.botCtx.Done():
			return
		case update := <-m.workChan:
			// обрабатываем обновление
			// тут нужно определить какой хэндлер нужно выбрать
			err := m.processUpdate(update)
			if err != nil {
				// Отправляем ошибку в канал ошибок
				select {
				case m.errChan <- err:
				case <-m.botCtx.Done():
					return
				default:
					// Если канал ошибок переполнен, логируем локально
					log.Printf("⚠️ Error channel full, logging error: %v", err)
				}
			}
		}
	}
}

func (m *PollingBotManager) errorHandler() {
	defer m.botWg.Done()

	for {
		select {
		case <-m.botCtx.Done():
			return
		case err := <-m.errChan:
			if err != nil {
				log.Printf("⚠️ Worker error: %v", err)
				// Здесь можно добавить метрики, алерты и т.д.
			}
		}
	}
}

// Stop - остановка бота.
func (m *PollingBotManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning.Load() {
		return nil
	}

	log.Println("⏹️ Останавливаем long polling бота...")

	// Отправляем сигнал остановки
	if m.botCancel != nil {
		m.botCancel()
	}

	// Ждём завершения с таймаутом
	done := make(chan struct{})

	go func() {
		m.botWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("✅ Бот остановлен")
	case <-time.After(5 * time.Second):
		log.Println("⚠️ Таймаут при остановке бота")
	case <-ctx.Done():
		log.Println("⚠️ Контекст отменен при остановке бота")
	}

	// Закрываем канал остановки
	close(m.stopChan)
	m.isRunning.CompareAndSwap(true, false)

	return nil
}

// IsRunning - проверка, запущен ли бот.
func (m *PollingBotManager) IsRunning() bool {
	return m.isRunning.Load()
}

// run - основной цикл бота (запускается в горутине).
func (m *PollingBotManager) run() {
	defer m.botWg.Done()

	log.Println("🤖 Telegram bot (long polling) started")

	// Запускаем бота. Start() блокируется, поэтому мы в горутине
	go func() {
		m.telegramBot.Start()
	}()

	// Ожидаем сигнала завершения
	select {
	case <-m.botCtx.Done():
		log.Println("📴 Получен сигнал остановки бота (контекст)")
	case <-m.stopChan:
		log.Println("📴 Получен сигнал остановки бота (канал)")
	}

	// Останавливаем бота корректно
	m.telegramBot.Stop()
	log.Println("🛑 Telegram bot (long polling) stopped")
}

// GetBot - получить экземпляр бота (если нужно).
func (m *PollingBotManager) GetBot() *tele.Bot {
	return m.telegramBot
}

func (m *PollingBotManager) processUpdate(c tele.Context) (err error) {
	// Определяем тип обновления и вызываем соответствующий метод хендлера
	// Защита от паник на уровне всего метода
	defer func() {
		if r := recover(); r != nil {
			// Логируем панику
			log.Printf("🔥 PANIC in processUpdate: %v", r)
			log.Printf("Stack trace: %s", debug.Stack())

			// Отправляем сообщение пользователю
			if c != nil && c.Chat() != nil {
				c.Send("⚠️ Произошла внутренняя ошибка. Администраторы уже уведомлены.")
			}

			// Превращаем панику в ошибку
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	// Проверяем, что контекст не nil
    if c == nil {
        return fmt.Errorf("context is nil")
    }

	// Проверяем, является ли сообщение командой
	if c.Message() != nil && c.Message().Text != "" {
		text := c.Message().Text

		// Проверяем команды
		switch {
		case text == "/start":
			return m.handler.ProcessStart(
				context.Background(),
				c.Chat().ID,
				c.Sender().ID,
			)
		case text == "/help":
			return m.handler.ProcessHelp(
				context.Background(),
				c.Chat().ID,
				c.Sender().ID,
			)
		default:
			// Обычное текстовое сообщение
			return m.handler.ProcessMessage(
				context.Background(),
				c.Chat().ID,
				c.Sender().ID,
				text,
			)
		}
	}

	// Проверяем callback
	if c.Callback() != nil {
		return m.handler.ProcessCallback(
			context.Background(),
			c.Chat().ID,
			c.Sender().ID,
			c.Callback().ID,
			c.Callback().Data,
			c.Callback().Message.ID,
		)
	}

	// Если ничего не подошло
	return m.handler.ProcessUnknown(
		context.Background(),
		c.Chat().ID,
		c.Sender().ID,
	)
}
