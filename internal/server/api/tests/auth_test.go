package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/crypto"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/middleware"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	svcmocks "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/mocks"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/logger"
)

// NewTestHandler создаёт Handler с моками и конфигом через dependency injection
func NewTestHandler(t *testing.T) (*api.Handler, *svcmocks.MockUsersRepo, *svcmocks.MockSessionsRepo) {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	users := svcmocks.NewMockUsersRepo(ctrl)
	sessions := svcmocks.NewMockSessionsRepo(ctrl)

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Issuer:     "issuer",
			Audience:   "audience",
			AccessTTL:  1 * time.Minute,
			RefreshTTL: 24 * time.Hour,
			JWT: config.JWTConfig{
				Algorithm:  "HS256",
				SigningKey: "supersecretkeysupersecretkey123456", // >= 32
			},
			Sessions: config.SessionsConfig{
				Store:              "db",
				RotateRefresh:      true,
				ReuseDetection:     true,
				MaxSessionsPerUser: 5,
			},
		},
		Password: config.PasswordConfig{
			Hasher: "argon2id",
			Argon2: config.Argon2Config{
				Time:      1,
				MemoryKiB: 64 * 1024,
				Threads:   1,
				KeyLen:    32,
				SaltLen:   16,
			},
		},
	}

	authSvc := service.NewAuthService(users, sessions, cfg)
	svc := &service.Services{Auth: authSvc}

	verifier := middleware.NewJWTVerifier(cfg.Auth.JWT.SigningKey, cfg.Auth.Issuer, cfg.Auth.Audience)
	log := logger.NewHTTPLogger()

	return api.NewHandler(svc, log, verifier), users, sessions
}

func TestHandler_Register_BadJSON(t *testing.T) {
	t.Parallel()

	h, _, _ := NewTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString("{bad json"))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatalf("expected error body, got empty")
	}
}

func TestHandler_Register_Success(t *testing.T) {
	t.Parallel()

	h, users, _ := NewTestHandler(t)

	email := "test@example.com"
	password := "StrongPass123"
	userID := uuid.New()

	users.EXPECT().
		Create(gomock.Any(), email, gomock.Any()).
		DoAndReturn(func(ctx context.Context, gotEmail, gotHash string) (uuid.UUID, error) {
			if gotEmail != email {
				t.Fatalf("expected email %q, got %q", email, gotEmail)
			}
			if gotHash == "" {
				t.Fatalf("expected non-empty password hash")
			}
			return userID, nil
		})

	body, _ := json.Marshal(api.RegisterRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set(api.ContentType, api.JsonContentType)
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d, body=%q", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var resp api.RegisterResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.UserID != userID.String() {
		t.Fatalf("expected user_id %q, got %q", userID.String(), resp.UserID)
	}
}

func TestHandler_Register_AlreadyExists(t *testing.T) {
	t.Parallel()

	h, users, _ := NewTestHandler(t)

	email := "test@example.com"
	password := "StrongPass123"

	users.EXPECT().
		Create(gomock.Any(), email, gomock.Any()).
		Return(uuid.Nil, serr.ErrAlreadyExists)

	body, _ := json.Marshal(api.RegisterRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestHandler_Login_BadJSON(t *testing.T) {
	t.Parallel()

	h, _, _ := NewTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("{bad json"))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandler_Login_Success(t *testing.T) {
	t.Parallel()

	h, users, sessions := NewTestHandler(t)

	email := "test@example.com"
	password := "StrongPass123"
	userID := uuid.New()

	hash, err := crypto.HashPassword(password, crypto.Argon2Params{
		Time:      1,
		MemoryKiB: 64 * 1024,
		Threads:   1,
		KeyLen:    32,
		SaltLen:   16,
	})
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	users.EXPECT().
		GetByEmail(gomock.Any(), email).
		Return(userID, hash, nil)

	sessions.EXPECT().
		Create(gomock.Any(), userID, gomock.Any(), gomock.Any()).
		Return(uuid.New(), nil)

	body, _ := json.Marshal(api.LoginRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set(api.ContentType, api.JsonContentType)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	var resp api.LoginResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected non-empty tokens, got %+v", resp)
	}
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	t.Parallel()

	h, users, _ := NewTestHandler(t)

	email := "test@example.com"
	password := "WrongPass123"

	users.EXPECT().
		GetByEmail(gomock.Any(), email).
		Return(uuid.Nil, "", serr.ErrNotFound)

	body, _ := json.Marshal(api.LoginRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandler_Refresh_BadJSON(t *testing.T) {
	t.Parallel()

	h, _, _ := NewTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString("{bad json"))
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandler_Refresh_Unauthorized_Expired(t *testing.T) {
	t.Parallel()

	h, _, sessions := NewTestHandler(t)

	refreshToken := "some-refresh-token"

	sessions.EXPECT().
		GetByRefreshHash(gomock.Any(), gomock.Any()).
		Return(uuid.New(), uuid.New(), time.Now().Add(-1*time.Minute), nil, nil, nil)

	body, _ := json.Marshal(api.RefreshRequest{RefreshToken: refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandler_Refresh_Success_RotateEnabled(t *testing.T) {
	t.Parallel()

	h, _, sessions := NewTestHandler(t)

	refreshToken := "some-refresh-token"
	userID := uuid.New()
	oldSessionID := uuid.New()

	sessions.EXPECT().
		GetByRefreshHash(gomock.Any(), gomock.Any()).
		Return(oldSessionID, userID, time.Now().Add(10*time.Minute), nil, nil, nil)

	newSessionID := uuid.New()
	sessions.EXPECT().
		Create(gomock.Any(), userID, gomock.Any(), gomock.Any()).
		Return(newSessionID, nil)

	sessions.EXPECT().
		RevokeAndReplace(gomock.Any(), oldSessionID, newSessionID).
		Return(nil)

	body, _ := json.Marshal(api.RefreshRequest{RefreshToken: refreshToken})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp api.RefreshResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected non-empty tokens, got %+v", resp)
	}
}
