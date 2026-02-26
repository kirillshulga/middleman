package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"middleman/internal/repository"
)

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

func (m *TxManager) Begin(ctx context.Context) (repository.Tx, error) {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &Tx{tx: tx}, nil
}
