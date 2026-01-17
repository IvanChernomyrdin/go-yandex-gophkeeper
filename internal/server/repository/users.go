package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"

	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

// UsersRepository предоставляет метод для создания и получения пользователей.
type UsersRepository struct {
	db *sql.DB
}

// NewUsersRepository создаёт новый UsersRepository.
//
// db — инициализированное подключение к PostgreSQL.
func NewUsersRepository(db *sql.DB) *UsersRepository {
	return &UsersRepository{db: db}
}

// Create создаёт нового пользователя.
//
// Принимает:
//   - context - контекст от родителя по которому завершиться выполнение
//   - email — уникальный email пользователя
//   - passwordHash — хэш пароля (argon2/bcrypt, без plaintext)
//
// Возвращает:
//   - id пользователя
//   - ErrAlreadyExists — если пользователь с таким email уже существует или ErrInternal — при любой другой ошибке БД
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

// GetByEmail возвращает пользователя по email.
//
// Возвращает:
//   - id пользователя
//   - password hash
//   - ErrNotFound — если пользователь не найден или ErrInternal — при ошибке БД
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
