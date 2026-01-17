// HTTP-хендлеры регистрации, логина, refresh токенов
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

// Каждый метод если будет возвращать ответ то будет это делать в JSON
// Вынес Content-Type и JSON для удобства
const (
	JsonContentType string = "application/json"
	ContentType     string = "Content-Type"
)

// RegisterRequest описывает тело запроса регистрации пользователя.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterResponse описывает успешный ответ регистрации.
type RegisterResponse struct {
	UserID string `json:"user_id"`
}

// LoginRequest описывает тело запроса входа пользователя.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse описывает успешный ответ входа пользователя.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshRequest описывает тело запроса обновления токенов.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshResponse описывает успешный ответ обновления токенов.
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Register обрабатывает регистрацию пользователя.
//
// @Summary      Register user
// @Description  Creates a new user account.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Register request"
// @Success      201 {object} RegisterResponse
// @Failure      400 {object} ErrorResponse "Invalid input or bad JSON"
// @Failure      409 {object} ErrorResponse "User already exists"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, serr.ErrBadJSON.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.Svc.Auth.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, serr.ErrInvalidInput):
			http.Error(w, serr.ErrInvalidInput.Error(), http.StatusBadRequest)
		case errors.Is(err, serr.ErrAlreadyExists):
			http.Error(w, serr.ErrAlreadyExists.Error(), http.StatusConflict)
		default:
			h.Log.Logger.Sugar().Error("register failed")
			http.Error(w, serr.ErrInternal.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set(ContentType, JsonContentType)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(RegisterResponse{UserID: id.String()})
}

// Login обрабатывает вход пользователя и выдачу пары токенов.
//
// @Summary      Login
// @Description  Authenticates user and returns access/refresh tokens.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login request"
// @Success      200 {object} LoginResponse
// @Failure      400 {object} ErrorResponse "Invalid input or bad JSON"
// @Failure      401 {object} ErrorResponse "Invalid credentials"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, serr.ErrBadJSON.Error(), http.StatusBadRequest)
		return
	}

	pair, err := h.Svc.Auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, serr.ErrInvalidInput):
			http.Error(w, serr.ErrInvalidInput.Error(), http.StatusBadRequest)
		case errors.Is(err, serr.ErrInvalidCredentials):
			http.Error(w, serr.ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		default:
			h.Log.Logger.Sugar().Error("login failed")
			http.Error(w, serr.ErrInternal.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set(ContentType, JsonContentType)
	json.NewEncoder(w).Encode(LoginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})
}

// Refresh обрабатывает обновление access-токена по refresh-токену.
//
// @Summary      Refresh tokens
// @Description  Rotates refresh token and returns new access token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshRequest true "Refresh request"
// @Success      200 {object} RefreshResponse
// @Failure      400 {object} ErrorResponse "Invalid input or bad JSON"
// @Failure      401 {object} ErrorResponse "Unauthorized or token revoked"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, serr.ErrBadJSON.Error(), http.StatusBadRequest)
		return
	}

	pair, err := h.Svc.Auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, serr.ErrInvalidInput):
			http.Error(w, serr.ErrInvalidInput.Error(), http.StatusBadRequest)

		case errors.Is(err, serr.ErrUnauthorized):
			http.Error(w, serr.ErrUnauthorized.Error(), http.StatusUnauthorized)
		default:
			h.Log.Logger.Sugar().Error("refresh failed")
			http.Error(w, serr.ErrInternal.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set(ContentType, JsonContentType)
	json.NewEncoder(w).Encode(RefreshResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})
}
