package tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	repoMocks "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/mocks"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

// UseID пустой
func TestSecretsService_ListSecrets_UserIDEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMocks.NewMockSecretsRepo(ctrl)
	svc := service.NewSecretsService(repo, config.SecretsConfig{})

	_, err := svc.ListSecrets(context.Background(), uuid.Nil)

	if err != serr.ErrUserIDEmpty {
		t.Fatalf("expected %v, got %v", serr.ErrUserIDEmpty, err)
	}
}

// Ошибка сервера
func TestSecretsService_ListSecrets_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMocks.NewMockSecretsRepo(ctrl)
	svc := service.NewSecretsService(repo, config.SecretsConfig{})

	userID := uuid.New()

	repo.EXPECT().
		ListSecrets(gomock.Any(), userID).
		Return(nil, serr.ErrInternal)

	_, err := svc.ListSecrets(context.Background(), userID)

	if err != serr.ErrInternal {
		t.Fatalf("expected %v, got %v", serr.ErrInternal, err)
	}
}

// // Успех
// func TestSecretsService_ListSecrets_Success(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	repo := repoMocks.NewMockSecretsRepo(ctrl)
// 	svc := service.NewSecretsService(repo, config.SecretsConfig{})

// 	userID := uuid.New()

// 	updatedAt := time.Now()
// 	createdAt := updatedAt.Add(-time.Hour)
// 	meta := "meta"

// 	expected := []models.Secret{
// 		{
// 			Type:      "text",
// 			Title:     "note",
// 			Payload:   "cipher",
// 			Meta:      &meta,
// 			Version:   1,
// 			UpdatedAt: updatedAt,
// 			CreatedAt: createdAt,
// 		},
// 	}

// 	repo.EXPECT().
// 		ListSecrets(gomock.Any(), userID).
// 		Return(expected, nil)

// 	result, err := svc.ListSecrets(context.Background(), userID)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}

// 	if len(result) != 1 {
// 		t.Fatalf("expected 1 secret, got %d", len(result))
// 	}

// 	if result[0].Title != "note" {
// 		t.Fatalf("unexpected title: %q", result[0].Title)
// 	}
// }
