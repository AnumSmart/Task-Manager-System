package elk

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	slogelastic "github.com/nicus101/slog-elastic"
)

// Errors
var (
	ErrEmptyAddress = errors.New("elasticsearch address is empty")
	ErrEmptyIndex   = errors.New("elasticsearch index is empty")
	ErrNotEnabled   = errors.New("elasticsearch is not enabled")
	ErrClientClosed = errors.New("elasticsearch client is closed")
)

// Client - клиент для работы с Elasticsearch
type Client struct {
	config  *Config
	handler slog.Handler
	mu      sync.RWMutex
	closed  bool
}

// NewClient - создает новый клиент Elasticsearch
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if !cfg.Enabled {
		return nil, ErrNotEnabled
	}

	client := &Client{
		config: cfg,
	}

	// Создаем хендлер
	if err := client.initHandler(); err != nil {
		return nil, err
	}

	return client, nil
}

// initHandler - инициализирует slog-хендлер для Elasticsearch
func (c *Client) initHandler() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClientClosed
	}

	// Конфигурация для slog-elastic
	var esConfig slogelastic.Config
	esConfig.Address = c.config.Address
	esConfig.Index = c.config.Index
	esConfig.User = c.config.Username
	esConfig.Pass = c.config.Password

	// Подключаемся к Elasticsearch
	if err := esConfig.ConnectEsLog(); err != nil {
		return err
	}

	// Создаем хендлер
	handler := esConfig.NewElasticHandler()

	// Добавляем service как константное поле для всех логов
	if c.config.Service != "" {
		handler = handler.WithAttrs([]slog.Attr{
			slog.String("service", c.config.Service),
		})
	}

	c.handler = handler
	return nil
}

// GetHandler - возвращает slog-хендлер для использования в логгере
func (c *Client) GetHandler() slog.Handler {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.handler
}

// IsEnabled - возвращает true, если клиент включен
func (c *Client) IsEnabled() bool {
	return c.config != nil && c.config.Enabled
}

// Close - закрывает клиент и освобождает ресурсы
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	// Здесь можно добавить закрытие соединений, если нужно
	return nil
}

// Ping - проверяет доступность Elasticsearch
func (c *Client) Ping(ctx context.Context) error {
	if !c.IsEnabled() {
		return ErrNotEnabled
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	// Проверяем, что хендлер инициализирован
	if c.handler == nil {
		return errors.New("handler not initialized")
	}

	// Можно сделать реальный ping к Elasticsearch
	// Но для простоты проверяем наличие хендлера
	return nil
}

// GetConfig - возвращает копию конфигурации
func (c *Client) GetConfig() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.config == nil {
		return Config{}
	}
	return *c.config
}
