package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"telegram-bot/internal/config"
	"telegram-bot/internal/server/handlers"
	"time"

	tele "gopkg.in/telebot.v4"
)

// PollingBotManager - управление long polling ботом
type PollingBotManager struct {
	telegramBot *tele.Bot                // экземпляр бота
	handler     *handlers.BotHttpHandler // хэндлер для бизнес-логики
	config      *config.BotConfig        // конфиг бота

	botWg     sync.WaitGroup     // для ожидания завершения бота
	botCtx    context.Context    // контекст для управления ботом
	botCancel context.CancelFunc // функция отмены
	stopChan  chan struct{}      // канал для остановки

	isRunning bool
	mu        sync.RWMutex
}

// NewPollingBotManager - конструктор менеджера бота
func NewPollingBotManager(
	botConfig *config.BotConfig,
	handler *handlers.BotHttpHandler,
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
	}, nil
}

// Start - запуск бота
func (m *PollingBotManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("bot already running")
	}

	// Создаём контекст для управления ботом
	m.botCtx, m.botCancel = context.WithCancel(context.Background())

	// Регистрируем обработчики
	m.registerHandlers()

	// Запускаем бота в горутине
	m.botWg.Add(1)
	go m.run()

	m.isRunning = true
	log.Println("✅ Long polling бот запущен")

	return nil
}

// Stop - остановка бота
func (m *PollingBotManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
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
	m.isRunning = false

	return nil
}

// IsRunning - проверка, запущен ли бот
func (m *PollingBotManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// registerHandlers - регистрация обработчиков бота
func (m *PollingBotManager) registerHandlers() {
	// Обработка callback-запросов от inline клавиатур
	m.telegramBot.Handle(tele.OnCallback, func(c tele.Context) error {
		return m.handler.HandleBotCallback(c)
	})

	// Обработка всех текстовых сообщений
	m.telegramBot.Handle(tele.OnText, func(c tele.Context) error {
		return m.handler.HandleBotMessage(c)
	})

	// Обработка команды /start
	m.telegramBot.Handle("/start", func(c tele.Context) error {
		return m.handler.HandleBotStart(c)
	})

	// Обработка команды /help
	m.telegramBot.Handle("/help", func(c tele.Context) error {
		return m.handler.HandleBotHelp(c)
	})

	log.Println("📝 Обработчики бота зарегистрированы")
}

// run - основной цикл бота (запускается в горутине)
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

// GetBot - получить экземпляр бота (если нужно)
func (m *PollingBotManager) GetBot() *tele.Bot {
	return m.telegramBot
}
