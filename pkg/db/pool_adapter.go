package postgresdb

import (
	"context"
	"fmt"
	"global_models/global_db"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Проверки реализации интерфейсов
var _ global_db.Pool = (*PoolAdapter)(nil)
var _ global_db.Rows = (*RowsAdapter)(nil)
var _ global_db.Row = (*RowAdapter)(nil)
var _ global_db.Tx = (*TxAdapter)(nil)

// PoolAdapter адаптирует *pgxpool.Pool к интерфейсу db.Pool
type PoolAdapter struct {
	pool *pgxpool.Pool
}

func NewPoolAdapter(pool *pgxpool.Pool) *PoolAdapter {
	return &PoolAdapter{pool: pool}
}

func (a *PoolAdapter) Close() error {
	a.pool.Close()
	return nil
}

func (a *PoolAdapter) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tag, err := a.pool.Exec(ctx, sql, args...)
	return tag.RowsAffected(), err
}

func (a *PoolAdapter) QueryRow(ctx context.Context, sql string, args ...any) global_db.Row {
	return a.pool.QueryRow(ctx, sql, args...)
}

func (a *PoolAdapter) Query(ctx context.Context, sql string, args ...any) (global_db.Rows, error) {
	rows, err := a.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &RowsAdapter{rows: rows}, nil
}

func (a *PoolAdapter) Begin(ctx context.Context) (global_db.Tx, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &TxAdapter{tx: tx}, nil
}

// RowAdapter адаптирует pgx.Row к интерфейсу db.Row
type RowAdapter struct {
	row pgx.Row
}

func (r *RowAdapter) Scan(dest ...any) error {
	return r.row.Scan(dest...)
}

// RowsAdapter адаптирует pgx.Rows к интерфейсу db.Rows
type RowsAdapter struct {
	rows pgx.Rows
}

func (r *RowsAdapter) Next() bool {
	return r.rows.Next()
}

func (r *RowsAdapter) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r *RowsAdapter) Close() {
	r.rows.Close()
}

func (r *RowsAdapter) Err() error {
	return r.rows.Err()
}

// TxAdapter адаптирует pgx.Tx к интерфейсу db.Tx
type TxAdapter struct {
	tx pgx.Tx
}

func (t *TxAdapter) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *TxAdapter) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func (t *TxAdapter) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tag, err := t.tx.Exec(ctx, sql, args...)
	return tag.RowsAffected(), err
}

func (t *TxAdapter) QueryRow(ctx context.Context, sql string, args ...any) global_db.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t *TxAdapter) Query(ctx context.Context, sql string, args ...any) (global_db.Rows, error) {
	rows, err := t.tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &RowsAdapter{rows: rows}, nil
}
