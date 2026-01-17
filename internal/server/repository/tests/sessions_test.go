package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"

	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

// Успех
func TestSessionsRepository_Create_OK(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewSessionsRepository(db)

	userID := uuid.New()
	sessID := uuid.New()
	hash := []byte("hash")
	exp := time.Now().Add(time.Hour)

	mock.ExpectQuery(`INSERT INTO sessions`).
		WithArgs(userID, hash, exp).
		WillReturnRows(
			sqlmock.NewRows([]string{"id"}).AddRow(sessID),
		)

	id, err := repo.Create(context.Background(), userID, hash, exp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id != sessID {
		t.Fatalf("expected %v, got %v", sessID, id)
	}
}

// Конфликт
func TestSessionsRepository_Create_Conflict(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewSessionsRepository(db)

	pgErr := &pgconn.PgError{
		Code: "23505", // unique_violation
	}

	mock.ExpectQuery(`INSERT INTO sessions`).
		WillReturnError(pgErr)

	_, err := repo.Create(
		context.Background(),
		uuid.New(),
		[]byte("hash"),
		time.Now(),
	)

	if err != serr.ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

// Найден рефреш
func TestSessionsRepository_GetByRefreshHash_OK(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewSessionsRepository(db)

	sessID := uuid.New()
	userID := uuid.New()
	exp := time.Now()

	mock.ExpectQuery(`SELECT id, user_id, expires_at`).
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "user_id", "expires_at", "revoked_at", "replaced_by"},
		).AddRow(sessID, userID, exp, nil, nil))

	gotSess, gotUser, _, revoked, replaced, err :=
		repo.GetByRefreshHash(context.Background(), []byte("hash"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotSess != sessID || gotUser != userID {
		t.Fatal("unexpected ids")
	}
	if revoked != nil || replaced != nil {
		t.Fatal("expected active session")
	}
}

// Не найден рефреш
func TestSessionsRepository_GetByRefreshHash_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewSessionsRepository(db)

	mock.ExpectQuery(`SELECT id, user_id, expires_at`).
		WillReturnError(sql.ErrNoRows)

	_, _, _, _, _, err :=
		repo.GetByRefreshHash(context.Background(), []byte("x"))

	if err != serr.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

// отозван и заменён
func TestSessionsRepository_RevokeAndReplace_OK(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewSessionsRepository(db)

	mock.ExpectExec(`UPDATE sessions`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RevokeAndReplace(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// отозван для всех пользователей юзера
func TestSessionsRepository_RevokeAllForUser_OK(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	repo := repository.NewSessionsRepository(db)

	mock.ExpectExec(`UPDATE sessions`).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err := repo.RevokeAllForUser(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
