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
	"telegram-bot/internal/server/workerpool"
	"time"

	tele "gopkg.in/telebot.v4"
)

// BotTask - адаптер для задачи бота
type BotTask struct {
	ctx     tele.Context
	handler *handlers.BotHttpHandler
}

func (t *BotTask) Process(ctx context.Context) error {
	return processUpdate(ctx, t.ctx, t.handler)
}

// PollingBotManager - управление long polling ботом
type PollingBotManager struct {
	telegramBot *tele.Bot
	handler     *handlers.BotHttpHandler
	config      *config.BotConfig

	workerPool *workerpool.WorkerPool // 👈 Воркерпул как поле

	botWg     sync.WaitGroup
	botCtx    context.Context
	botCancel context.CancelFunc
	stopChan  chan struct{}
	isRunning atomic.Bool
	mu        sync.RWMutex
}

// NewPollingBotManager - конструктор
func NewPollingBotManager(
	botConfig *config.BotConfig,
	handler *handlers.BotHttpHandler,
	numWorkers int,
	taskBuffer int,
	errBuffer int,
) (*PollingBotManager, error) {
	// Настройки бота
	pref := tele.Settings{
		Token: botConfig.BotToken,
		Poller: &tele.LongPoller{
			Timeout: 30 * time.Second,
		},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	// Создаем воркерпул
	pool := workerpool.NewWorkerPool(numWorkers, taskBuffer, errBuffer)

	// Устанавливаем обработчик ошибок (опционально)
	pool.SetErrorHandler(func(err error) {
		log.Printf("⚠️ Bot worker error: %v", err)
		// Можно добавить метрики, алерты и т.д.
	})

	return &PollingBotManager{
		telegramBot: bot,
		handler:     handler,
		config:      botConfig,
		workerPool:  pool,
		stopChan:    make(chan struct{}),
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

	// Запускаем воркерпул
	if err := m.workerPool.Start(m.botCtx); err != nil {
		return fmt.Errorf("start worker pool: %w", err)
	}

	// Регистрируем обработчики
	m.registerHandlers()

	// Запускаем бота в отдельной горутине
	m.botWg.Add(1)
	go func() {
		defer m.botWg.Done()
		log.Println("🤖 Bot started")
		m.telegramBot.Start() // Блокируется здесь
		log.Println("🛑 Bot stopped")
	}()
	m.isRunning.Store(true)

	log.Println("✅ Long polling бот запущен")

	return nil
}

// Stop - остановка бота
func (m *PollingBotManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning.Load() {
		return nil
	}

	log.Println("⏹️ Stopping bot...")

	// Останавливаем бота
	m.telegramBot.Stop() // 👈 Это заставит Start() выйти из блокировки

	// Отменяем контекст
	if m.botCancel != nil {
		m.botCancel()
	}

	// Останавливаем воркерпул
	if err := m.workerPool.Stop(ctx); err != nil {
		log.Printf("⚠️ Error stopping worker pool: %v", err)
	}
	
	// Ждем завершения всех горутин
	done := make(chan struct{})
	go func() {
		m.botWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("✅ Bot stopped gracefully")
	case <-ctx.Done():
		log.Println("⚠️ Timeout while stopping bot")
	}

	m.isRunning.Store(false)
	log.Println("✅ Bot stopped")

	return nil
}

// IsRunning - проверка статуса
func (m *PollingBotManager) IsRunning() bool {
	return m.isRunning.Load()
}

// GetBot - получить экземпляр бота
func (m *PollingBotManager) GetBot() *tele.Bot {
	return m.telegramBot
}

// processUpdate - обработка обновления (вызывается из задачи)
func processUpdate(ctx context.Context, c tele.Context, handler *handlers.BotHttpHandler) (err error) {
	// Защита от паник
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔥 PANIC in processUpdate: %v", r)
			log.Printf("Stack trace: %s", debug.Stack())

			if c != nil && c.Chat() != nil {
				c.Send("⚠️ Произошла внутренняя ошибка. Администраторы уже уведомлены.")
			}
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	if c == nil {
		return fmt.Errorf("context is nil")
	}

	// Проверяем, является ли сообщение командой
	if c.Message() != nil && c.Message().Text != "" {
		text := c.Message().Text

		switch {
		case text == "/start":
			return handler.ProcessStart(ctx, c.Chat().ID, c.Sender().ID)
		case text == "/help":
			return handler.ProcessHelp(ctx, c.Chat().ID, c.Sender().ID)
		default:
			return handler.ProcessMessage(ctx, c.Chat().ID, c.Sender().ID, text)
		}
	}

	// Проверяем callback
	if c.Callback() != nil {
		return handler.ProcessCallback(
			ctx,
			c.Chat().ID,
			c.Sender().ID,
			c.Callback().ID,
			c.Callback().Data,
			c.Callback().Message.ID,
		)
	}

	// Если ничего не подошло
	return handler.ProcessUnknown(ctx, c.Chat().ID, c.Sender().ID)
}
