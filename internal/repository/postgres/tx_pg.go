package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Tx struct {
	tx pgx.Tx
}

func (t *Tx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *Tx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}
