package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/models"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	sharModels "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/models"
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
//   - []models.SecretResponse — список секретов (может быть пустым)
//   - ErrInternal — при любой ошибке работы с БД
func (r *SecretsRepository) ListSecrets(ctx context.Context, userID uuid.UUID) ([]sharModels.Secret, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, type, title, payload, meta, version, updated_at, created_at
		FROM secrets
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, serr.ErrInternal
	}
	defer rows.Close()

	var result []sharModels.Secret

	for rows.Next() {
		var res sharModels.Secret
		if err := rows.Scan(&res.ID, &res.Type, &res.Title, &res.Payload, &res.Meta, &res.Version, &res.UpdatedAt, &res.CreatedAt); err != nil {
			return nil, serr.ErrInternal
		}
		result = append(result, res)
	}
	if err := rows.Err(); err != nil {
		return nil, serr.ErrInternal
	}

	return result, nil
}

// UpdateSecret обновляет существующий секрет пользователя.
//
// Обновление выполняется по паре (userID, secretID) с использованием
// optimistic locking по полю version.
//
// Алгоритм работы:
//  1. Выполняется UPDATE с проверкой текущей версии секрета.
//  2. Если обновление прошло успешно — метод возвращает nil.
//  3. Если ни одна строка не была обновлена:
//     - проверяется существование секрета;
//     - если секрет не найден — возвращается ErrNotFound;
//     - если секрет существует, но версия отличается — ErrConflict.
//
// Метод НЕ возвращает обновлённые данные секрета,
// только сигнализирует об успехе или ошибке.
//
// Возможные ошибки:
//   - ErrNotFound  — секрет не существует или не принадлежит пользователю
//   - ErrConflict  — версия секрета устарела (обнаружен конфликт изменений)
//   - ErrInternal  — внутренняя ошибка базы данных
func (r *SecretsRepository) UpdateSecret(ctx context.Context, userID uuid.UUID, secretID uuid.UUID, data models.UpdateSecretRequest) error {
	res, err := r.db.ExecContext(ctx, `
	UPDATE secrets
	SET
		type      = COALESCE($1, type),
		title     = COALESCE($2, title),
		payload   = COALESCE($3, payload),
		meta      = COALESCE($4, meta),
		version   = version + 1,
		updated_at = now()
	WHERE user_id = $5
	  AND id = $6
	  AND version = $7
`,
		data.Type,
		data.Title,
		data.Payload,
		data.Meta,
		userID,
		secretID,
		data.Version,
	)
	if err != nil {
		return serr.ErrInternal
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return serr.ErrInternal
	}

	if affected > 0 {
		return nil
	}

	// выясняем: конфликт или not found
	var exists bool
	err = r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM secrets
			WHERE user_id = $1 AND id = $2
		)
	`, userID, secretID).Scan(&exists)

	if err != nil {
		return serr.ErrInternal
	}

	if !exists {
		return serr.ErrNotFound
	}

	return serr.ErrSecretVersionConflict
}

// DeleteSecret удаляет секрет пользователя с проверкой версии.
//
// Секрет удаляется только если существует запись с указанными:
//   - userID
//   - secretID
//   - version
//
// Если версия не совпадает, удаление не выполняется.
//
// Алгоритм:
//  1. Выполняется DELETE с проверкой version
//  2. Если ни одна строка не затронута:
//     - проверяется существование секрета
//     - если секрета нет вернётся ErrNotFound
//     - если есть, но версия не совпала вернётся ErrConflict
//
// Параметры:
//   - ctx      — контекст выполнения
//   - userID   — идентификатор пользователя
//   - secretID — идентификатор секрета
//   - version  — ожидаемая версия секрета
//
// Возможные ошибки:
//   - ErrNotFound  — секрет не найден
//   - ErrConflict  — версия не совпадает
//   - ErrInternal  — ошибка базы данных
//
// Успех:
//   - nil — секрет успешно удалён
func (r *SecretsRepository) DeleteSecret(ctx context.Context, userID uuid.UUID, secretID uuid.UUID, version int) error {
	res, err := r.db.ExecContext(ctx, `
    	DELETE FROM secrets
    	WHERE user_id = $1
    		AND id = $2
    		AND version = $3`, userID, secretID, version)

	if err != nil {
		return serr.ErrInternal
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return serr.ErrInternal
	}

	if affected > 0 {
		return nil
	}

	// различаем причину
	var exists bool
	err = r.db.QueryRowContext(ctx, `
    	SELECT EXISTS (
        SELECT 1 FROM secrets
        WHERE user_id = $1 AND id = $2)`, userID, secretID).Scan(&exists)

	if err != nil {
		return serr.ErrInternal
	}

	if !exists {
		return serr.ErrNotFound
	}

	return serr.ErrConflict
}
