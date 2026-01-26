package tests

import (
	"context"
	"testing"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/mocks"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// helper: создаёт SecretsService с моками
func newTestSecretsService(t *testing.T, cfg config.SecretsConfig) (*service.SecretsService, *mocks.MockSecretsRepo) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	repo := mocks.NewMockSecretsRepo(ctrl)
	svc := service.NewSecretsService(repo, cfg)
	return svc, repo
}

// Успешное создание секрета
func TestSecretsService_Create_OK(t *testing.T) {
	t.Parallel()

	cfg := config.SecretsConfig{
		StoreCiphertext: true,
		MaxPayloadBytes: 1024,
		MaxMetaBytes:    256,
		AllowedTypes:    []string{"text", "login_password"},
	}

	svc, repo := newTestSecretsService(t, cfg)

	userID := uuid.New()
	secretID := uuid.New()
	now := time.Now()

	repo.EXPECT().
		Create(gomock.Any(), userID, service.SecretText, "My secret", "ciphertext", nil).
		Return(secretID, 1, now, nil)

	id, version, updatedAt, err := svc.Create(context.Background(), userID, "text", "My secret", "ciphertext", nil)

	require.NoError(t, err)
	require.Equal(t, secretID, id)
	require.Equal(t, 1, version)
	require.Equal(t, now, updatedAt)
}

// Payload слишком большой
func TestSecretsService_Create_PayloadTooLarge(t *testing.T) {
	t.Parallel()

	cfg := config.SecretsConfig{
		MaxPayloadBytes: 5,
		AllowedTypes:    []string{"text"},
	}

	svc, _ := newTestSecretsService(t, cfg)

	userID := uuid.New()

	_, _, _, err := svc.Create(context.Background(), userID, "text", "title", "very-long-payload", nil)

	require.ErrorIs(t, err, serr.ErrPayloadTooLarge)
}

// Недопустимый тип секрета
func TestSecretsService_Create_InvalidType(t *testing.T) {
	t.Parallel()

	cfg := config.SecretsConfig{
		AllowedTypes: []string{"text"},
	}

	svc, _ := newTestSecretsService(t, cfg)

	userID := uuid.New()

	_, _, _, err := svc.Create(context.Background(), userID, "binary", "title", "payload", nil)

	require.ErrorIs(t, err, serr.ErrInvalidInput)
}
