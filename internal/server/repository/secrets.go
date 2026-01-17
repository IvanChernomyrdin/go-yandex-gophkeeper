package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	"github.com/google/uuid"
)

// SecretsRepository реализует доступ к хранилищу секретов (PostgreSQL).
// Отвечает исключительно за сохранение и извлечение данных без бизнес-логики.
type SecretsRepository struct {
	db *sql.DB
}

// NewSecretsRepository создаёт новый экземпляр SecretsRepository.
func NewSecretsRepository(db *sql.DB) *SecretsRepository {
	return &SecretsRepository{db: db}
}

// Create сохраняет новый секрет пользователя.
//
// Ожидается, что payload уже зашифрован на стороне клиента (E2E).
//
// Возвращает:
//   - id        — UUID созданного секрета
//   - version   — версия секрета (начиная с 1)
//   - updatedAt — время создания/обновления
//
// Ошибки:
//   - ErrInternal — ошибка базы данных
func (r *SecretsRepository) Create(
	ctx context.Context,
	userID uuid.UUID,
	typ service.SecretType,
	title string,
	payload string,
	meta *string,
) (uuid.UUID, int, time.Time, error) {

	var (
		id        uuid.UUID
		version   int
		updatedAt time.Time
	)

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO secrets (user_id, type, title, payload, meta)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, version, updated_at
	`,
		userID,
		string(typ),
		title,
		[]byte(payload),
		meta,
	).Scan(&id, &version, &updatedAt)

	if err != nil {
		return uuid.Nil, 0, time.Time{}, serr.ErrInternal
	}

	return id, version, updatedAt, nil
}
