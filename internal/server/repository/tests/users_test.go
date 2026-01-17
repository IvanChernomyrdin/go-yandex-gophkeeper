package tests

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/repository"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

// Успех
func TestUsersRepository_Create_OK(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewUsersRepository(db)

	id := uuid.New()

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("test@mail.com", "hash").
		WillReturnRows(
			sqlmock.NewRows([]string{"id"}).AddRow(id),
		)

	got, err := repo.Create(context.Background(), "test@mail.com", "hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != id {
		t.Fatalf("expected %v, got %v", id, got)
	}
}

// Такой пользователь уже есть
func TestUsersRepository_Create_AlreadyExists(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewUsersRepository(db)

	pgErr := &pgconn.PgError{
		Code: "23505", // unique_violation
	}

	mock.ExpectQuery(`INSERT INTO users`).
		WillReturnError(pgErr)

	_, err := repo.Create(context.Background(), "test@mail.com", "hash")

	if err != serr.ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

// Ошибка сервера
func TestUsersRepository_Create_InternalError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewUsersRepository(db)

	mock.ExpectQuery(`INSERT INTO users`).
		WillReturnError(sql.ErrConnDone)

	_, err := repo.Create(context.Background(), "test@mail.com", "hash")

	if err != serr.ErrInternal {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}

// поиск по email
func TestUsersRepository_GetByEmail_OK(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewUsersRepository(db)

	id := uuid.New()
	hash := "hash"

	mock.ExpectQuery(`SELECT id, password_hash FROM users`).
		WithArgs("test@mail.com").
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "password_hash"}).
				AddRow(id, hash),
		)

	gotID, gotHash, err := repo.GetByEmail(context.Background(), "test@mail.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != id || gotHash != hash {
		t.Fatalf("unexpected result")
	}
}

// не найден по email
func TestUsersRepository_GetByEmail_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewUsersRepository(db)

	mock.ExpectQuery(`SELECT id, password_hash FROM users`).
		WithArgs("test@mail.com").
		WillReturnError(sql.ErrNoRows)

	_, _, err := repo.GetByEmail(context.Background(), "test@mail.com")

	if err != serr.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ошибка сервера при поиске по email
func TestUsersRepository_GetByEmail_InternalError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewUsersRepository(db)

	mock.ExpectQuery(`SELECT id, password_hash FROM users`).
		WithArgs("test@mail.com").
		WillReturnError(sql.ErrConnDone)

	_, _, err := repo.GetByEmail(context.Background(), "test@mail.com")

	if err != serr.ErrInternal {
		t.Fatalf("expected ErrInternal, got %v", err)
	}
}
