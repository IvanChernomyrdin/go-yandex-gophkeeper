package tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	repoMocks "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/mocks"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/models"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	utils "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/utils"
)

// userID пустой
func TestSecretsService_UpdateSecret_UserIDEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMocks.NewMockSecretsRepo(ctrl)
	svc := service.NewSecretsService(repo, config.SecretsConfig{})

	err := svc.UpdateSecret(
		context.Background(),
		uuid.Nil,
		uuid.New(),
		models.UpdateSecretRequest{},
	)

	if err != serr.ErrUserIDEmpty {
		t.Fatalf("expected %v, got %v", serr.ErrUserIDEmpty, err)
	}
}

// Репозиторий вернул ошибку
func TestSecretsService_UpdateSecret_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMocks.NewMockSecretsRepo(ctrl)
	svc := service.NewSecretsService(repo, config.SecretsConfig{})

	userID := uuid.New()
	secretID := uuid.New()

	req := models.UpdateSecretRequest{
		Type:    utils.StrPtr("text"),
		Title:   utils.StrPtr("note"),
		Payload: utils.StrPtr("cipher"),
		Version: 1,
	}

	repo.EXPECT().
		UpdateSecret(gomock.Any(), userID, secretID, req).
		Return(serr.ErrConflict)

	err := svc.UpdateSecret(
		context.Background(),
		userID,
		secretID,
		req,
	)

	if err != serr.ErrConflict {
		t.Fatalf("expected %v, got %v", serr.ErrConflict, err)
	}
}

// Успех
func TestSecretsService_UpdateSecret_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMocks.NewMockSecretsRepo(ctrl)
	svc := service.NewSecretsService(repo, config.SecretsConfig{})

	userID := uuid.New()
	secretID := uuid.New()
	meta := "meta"

	req := models.UpdateSecretRequest{
		Type:    utils.StrPtr("text"),
		Title:   utils.StrPtr("note"),
		Payload: utils.StrPtr("cipher"),
		Meta:    &meta,
		Version: 1,
	}

	repo.EXPECT().
		UpdateSecret(gomock.Any(), userID, secretID, req).
		Return(nil)

	err := svc.UpdateSecret(
		context.Background(),
		userID,
		secretID,
		req,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
