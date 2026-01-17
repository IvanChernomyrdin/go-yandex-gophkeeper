package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/models"
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

// ListSecrets возвращает список всех секретов пользователя.
//
// Секреты возвращаются:
//   - только для указанного userID
//   - отсортированы по updated_at в порядке убывания (сначала последние)
//
// Возвращает:
//   - []GetAllSecretsResponse — список секретов (может быть пустым)
//   - ErrInternal — при любой ошибке работы с БД
func (r *SecretsRepository) ListSecrets(ctx context.Context, userID uuid.UUID) ([]models.GetAllSecretsResponse, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT type, title, payload, meta, version, updated_at, created_at
		FROM secrets
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, serr.ErrInternal
	}
	defer rows.Close()

	var result []models.GetAllSecretsResponse

	for rows.Next() {
		var res models.GetAllSecretsResponse
		if err := rows.Scan(&res.Type, &res.Title, &res.Payload, &res.Meta, &res.Version, &res.UpdatedAt, &res.CreatedAt); err != nil {
			return nil, serr.ErrInternal
		}
		result = append(result, res)
	}
	if err := rows.Err(); err != nil {
		return nil, serr.ErrInternal
	}

	return result, nil
}
