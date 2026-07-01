package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"telegram-bot/internal/config"
	"telegram-bot/internal/server/handlers"
	"time"

	"github.com/gin-gonic/gin"
)

// BotGateway - HTTP сервер для ботов
type BotGateway struct {
	httpServer *http.Server                // базовый сервер из пакета http
	router     *gin.Engine                 // роутер gin
	config     *config.BotHttpServerConfig // конфиг http сервера
	botConfig  *config.BotConfig           // конфиг бота
	Handler    *handlers.BotHttpHandler    // хэндлер

	// Композиция: встраиваем управление ботом
	*PollingBotManager // ← выносим логику бота в отдельную структуру
}

// Конструктор для сервера
func NewBotGateway(ctx context.Context, config *config.BotHttpServerConfig,
	botConf *config.BotConfig,
	handler *handlers.BotHttpHandler,
) (*BotGateway, error) {

	// Создаём роутер
	router := gin.Default()
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, err
	}

	// Middleware для проброса контекста
	router.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "request_id", c.GetHeader("X-Request-ID"))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	gateway := &BotGateway{
		router:    router,
		config:    config,
		botConfig: botConf,
		Handler:   handler,
	}

	// Создаём менеджер бота (если нужен polling)
	if config.Mode == "polling" {
		botManager, err := NewPollingBotManager(botConf, handler)
		if err != nil {
			return nil, fmt.Errorf("create polling bot manager: %w", err)
		}
		gateway.PollingBotManager = botManager
	}

	return gateway, nil
}

// registerRoutes - регистрация всех маршрутов
func (a *BotGateway) registerRoutes() {
	// Health check
	a.router.GET("/health", a.healthCheck)

	// Метрики
	a.router.GET("/metrics", a.metricsHandler)

	// Админские эндпоинты
	api := a.router.Group("/api/v1")
	{
		api.GET("/status", a.getStatus)
		api.POST("/stop", a.stopBot)
		api.POST("/start", a.startBot)
	}
}

// Run - запуск сервера
func (a *BotGateway) Run() error {
	// Регистрируем маршруты
	a.registerRoutes()

	// Если режим polling - запускаем бота
	if a.config.Mode == "polling" && a.PollingBotManager != nil {
		if err := a.PollingBotManager.Start(); err != nil {
			return fmt.Errorf("start polling bot: %w", err)
		}
		log.Println("✅ Long polling бот запущен в фоновом режиме")
	}

	// Создаём и запускаем HTTP сервер
	a.httpServer = &http.Server{
		Handler: a.router,
		Addr:    a.config.Addr(),
	}

	log.Printf("🚀 Starting HTTP server on %s in %s mode", a.config.Addr(), a.config.Mode)
	return a.httpServer.ListenAndServe()
}

// Shutdown - graceful shutdown
func (a *BotGateway) Shutdown(ctx context.Context) error {
	log.Println("🛑 Начинаем graceful shutdown...")

	// 1. Останавливаем HTTP сервер
	if a.httpServer != nil {
		log.Println("⏹️ Останавливаем HTTP сервер...")
		if err := a.httpServer.Shutdown(ctx); err != nil {
			log.Printf("⚠️ HTTP server shutdown error: %v", err)
		}
	}

	// 2. Останавливаем бота (если запущен)
	if a.PollingBotManager != nil {
		log.Println("⏹️ Останавливаем Telegram бота...")
		if err := a.PollingBotManager.Stop(ctx); err != nil {
			log.Printf("⚠️ Bot stop error: %v", err)
		}
	}

	log.Println("✅ HTTP BOT Server shutdown completed")
	return nil
}

// healthCheck - проверка состояния
func (a *BotGateway) healthCheck(c *gin.Context) {
	status := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Unix(),
		"mode":   a.config.Mode,
		"bot": map[string]interface{}{
			"running": a.PollingBotManager != nil && a.PollingBotManager.IsRunning(),
		},
	}
	c.JSON(http.StatusOK, status)
}

// metricsHandler - метрики
func (a *BotGateway) metricsHandler(c *gin.Context) {
	metrics := map[string]interface{}{
		"messages_processed": 0, // TODO: добавить счетчики
		"active_sessions":    0,
		"bot_status":         a.PollingBotManager != nil && a.PollingBotManager.IsRunning(),
	}
	c.JSON(http.StatusOK, metrics)
}

// getStatus - статус бота
func (a *BotGateway) getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"mode":    a.config.Mode,
		"running": a.PollingBotManager != nil && a.PollingBotManager.IsRunning(),
	})
}

// stopBot - остановка бота (админский эндпоинт)
func (a *BotGateway) stopBot(c *gin.Context) {
	if a.PollingBotManager == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bot not initialized"})
		return
	}

	if err := a.PollingBotManager.Stop(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "bot stopped"})
}

// startBot - запуск бота (админский эндпоинт)
func (a *BotGateway) startBot(c *gin.Context) {
	if a.PollingBotManager == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bot not initialized"})
		return
	}

	if err := a.PollingBotManager.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "bot started"})
}
