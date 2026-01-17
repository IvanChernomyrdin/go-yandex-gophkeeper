package tests

import (
	"context"
	"testing"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/mocks"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	utils "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newSecretsService(t *testing.T) (*service.SecretsService, *mocks.MockSecretsRepo) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	repo := mocks.NewMockSecretsRepo(ctrl)

	cfg := config.SecretsConfig{
		MaxPayloadBytes: 1024,
		MaxMetaBytes:    256,
		AllowedTypes: []string{
			"login_password",
			"text",
			"binary",
			"bank_card",
			"otp",
		},
	}

	return service.NewSecretsService(repo, cfg), repo
}

func TestSecretsService_Create_OK(t *testing.T) {
	svc, repo := newSecretsService(t)

	ctx := context.Background()
	userID := uuid.New()
	secretID := uuid.New()
	now := time.Now()
	meta := "meta"

	repo.EXPECT().
		Create(ctx, userID, service.SecretText, "title", "payload", &meta).
		Return(secretID, 1, now, nil)

	id, version, updatedAt, err := svc.Create(
		ctx,
		userID,
		"text",
		"title",
		"payload",
		&meta,
	)

	require.NoError(t, err)
	require.Equal(t, secretID, id)
	require.Equal(t, 1, version)
	require.Equal(t, now, updatedAt)
}

func TestSecretsService_Create_InvalidType(t *testing.T) {
	svc, _ := newSecretsService(t)

	_, _, _, err := svc.Create(
		context.Background(),
		uuid.New(),
		"unknown",
		"title",
		"payload",
		nil,
	)

	require.ErrorIs(t, err, serr.ErrInvalidInput)
}

func TestSecretsService_Create_PayloadTooLarge(t *testing.T) {
	svc, _ := newSecretsService(t)

	payload := make([]byte, 2048)

	_, _, _, err := svc.Create(
		context.Background(),
		uuid.New(),
		"text",
		"title",
		string(payload),
		nil,
	)

	require.ErrorIs(t, err, serr.ErrPayloadTooLarge)
}

func TestSecretsService_Create_MetaTooLarge(t *testing.T) {
	svc, _ := newSecretsService(t)

	meta := make([]byte, 512)

	_, _, _, err := svc.Create(
		context.Background(),
		uuid.New(),
		"text",
		"title",
		"payload",
		utils.Ptr(string(meta)),
	)

	require.ErrorIs(t, err, serr.ErrInvalidInput)
}

func TestSecretsService_Create_EmptyFields(t *testing.T) {
	svc, _ := newSecretsService(t)

	_, _, _, err := svc.Create(
		context.Background(),
		uuid.New(),
		"text",
		"",
		"",
		nil,
	)

	require.ErrorIs(t, err, serr.ErrInvalidInput)
}
