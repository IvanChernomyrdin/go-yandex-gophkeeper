package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"

	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

type UsersRepository struct {
	db *sql.DB
}

func NewUsersRepository(db *sql.DB) *UsersRepository {
	return &UsersRepository{db: db}
}

func (r *UsersRepository) Create(ctx context.Context, email, passwordHash string) (uuid.UUID, error) {
	var id uuid.UUID

	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash)
		 VALUES ($1,$2)
		 RETURNING id`,
		email, passwordHash,
	).Scan(&id)

	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" { // unique_violation
				return uuid.Nil, serr.ErrAlreadyExists
			}
		}
		return uuid.Nil, serr.ErrInternal
	}

	return id, nil
}

func (r *UsersRepository) GetByEmail(ctx context.Context, email string) (uuid.UUID, string, error) {
	var (
		id   uuid.UUID
		hash string
	)

	err := r.db.QueryRowContext(ctx,
		`SELECT id, password_hash FROM users WHERE email=$1`,
		email,
	).Scan(&id, &hash)

	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, "", serr.ErrNotFound
		}
		return uuid.Nil, "", serr.ErrInternal
	}

	return id, hash, nil
}
