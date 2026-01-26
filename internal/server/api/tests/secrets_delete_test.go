package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/middleware"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	repoMocks "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/mocks"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// helper: создаёт Handler с моками SecretsService
func newTestHandlerWithSecrets(t *testing.T) (*api.Handler, *repoMocks.MockSecretsRepo) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	repo := repoMocks.NewMockSecretsRepo(ctrl)
	svc := service.NewSecretsService(repo, config.SecretsConfig{})
	handler := api.NewHandler(&service.Services{Secrets: svc}, nil, nil)

	return handler, repo
}

func TestHandler_DeleteSecret_Success(t *testing.T) {
	t.Parallel()

	h, repo := newTestHandlerWithSecrets(t)

	userID := uuid.New()
	secretID := uuid.New()

	repo.EXPECT().
		DeleteSecret(gomock.Any(), userID, secretID, 1).
		Return(nil)

	r := chi.NewRouter()
	r.Delete("/secrets/{id}", h.DeleteSecret)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/secrets/"+secretID.String()+"?version=1",
		nil,
	)
	req = req.WithContext(middleware.ContextWithUserID(req.Context(), userID))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestHandler_DeleteSecret_Unauthorized(t *testing.T) {
	t.Parallel()

	h, _ := newTestHandlerWithSecrets(t)

	r := chi.NewRouter()
	r.Delete("/secrets/{id}", h.DeleteSecret)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/secrets/"+uuid.New().String()+"?version=1",
		nil,
	)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandler_DeleteSecret_Conflict(t *testing.T) {
	t.Parallel()

	h, repo := newTestHandlerWithSecrets(t)

	userID := uuid.New()
	secretID := uuid.New()

	repo.EXPECT().
		DeleteSecret(gomock.Any(), userID, secretID, 1).
		Return(serr.ErrConflict)

	r := chi.NewRouter()
	r.Delete("/secrets/{id}", h.DeleteSecret)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/secrets/"+secretID.String()+"?version=1",
		nil,
	)
	req = req.WithContext(middleware.ContextWithUserID(req.Context(), userID))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}
