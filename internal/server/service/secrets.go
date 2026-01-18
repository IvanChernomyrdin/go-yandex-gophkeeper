package service

import (
	"context"
	"strings"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/models"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	"github.com/google/uuid"
)

// SecretsService реализует бизнес-логику работы с пользовательскими секретами.
// Сервис:
//   - валидирует входные данные;
//   - применяет политику хранения (SecretsConfig);
//   - не знает о HTTP и БД напрямую.
type SecretsService struct {
	repo   SecretsRepo
	policy config.SecretsConfig
}

// NewSecretsService создаёт новый SecretsService.
func NewSecretsService(repo SecretsRepo, cfg config.SecretsConfig) *SecretsService {
	return &SecretsService{
		repo:   repo,
		policy: cfg,
	}
}

// validateType проверяет, разрешён ли тип секрета политикой сервера.
func (s *SecretsService) validateType(t SecretType) error {
	for _, allowed := range s.policy.AllowedTypes {
		if string(t) == allowed {
			return nil
		}
	}
	return serr.ErrInvalidInput
}

// Create создаёт новый секрет пользователя.
//
// Ожидается, что payload уже зашифрован на стороне клиента.
// Сервер хранит только ciphertext.
//
// Валидации:
//   - title и payload не пустые;
//   - тип секрета разрешён политикой;
//   - размер payload и meta не превышает лимитов.
//
// Ошибки:
//   - ErrInvalidInput — невалидные данные;
//   - ErrPayloadTooLarge — превышен лимит payload;
//   - ErrInternal — ошибка хранилища.
func (s *SecretsService) Create(ctx context.Context, userID uuid.UUID, typ string, title string, payload string, meta *string) (uuid.UUID, int, time.Time, error) {
	if title == "" || payload == "" {
		return uuid.Nil, 0, time.Time{}, serr.ErrInvalidInput
	}

	st := SecretType(strings.TrimSpace(typ))
	if err := s.validateType(st); err != nil {
		return uuid.Nil, 0, time.Time{}, err
	}

	if int64(len(payload)) > s.policy.MaxPayloadBytes {
		return uuid.Nil, 0, time.Time{}, serr.ErrPayloadTooLarge
	}

	if meta != nil && int64(len(*meta)) > s.policy.MaxMetaBytes {
		return uuid.Nil, 0, time.Time{}, serr.ErrInvalidInput
	}

	return s.repo.Create(ctx, userID, st, title, payload, meta)
}

// ListSecrets возвращает список всех секретов пользователя.
//
// Метод проверяет корректность userID и делегирует получение данных
// в слой репозитория. Порядок секретов определяется реализацией
// репозитория (сортировка по updated_at DESC).
//
// Параметры:
//   - ctx — контекст запроса (для отмены, дедлайнов и трассировки)
//   - userID — идентификатор пользователя
//
// Возвращает:
//   - срез моделей models.SecretResponse при успешном выполнении
//   - serr.ErrUserIDEmpty, если userID равен uuid.Nil
//   - ошибку, полученную из слоя репозитория
func (s *SecretsService) ListSecrets(ctx context.Context, userID uuid.UUID) ([]models.SecretResponse, error) {

	if userID == uuid.Nil {
		return nil, serr.ErrUserIDEmpty
	}

	return s.repo.ListSecrets(ctx, userID)
}

// UpdateSecret обновляет секрет пользователя.
//
// Секрет определяется по userID и secretID.
// Метод использует optimistic locking (version) для предотвращения
// потери данных при конкурентных обновлениях.
//
// Метод не возвращает тело ответа — только статус выполнения операции.
//
// Возможные ошибки:
//   - ErrUserIDEmpty — userID не передан
//   - ErrNotFound    — секрет не найден
//   - ErrConflict    — конфликт версий
//   - ErrInternal    — внутренняя ошибка
func (s *SecretsService) UpdateSecret(ctx context.Context, userID uuid.UUID, secretID uuid.UUID, data models.UpdateSecretRequest) error {
	if userID == uuid.Nil {
		return serr.ErrUserIDEmpty
	}
	return s.repo.UpdateSecret(ctx, userID, secretID, data)
}

// DeleteSecret удаляет секрет пользователя с проверкой версии (optimistic locking).
//
// Метод удаляет секрет, принадлежащий пользователю userID, только если
// текущая версия секрета совпадает с переданной version.
// Если версия не совпадает, операция не выполняется и возвращается конфликт.
//
// Параметры:
//   - ctx      — контекст выполнения
//   - userID   — идентификатор пользователя (обязателен)
//   - secretID — идентификатор секрета
//   - version  — ожидаемая текущая версия секрета
//
// Возможные ошибки:
//   - ErrUserIDEmpty — если userID == uuid.Nil
//   - ErrNotFound    — если секрет не найден
//   - ErrConflict    — если версия секрета не совпадает
//   - ErrInternal    — внутренняя ошибка репозитория
//
// Успех:
//   - nil — секрет успешно удалён
func (s *SecretsService) DeleteSecret(ctx context.Context, userID uuid.UUID, secretID uuid.UUID, version int) error {
	if userID == uuid.Nil {
		return serr.ErrUserIDEmpty
	}
	return s.repo.DeleteSecret(ctx, userID, secretID, version)
}
