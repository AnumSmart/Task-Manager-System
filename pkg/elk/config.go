package elk

// Config - конфигурация для подключения к Elasticsearch
type Config struct {
	Enabled       bool   `yaml:"enabled" json:"enabled"`               // включена ли отправка
	Address       string `yaml:"address" json:"address"`               // адрес ES, например http://localhost:9200
	Index         string `yaml:"index" json:"index"`                   // имя индекса для логов
	Username      string `yaml:"username" json:"username"`             // опционально
	Password      string `yaml:"password" json:"password"`             // опционально
	MaxRetries    int    `yaml:"max_retries" json:"max_retries"`       // количество попыток при ошибке
	FlushInterval int    `yaml:"flush_interval" json:"flush_interval"` // интервал отправки в секундах
	BulkSize      int    `yaml:"bulk_size" json:"bulk_size"`           // размер батча для отправки
	Async         bool   `yaml:"async" json:"async"`                   // асинхронная отправка
	Service       string `yaml:"service" json:"service"`               // имя сервиса (добавляется в каждый лог)
}

// DefaultConfig - возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Enabled:       false,
		Address:       "http://localhost:9200",
		Index:         "app-logs",
		MaxRetries:    3,
		FlushInterval: 5,
		BulkSize:      100,
		Async:         true,
	}
}

// Validate - проверяет валидность конфигурации
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Address == "" {
		return ErrEmptyAddress
	}
	if c.Index == "" {
		return ErrEmptyIndex
	}
	return nil
}
