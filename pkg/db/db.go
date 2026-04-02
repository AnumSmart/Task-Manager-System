package postgresdb

import (
	"context"
	"fmt"
	"global_models/global_db"

	configs "pkg/config"

	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// NewPoolWithConfig - создает pool из конфига и возвращает db.Pool (глобальный интерфейс)
// конструктор создаёт сущность на базе конфига и применяет адаптер, чтобы соответсвовать глобальному интерфейсу db.Pool
func NewPoolWithConfig(ctx context.Context, conf *configs.PostgresDBConfig) (global_db.Pool, error) {
	// 1. Парсим конфиг
	poolConfig, err := pgxpool.ParseConfig(conf.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DB DSN: %w", err)
	}

	// 2. Настраиваем pool
	poolConfig.MaxConns = conf.MaxConns
	poolConfig.MinConns = conf.MinConns
	poolConfig.HealthCheckPeriod = conf.HealthCheckPeriod
	poolConfig.MaxConnLifetime = conf.MaxConnLifetime
	poolConfig.MaxConnIdleTime = conf.MaxConnIdleTime
	poolConfig.ConnConfig.ConnectTimeout = conf.ConnectTimeout

	// 3. Создаем pool
	pool, err := pgxpool.ConnectConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// 4. Проверяем соединение
	connCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(connCtx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 5. Возвращаем адаптер как db.Pool
	return NewPoolAdapter(pool), nil
}
