package service

import (
	"context"
	"strings"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
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
func (s *SecretsService) Create(
	ctx context.Context,
	userID uuid.UUID,
	typ string,
	title string,
	payload string,
	meta *string,
) (uuid.UUID, int, time.Time, error) {

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
