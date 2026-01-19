package tests

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/repository"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

// Успех
func TestSecretsRepository_ListSecrets_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewSecretsRepository(db)

	userID := uuid.New()

	updatedAt := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	createdAt := time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
	meta := "meta"
	id := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id",
		"type",
		"title",
		"payload",
		"meta",
		"version",
		"updated_at",
		"created_at",
	}).AddRow(
		id,
		"text",
		"note",
		"ciphertext",
		meta,
		1,
		updatedAt,
		createdAt,
	)

	mock.ExpectQuery(`(?s)SELECT id, type, title, payload, meta, version, updated_at, created_at.*FROM secrets.*WHERE user_id = \$1.*ORDER BY updated_at DESC`).
		WithArgs(userID).
		WillReturnRows(rows)

	result, err := repo.ListSecrets(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(result))
	}

	got := result[0]
	if got.Type != "text" {
		t.Fatalf("unexpected type: %s", got.Type)
	}
	if got.Title != "note" {
		t.Fatalf("unexpected title: %s", got.Title)
	}
	if string(got.Payload) != "ciphertext" {
		t.Fatalf("unexpected payload: %s", got.Payload)
	}
	if got.Meta == nil || *got.Meta != "meta" {
		t.Fatalf("unexpected meta: %v", got.Meta)
	}
	if got.Version != 1 {
		t.Fatalf("unexpected version: %d", got.Version)
	}
	if !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected updated_at")
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected created_at")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// Тест ошибки БД
func TestSecretsRepository_ListSecrets_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := repository.NewSecretsRepository(db)
	userID := uuid.New()

	mock.ExpectQuery(`SELECT type, title, payload, meta, version, updated_at, created_at`).
		WithArgs(userID).
		WillReturnError(assertErr{})

	_, err = repo.ListSecrets(context.Background(), userID)
	if err != serr.ErrInternal {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "db error" }
