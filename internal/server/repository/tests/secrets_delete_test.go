package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/repository"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	"github.com/google/uuid"
)

// Успех
func TestSecretsRepository_DeleteSecret_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	repo := repository.NewSecretsRepository(db)

	userID := uuid.New()
	secretID := uuid.New()
	version := 2

	mock.ExpectExec(`DELETE FROM secrets`).
		WithArgs(userID, secretID, version).
		WillReturnResult(sqlmock.NewResult(0, 1)) // 1 row affected

	err = repo.DeleteSecret(context.Background(), userID, secretID, version)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Конфликт версий
func TestSecretsRepository_DeleteSecret_Conflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	repo := repository.NewSecretsRepository(db)

	userID := uuid.New()
	secretID := uuid.New()
	version := 3

	mock.ExpectExec(`DELETE FROM secrets`).
		WithArgs(userID, secretID, version).
		WillReturnResult(sqlmock.NewResult(0, 0)) // ничего не удалено

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(userID, secretID).
		WillReturnRows(
			sqlmock.NewRows([]string{"exists"}).AddRow(true),
		)

	err = repo.DeleteSecret(context.Background(), userID, secretID, version)
	if !errors.Is(err, serr.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

// Секрет не найден
func TestSecretsRepository_DeleteSecret_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	repo := repository.NewSecretsRepository(db)

	userID := uuid.New()
	secretID := uuid.New()
	version := 1

	mock.ExpectExec(`DELETE FROM secrets`).
		WithArgs(userID, secretID, version).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(userID, secretID).
		WillReturnRows(
			sqlmock.NewRows([]string{"exists"}).AddRow(false),
		)

	err = repo.DeleteSecret(context.Background(), userID, secretID, version)
	if !errors.Is(err, serr.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// Ошибка бд
func TestSecretsRepository_DeleteSecret_InternalError_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	repo := repository.NewSecretsRepository(db)

	mock.ExpectExec(`DELETE FROM secrets`).
		WillReturnError(errors.New("db error"))

	err = repo.DeleteSecret(context.Background(), uuid.New(), uuid.New(), 1)
	if !errors.Is(err, serr.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}

// Ошибка при проверки секрета на существование
func TestSecretsRepository_DeleteSecret_InternalError_Exists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	repo := repository.NewSecretsRepository(db)

	mock.ExpectExec(`DELETE FROM secrets`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnError(errors.New("db error"))

	err = repo.DeleteSecret(context.Background(), uuid.New(), uuid.New(), 1)
	if !errors.Is(err, serr.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}
