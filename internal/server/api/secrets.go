package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/middleware"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/models"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	sharedModels "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Secret — swagger-схема секрета (копия sharedModels.Secret).
// Нужна, потому что swag плохо резолвит типы через alias/импорт-пакеты.
type Secret struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Payload   string    `json:"payload"`
	Meta      *string   `json:"meta,omitempty"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

// GetAllSecretsResponse — swagger-схема ответа GET /secrets.
type GetAllSecretsResponse struct {
	Secrets []Secret `json:"secrets"`
}

// UpdateSecretRequest — алиас для swagger, чтобы swag видел тип запроса.
type UpdateSecretRequest = models.UpdateSecretRequest

// CreateSecretRequest тело запроса создания секрета.
//
// Payload — это ciphertext, зашифрованный на клиенте.
// Сервер не имеет доступа к plaintext.
type CreateSecretRequest struct {
	Type    string  `json:"type"`           // login_password | text | binary | bank_card | otp
	Title   string  `json:"title"`          // произвольный заголовок секрета
	Payload string  `json:"payload"`        // ciphertext (base64 / json / etc)
	Meta    *string `json:"meta,omitempty"` // необязательные метаданные
}

// CreateSecretResponse успешный ответ создания секрета.
type CreateSecretResponse struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ErrorResponse стандартный формат ошибки API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateSecret создаёт новый секрет для аутентифицированного пользователя.
//
// @Summary      Create secret
// @Description  Creates a new secret for authenticated user. Server stores ciphertext only.
// @Tags         secrets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateSecretRequest true "Create secret request"
// @Success      201 {object} CreateSecretResponse
// @Failure      400 {object} ErrorResponse "Invalid input or bad JSON"
// @Failure      401 {object} ErrorResponse "Unauthorized"
// @Failure      413 {object} ErrorResponse "Payload too large"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /secrets [post]
func (h *Handler) CreateSecret(w http.ResponseWriter, r *http.Request) {
	var req CreateSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, serr.ErrBadJSON)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, serr.ErrUnauthorized)
		return
	}

	id, version, updatedAt, err := h.Svc.Secrets.Create(
		r.Context(),
		userID,
		req.Type,
		req.Title,
		req.Payload,
		req.Meta,
	)

	if err != nil {
		switch {
		case errors.Is(err, serr.ErrInvalidInput):
			WriteError(w, http.StatusBadRequest, err)
		case errors.Is(err, serr.ErrPayloadTooLarge):
			WriteError(w, http.StatusRequestEntityTooLarge, err)
		case errors.Is(err, serr.ErrUnauthorized):
			WriteError(w, http.StatusUnauthorized, err)
		default:
			h.Log.Logger.Sugar().Errorw(
				"create secret failed",
				"error", err,
				"user_id", userID.String(),
			)
			WriteError(w, http.StatusInternalServerError, serr.ErrInternal)
		}
		return
	}

	w.Header().Set(ContentType, JsonContentType)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateSecretResponse{
		ID:        id.String(),
		Version:   version,
		UpdatedAt: updatedAt,
	})
}

// ListSecrets godoc
// @Summary      List secrets
// @Description  Returns all secrets belonging to the authenticated user.
// @Description  Payload is returned as ciphertext (E2E encryption).
// @Tags         secrets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} GetAllSecretsResponse
// @Failure      401 {object} ErrorResponse "Unauthorized"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /secrets [get]
func (h *Handler) ListSecrets(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, serr.ErrUnauthorized)
		return
	}

	secret, err := h.Svc.Secrets.ListSecrets(r.Context(), userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, serr.ErrInternal)
		return
	}

	data := sharedModels.GetAllSecretsResponse{
		Secrets: secret,
	}

	w.Header().Set(ContentType, JsonContentType)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// UpdateSecret godoc
// @Summary      Update secret
// @Description  Updates an existing secret belonging to the authenticated user.
// @Description  Uses optimistic locking (version / updated_at check).
// @Tags         secrets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string              true  "Secret ID (UUID)"
// @Param        body  body  UpdateSecretRequest  true  "Updated secret data"
// @Success      204 "No Content"
// @Failure      400 {object} ErrorResponse "Bad request"
// @Failure      401 {object} ErrorResponse "Unauthorized"
// @Failure      404 {object} ErrorResponse "Not found"
// @Failure      409 {object} ErrorResponse "Conflict"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /secrets/{id} [put]
func (h *Handler) UpdateSecret(w http.ResponseWriter, r *http.Request) {
	secretIDStr := chi.URLParam(r, "id")
	secretID, err := uuid.Parse(secretIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, serr.ErrInvalidInput)
		return
	}

	var req UpdateSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, serr.ErrBadJSON)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, serr.ErrUnauthorized)
		return
	}

	err = h.Svc.Secrets.UpdateSecret(
		r.Context(),
		userID,
		secretID,
		req,
	)
	if err != nil {
		switch {
		case errors.Is(err, serr.ErrConflict):
			WriteError(w, http.StatusConflict, err)
		case errors.Is(err, serr.ErrNotFound):
			WriteError(w, http.StatusNotFound, err)
		default:
			h.Log.Logger.Sugar().Errorw(
				"update secret failed",
				"error", err,
				"user_id", userID.String(),
				"secret_id", secretID.String(),
			)
			WriteError(w, http.StatusInternalServerError, serr.ErrInternal)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteSecret godoc
// @Summary      Удалить секрет
// @Description  Удаляет секрет пользователя с проверкой версии (optimistic locking).
// @Description  Если версия не совпадает — возвращается конфликт.
// @Tags         secrets
// @Accept       json
// @Produce      json
// @Param        id       path     string true  "ID секрета" format(uuid)
// @Param        version  query    int    true  "Версия секрета"
// @Success      204 "Секрет успешно удалён"
// @Failure      400 {object} ErrorResponse "Некорректный ID или версия"
// @Failure      401 {object} ErrorResponse "Не авторизован"
// @Failure      404 {object} ErrorResponse "Секрет не найден"
// @Failure      409 {object} ErrorResponse "Конфликт версий"
// @Failure      500 {object} ErrorResponse "Внутренняя ошибка"
// @Security     BearerAuth
// @Router       /secrets/{id} [delete]
func (h *Handler) DeleteSecret(w http.ResponseWriter, r *http.Request) {
	secretIDStr := chi.URLParam(r, "id")
	secretID, err := uuid.Parse(secretIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, serr.ErrInvalidInput)
		return
	}

	versionStr := r.URL.Query().Get("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil || version <= 0 {
		WriteError(w, http.StatusBadRequest, serr.ErrInvalidInput)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, serr.ErrUnauthorized)
		return
	}

	err = h.Svc.Secrets.DeleteSecret(
		r.Context(),
		userID,
		secretID,
		version,
	)
	if err != nil {
		switch {
		case errors.Is(err, serr.ErrConflict):
			WriteError(w, http.StatusConflict, err)
		case errors.Is(err, serr.ErrNotFound):
			WriteError(w, http.StatusNotFound, err)
		default:
			h.Log.Logger.Sugar().Errorw(
				"delete secret failed",
				"error", err,
				"user_id", userID.String(),
				"secret_id", secretID.String(),
			)
			WriteError(w, http.StatusInternalServerError, serr.ErrInternal)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
