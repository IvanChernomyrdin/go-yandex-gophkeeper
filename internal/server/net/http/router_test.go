package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/google/uuid"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/crypto"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/middleware"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	svcmocks "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/mocks"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/logger"
)

func TestRouter_AuthLogin_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// --- arrange: mocks ---
	usersRepo := svcmocks.NewMockUsersRepo(ctrl)
	sessionsRepo := svcmocks.NewMockSessionsRepo(ctrl)

	// --- arrange: cfg (минимальная валидная для AuthService) ---
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

	// --- arrange: real service + handler + router ---
	authSvc := service.NewAuthService(usersRepo, sessionsRepo, cfg)
	svc := &service.Services{Auth: authSvc}

	verifier := middleware.NewJWTVerifier(cfg.Auth.JWT.SigningKey, cfg.Auth.Issuer, cfg.Auth.Audience)
	httpLogger := logger.NewHTTPLogger()

	h := api.NewHandler(svc, httpLogger, verifier)
	router := NewRouter(h)

	// --- arrange: ожидания моков ---
	email := "test@example.com"
	password := "StrongPass123"
	userID := uuid.New()

	// HashPassword должен совпасть по формату с VerifyPassword внутри сервиса.
	hash, err := crypto.HashPassword(password, crypto.Argon2Params{
		Time:      cfg.Password.Argon2.Time,
		MemoryKiB: cfg.Password.Argon2.MemoryKiB,
		Threads:   cfg.Password.Argon2.Threads,
		KeyLen:    cfg.Password.Argon2.KeyLen,
		SaltLen:   cfg.Password.Argon2.SaltLen,
	})
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	usersRepo.
		EXPECT().
		GetByEmail(gomock.Any(), email).
		DoAndReturn(func(ctx context.Context, gotEmail string) (uuid.UUID, string, error) {
			// Важно: сервис нормализует email: strings.ToLower+TrimSpace
			if gotEmail != email {
				t.Fatalf("expected email %q, got %q", email, gotEmail)
			}
			return userID, hash, nil
		})

	sessionsRepo.
		EXPECT().
		Create(gomock.Any(), userID, gomock.Any(), gomock.Any()).
		Return(uuid.New(), nil)

	// --- act ---
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// --- assert ---
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body=%q", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.AccessToken == "" {
		t.Fatalf("expected non-empty access_token")
	}
	if resp.RefreshToken == "" {
		t.Fatalf("expected non-empty refresh_token")
	}

	// Мини-проверка, что access похож на JWT (три части через точку)
	if parts := strings.Count(resp.AccessToken, "."); parts < 2 {
		t.Fatalf("access_token does not look like JWT: %q", resp.AccessToken)
	}
}
