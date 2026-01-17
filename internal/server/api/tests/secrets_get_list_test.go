package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/google/uuid"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/middleware"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	repoMocks "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/mocks"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/models"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

// Нет userID в context
func TestHandler_ListSecrets_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMocks.NewMockSecretsRepo(ctrl)

	svc := service.NewSecretsService(repo, config.SecretsConfig{})
	h := api.NewHandler(&service.Services{Secrets: svc}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/secrets", nil)
	rec := httptest.NewRecorder()

	h.ListSecrets(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

// Ошибка сервера
func TestHandler_ListSecrets_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMocks.NewMockSecretsRepo(ctrl)

	svc := service.NewSecretsService(repo, config.SecretsConfig{})
	h := api.NewHandler(&service.Services{Secrets: svc}, nil, nil)

	userID := uuid.New()

	repo.EXPECT().
		ListSecrets(gomock.Any(), userID).
		Return(nil, serr.ErrInternal)

	req := httptest.NewRequest(http.MethodGet, "/secrets", nil)
	req = req.WithContext(
		middleware.ContextWithUserID(req.Context(), userID.String()),
	)

	rec := httptest.NewRecorder()
	h.ListSecrets(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// Успех
func TestHandler_ListSecrets_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repoMocks.NewMockSecretsRepo(ctrl)

	svc := service.NewSecretsService(repo, config.SecretsConfig{})
	h := api.NewHandler(&service.Services{Secrets: svc}, nil, nil)

	userID := uuid.New()

	updatedAt, _ := time.Parse(time.RFC3339, "2025-01-01T10:00:00Z")
	createdAt, _ := time.Parse(time.RFC3339, "2025-01-01T09:00:00Z")
	meta := "meta"

	expected := []models.GetAllSecretsResponse{
		{
			Type:      "text",
			Title:     "note",
			Payload:   []byte("ciphertext"),
			Meta:      &meta,
			Version:   1,
			UpdatedAt: updatedAt,
			CreatedAt: createdAt,
		},
	}

	repo.EXPECT().
		ListSecrets(gomock.Any(), userID).
		Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/secrets", nil)
	req = req.WithContext(
		middleware.ContextWithUserID(req.Context(), userID.String()),
	)

	rec := httptest.NewRecorder()
	h.ListSecrets(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	var resp []models.GetAllSecretsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(resp))
	}

	if resp[0].Title != "note" {
		t.Fatalf("unexpected title %q", resp[0].Title)
	}
}
