// Package repository содержит реализации слоя доступа к данным (Repository layer).
//
// Репозитории инкапсулируют работу с БД и не содержат бизнес-логики.
// Все ошибки приводятся к доменным ошибкам из internal/shared/errors.
package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"

	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

// SessionsRepository отвечает за хранение и управление refresh-сессиями пользователя.
//
// Используется для:
//   - хранения refresh-токенов (в виде хэшей)
//   - реализации refresh token rotation
//   - принудительного logout со всех устройств
type SessionsRepository struct {
	db *sql.DB
}

// NewSessionsRepository создает новый SessionsRepository.
func NewSessionsRepository(db *sql.DB) *SessionsRepository {
	return &SessionsRepository{db: db}
}

// Create создает новую refresh-сессию пользователя.
//
// Сохраняет:
//   - userID
//   - хэш refresh-токена
//   - срок действия
//
// Возвращает:
//   - id созданной сессии
//   - ErrConflict при нарушении уникальности или ErrInternal при других ошибках БД
func (r *SessionsRepository) Create(ctx context.Context, userID uuid.UUID, refreshHash []byte, expiresAt time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO sessions (user_id, refresh_hash, expires_at)
		 VALUES ($1,$2,$3)
		 RETURNING id`,
		userID, refreshHash, expiresAt,
	).Scan(&id)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return uuid.Nil, serr.ErrConflict
		}
		return uuid.Nil, serr.ErrInternal
	}
	return id, nil
}

// GetByRefreshHash возвращает сессию по хэшу refresh-токена.
//
// Используется при обновлении access-токена.
//
// Возвращает:
//   - id сессии
//   - id пользователя
//   - expiresAt
//   - revokedAt (nil если активна)
//   - replacedBy (nil если не была заменена)
//
// Ошибки:
//   - ErrUnauthorized если сессия не найдена или ErrInternal при ошибке БД
func (r *SessionsRepository) GetByRefreshHash(ctx context.Context, refreshHash []byte) (uuid.UUID, uuid.UUID, time.Time, *time.Time, *uuid.UUID, error) {
	var (
		sessID    uuid.UUID
		userID    uuid.UUID
		expiresAt time.Time

		revokedAt sql.NullTime
		replaced  sql.NullString
	)

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, expires_at, revoked_at, replaced_by
		   FROM sessions
		  WHERE refresh_hash=$1`,
		refreshHash,
	).Scan(&sessID, &userID, &expiresAt, &revokedAt, &replaced)

	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, uuid.Nil, time.Time{}, nil, nil, serr.ErrUnauthorized
		}
		return uuid.Nil, uuid.Nil, time.Time{}, nil, nil, serr.ErrInternal
	}

	var revokedPtr *time.Time
	if revokedAt.Valid {
		t := revokedAt.Time
		revokedPtr = &t
	}

	var replacedPtr *uuid.UUID
	if replaced.Valid {
		if id, e := uuid.Parse(replaced.String); e == nil {
			replacedPtr = &id
		}
	}

	return sessID, userID, expiresAt, revokedPtr, replacedPtr, nil
}

// RevokeAndReplace отзывает старую refresh-сессию
// и помечает ее замененной новой.
//
// Используется для refresh token rotation.
func (r *SessionsRepository) RevokeAndReplace(ctx context.Context, oldID, newID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions
		    SET revoked_at = now(),
		        replaced_by = $2
		  WHERE id = $1
		    AND revoked_at IS NULL`,
		oldID, newID,
	)
	if err != nil {
		return serr.ErrInternal
	}
	return nil
}

// RevokeAllForUser отзывает все активные refresh-сессии пользователя.
//
// Используется при logout.
func (r *SessionsRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions
		    SET revoked_at = now()
		  WHERE user_id = $1
		    AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return serr.ErrInternal
	}
	return nil
}
