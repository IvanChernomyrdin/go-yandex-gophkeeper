package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/middleware"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

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
// Сервер:
//   - принимает только ciphertext (E2E, без расшифровки);
//   - валидирует тип секрета;
//   - проверяет ограничения на размер payload и meta;
//   - создаёт первую версию секрета (version = 1).
//
// Требует JWT-аутентификацию.
//
// Возможные ошибки:
//   - ErrInvalidInput — неверные поля запроса;
//   - ErrPayloadTooLarge — превышены лимиты размера;
//   - ErrUnauthorized — пользователь не аутентифицирован;
//   - ErrInternal — внутренняя ошибка сервера.
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

	// Получаем userID из JWT context
	userIDStr, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, serr.ErrUnauthorized)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
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

// ListSecrets возвращает все секреты текущего пользователя.
//
// Пользователь определяется по JWT-токену (middleware).
// Сервер не расшифровывает payload и возвращает ciphertext как есть.
//
// Возможные ошибки:
//   - 401 Unauthorized: отсутствует или некорректный JWT;
//   - 500 Internal Server Error: ошибка доступа к хранилищу.
//
// Ответ всегда возвращается в формате JSON.

// ListSecrets godoc
// @Summary      List secrets
// @Description  Returns all secrets belonging to the authenticated user.
// @Description  Payload is returned as ciphertext (E2E encryption).
// @Tags         secrets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} models.GetAllSecretsResponse
// @Failure      401 {object} api.ErrorResponse "Unauthorized"
// @Failure      500 {object} api.ErrorResponse "Internal server error"
// @Router       /secrets [get]
func (h *Handler) ListSecrets(w http.ResponseWriter, r *http.Request) {
	// Получаем userID из JWT context
	userIDStr, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, serr.ErrUnauthorized)
		return
	}
	// переводим в uuid.UUID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, serr.ErrUnauthorized)
		return
	}
	// вызываем сервис
	data, err := h.Svc.Secrets.ListSecrets(r.Context(), userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, serr.ErrInternal)
		return
	}
	w.Header().Set(ContentType, JsonContentType)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}
