package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/crypto"
	crypt "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/crypto"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service/mocks"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

// созда1м сервис
func newAuthService(t *testing.T) (*service.AuthService, *mocks.MockUsersRepo, *mocks.MockSessionsRepo) {
	t.Helper()

	ctrl := gomock.NewController(t)

	users := mocks.NewMockUsersRepo(ctrl)
	sessions := mocks.NewMockSessionsRepo(ctrl)

	cfg := testConfig() // ниже покажу

	svc := service.NewAuthService(users, sessions, cfg)
	return svc, users, sessions
}

// Успех
func TestAuthService_Login_OK(t *testing.T) {
	ctx := context.Background()
	svc, users, sessions := newAuthService(t)

	userID := uuid.New()
	password := "strongpassword"

	params := crypt.Argon2Params{
		Time:      testConfig().Password.Argon2.Time,
		MemoryKiB: testConfig().Password.Argon2.MemoryKiB,
		Threads:   testConfig().Password.Argon2.Threads,
		KeyLen:    testConfig().Password.Argon2.KeyLen,
		SaltLen:   testConfig().Password.Argon2.SaltLen,
	}

	hash, err := crypt.HashPassword(password, params)
	require.NoError(t, err)

	users.EXPECT().
		GetByEmail(ctx, "test@mail.com").
		Return(userID, hash, nil)

	sessions.EXPECT().
		Create(gomock.Any(), userID, gomock.Any(), gomock.Any()).
		Return(uuid.New(), nil)

	tokens, err := svc.Login(ctx, "test@mail.com", password)

	require.NoError(t, err)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
}

// Неверный пароль
func TestAuthService_Login_InvalidPassword(t *testing.T) {
	ctx := context.Background()
	svc, users, _ := newAuthService(t)

	userID := uuid.New()

	// используем те же Argon2 параметры, что и сервис
	cfg := testConfig()
	params := crypto.Argon2Params{
		Time:      cfg.Password.Argon2.Time,
		MemoryKiB: cfg.Password.Argon2.MemoryKiB,
		Threads:   cfg.Password.Argon2.Threads,
		KeyLen:    cfg.Password.Argon2.KeyLen,
		SaltLen:   cfg.Password.Argon2.SaltLen,
	}

	// хешируем ПРАВИЛЬНЫЙ пароль
	hash, err := crypt.HashPassword("correct-password", params)
	require.NoError(t, err)

	users.EXPECT().
		GetByEmail(ctx, "test@mail.com").
		Return(userID, hash, nil)

	// пробуем войти с НЕПРАВИЛЬНЫМ паролем
	_, err = svc.Login(ctx, "test@mail.com", "wrong-password")

	require.ErrorIs(t, err, serr.ErrInvalidCredentials)
}

// Email не существует
func TestAuthService_Login_EmailNotFound(t *testing.T) {
	ctx := context.Background()
	svc, users, _ := newAuthService(t)

	users.EXPECT().
		GetByEmail(ctx, "test@mail.com").
		Return(uuid.Nil, "", serr.ErrNotFound)

	_, err := svc.Login(ctx, "test@mail.com", "password")

	require.ErrorIs(t, err, serr.ErrInvalidCredentials)
}

// Refresh успешная ротация
func TestAuthService_Refresh_Rotate_OK(t *testing.T) {
	ctx := context.Background()
	svc, _, sessions := newAuthService(t)

	oldSessID := uuid.New()
	newSessID := uuid.New()
	userID := uuid.New()

	expires := time.Now().Add(time.Hour)

	sessions.EXPECT().
		GetByRefreshHash(ctx, gomock.Any()).
		Return(oldSessID, userID, expires, nil, nil, nil)

	sessions.EXPECT().
		Create(ctx, userID, gomock.Any(), gomock.Any()).
		Return(newSessID, nil)

	sessions.EXPECT().
		RevokeAndReplace(ctx, oldSessID, newSessID).
		Return(nil)

	tokens, err := svc.Refresh(ctx, "refresh-token")

	require.NoError(t, err)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
}

// Повторное использование refresh
func TestAuthService_Refresh_ReusedToken(t *testing.T) {
	ctx := context.Background()
	svc, _, sessions := newAuthService(t)

	userID := uuid.New()
	now := time.Now()
	revoked := now.Add(-time.Minute)

	sessions.EXPECT().
		GetByRefreshHash(ctx, gomock.Any()).
		Return(uuid.New(), userID, now.Add(time.Hour), &revoked, nil, nil)

	sessions.EXPECT().
		RevokeAllForUser(ctx, userID).
		Return(nil)

	_, err := svc.Refresh(ctx, "reused-token")

	require.ErrorIs(t, err, serr.ErrUnauthorized)
}

// Тестовый конфиг
func testConfig() *config.Config {
	return &config.Config{
		Auth: config.AuthConfig{
			Issuer:     "test",
			Audience:   "test",
			AccessTTL:  time.Minute,
			RefreshTTL: time.Hour,
			Sessions: config.SessionsConfig{
				RotateRefresh:  true,
				ReuseDetection: true,
			},
			JWT: config.JWTConfig{
				SigningKey: "secret",
			},
		},
		Password: config.PasswordConfig{
			Argon2: config.Argon2Config{
				Time:      1,
				MemoryKiB: 64 * 1024,
				Threads:   1,
				KeyLen:    32,
				SaltLen:   16,
			},
		},
	}
}
