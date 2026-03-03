package postgres

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupIntegrationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	if err := resetSchema(ctx, pool); err != nil {
		t.Fatalf("resetSchema() error = %v", err)
	}

	return pool
}

func resetSchema(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		return err
	}

	migrationPath, err := migrationFilePath()
	if err != nil {
		return err
	}

	sqlBytes, err := os.ReadFile(migrationPath)
	if err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
		return err
	}

	return nil
}

func migrationFilePath() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}

	dir := filepath.Dir(filename)
	return filepath.Clean(filepath.Join(dir, "../../../migrations/001_init.sql")), nil
}
