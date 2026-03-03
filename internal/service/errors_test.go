package service

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	t.Parallel()

	if !isUniqueViolation(&pgconn.PgError{Code: "23505"}) {
		t.Fatal("expected true for unique violation")
	}
	if isUniqueViolation(&pgconn.PgError{Code: "22001"}) {
		t.Fatal("expected false for non-unique violation")
	}
	if isUniqueViolation(errors.New("plain error")) {
		t.Fatal("expected false for generic error")
	}
}
