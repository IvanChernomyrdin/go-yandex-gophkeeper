package tests

import (
	"context"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	repoMocks "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/mocks"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// userID пустой
func TestSecretsService_DeleteSecret_UserIDEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMocks.NewMockSecretsRepo(ctrl)
	svc := service.NewSecretsService(repo, config.SecretsConfig{})

	err := svc.DeleteSecret(
		context.Background(),
		uuid.Nil,
		uuid.New(),
		1,
	)

	if err != serr.ErrUserIDEmpty {
		t.Fatalf("expected %v, got %v", serr.ErrUserIDEmpty, err)
	}
}

// ошибка репозитория
func TestSecretsService_DeleteSecret_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMocks.NewMockSecretsRepo(ctrl)
	svc := service.NewSecretsService(repo, config.SecretsConfig{})

	userID := uuid.New()
	secretID := uuid.New()

	repo.EXPECT().
		DeleteSecret(gomock.Any(), userID, secretID, 1).
		Return(serr.ErrConflict)

	err := svc.DeleteSecret(
		context.Background(),
		userID,
		secretID,
		1,
	)

	if err != serr.ErrConflict {
		t.Fatalf("expected %v, got %v", serr.ErrConflict, err)
	}
}

// успех
func TestSecretsService_DeleteSecret_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMocks.NewMockSecretsRepo(ctrl)
	svc := service.NewSecretsService(repo, config.SecretsConfig{})

	userID := uuid.New()
	secretID := uuid.New()

	repo.EXPECT().
		DeleteSecret(gomock.Any(), userID, secretID, 1).
		Return(nil)

	err := svc.DeleteSecret(
		context.Background(),
		userID,
		secretID,
		1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
