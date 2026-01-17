package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/repository"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func TestSecretsRepository_Create_OK(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repository.NewSecretsRepository(db)

	ctx := context.Background()
	userID := uuid.New()
	secretID := uuid.New()
	now := time.Now()

	meta := "meta"

	mock.ExpectQuery(`INSERT INTO secrets`).
		WithArgs(
			userID,
			string(service.SecretText),
			"title",
			[]byte("payload"),
			&meta,
		).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "version", "updated_at"}).
				AddRow(secretID, 1, now),
		)

	id, version, updatedAt, err := repo.Create(
		ctx,
		userID,
		service.SecretText,
		"title",
		"payload",
		&meta,
	)

	require.NoError(t, err)
	require.Equal(t, secretID, id)
	require.Equal(t, 1, version)
	require.WithinDuration(t, now, updatedAt, time.Second)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSecretsRepository_Create_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repository.NewSecretsRepository(db)

	ctx := context.Background()
	userID := uuid.New()

	mock.ExpectQuery(`INSERT INTO secrets`).
		WillReturnError(sql.ErrConnDone)

	id, version, updatedAt, err := repo.Create(
		ctx,
		userID,
		service.SecretText,
		"title",
		"payload",
		nil,
	)

	require.ErrorIs(t, err, serr.ErrInternal)
	require.Equal(t, uuid.Nil, id)
	require.Zero(t, version)
	require.True(t, updatedAt.IsZero())

	require.NoError(t, mock.ExpectationsWereMet())
}
