package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/crypto"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

type AuthService struct {
	users    UsersRepo
	sessions SessionsRepo

	pass crypto.Argon2Params
	jwt  crypto.JWTConfig

	refreshTTL     time.Duration
	rotateRefresh  bool
	reuseDetection bool
}

func NewAuthService(users UsersRepo, sessions SessionsRepo, cfg *config.Config) *AuthService {
	return &AuthService{
		users:    users,
		sessions: sessions,

		pass: crypto.Argon2Params{
			Time:      cfg.Password.Argon2.Time,
			MemoryKiB: cfg.Password.Argon2.MemoryKiB,
			Threads:   cfg.Password.Argon2.Threads,
			KeyLen:    cfg.Password.Argon2.KeyLen,
			SaltLen:   cfg.Password.Argon2.SaltLen,
		},
		jwt: crypto.JWTConfig{
			Issuer:     cfg.Auth.Issuer,
			Audience:   cfg.Auth.Audience,
			SigningKey: cfg.Auth.JWT.SigningKey,
			AccessTTL:  cfg.Auth.AccessTTL,
		},

		refreshTTL:     cfg.Auth.RefreshTTL,
		rotateRefresh:  cfg.Auth.Sessions.RotateRefresh,
		reuseDetection: cfg.Auth.Sessions.ReuseDetection,
	}
}

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) Register(ctx context.Context, email, password string) (uuid.UUID, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)

	if email == "" || password == "" || !emailRe.MatchString(email) || len(password) < 8 {
		return uuid.Nil, serr.ErrInvalidInput
	}

	hash, err := crypto.HashPassword(password, s.pass)
	if err != nil {
		return uuid.Nil, serr.ErrInternal
	}
	return s.users.Create(ctx, email, hash)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (TokenPair, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return TokenPair{}, serr.ErrInvalidInput
	}

	userID, hash, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// не палим существование email
		if errors.Is(err, serr.ErrNotFound) {
			return TokenPair{}, serr.ErrInvalidCredentials
		}
		return TokenPair{}, err
	}

	ok, err := crypto.VerifyPassword(password, hash)
	if err != nil {
		return TokenPair{}, serr.ErrInternal
	}
	if !ok {
		return TokenPair{}, serr.ErrInvalidCredentials
	}

	access, err := crypto.NewAccessToken(userID.String(), s.jwt)
	if err != nil {
		return TokenPair{}, serr.ErrInternal
	}

	refresh, err := crypto.NewRefreshToken()
	if err != nil {
		return TokenPair{}, serr.ErrInternal
	}
	refreshHash := crypto.HashRefreshToken(refresh)

	_, err = s.sessions.Create(ctx, userID, refreshHash, time.Now().Add(s.refreshTTL))
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

// Refresh обрабатывает запрос на обновление access токена по refresh токену.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return TokenPair{}, serr.ErrInvalidInput
	}

	hash := crypto.HashRefreshToken(refreshToken)

	sessID, userID, expiresAt, revokedAt, replacedBy, err := s.sessions.GetByRefreshHash(ctx, hash)
	if err != nil {
		return TokenPair{}, err // ErrUnauthorized/ErrInternal
	}

	now := time.Now()
	if expiresAt.Before(now) {
		return TokenPair{}, serr.ErrUnauthorized
	}

	// reuse detection: если токен уже отозван — значит кто-то пытается переиспользовать
	if revokedAt != nil {
		if s.reuseDetection {
			_ = s.sessions.RevokeAllForUser(ctx, userID)
		}
		return TokenPair{}, serr.ErrUnauthorized
	}

	access, err := crypto.NewAccessToken(userID.String(), s.jwt)
	if err != nil {
		return TokenPair{}, serr.ErrInternal
	}

	// если rotate_refresh выключен — возвращаем только новый access, refresh тот же
	if !s.rotateRefresh {
		return TokenPair{AccessToken: access, RefreshToken: refreshToken}, nil
	}

	// rotation: выдаём новый refresh, старый отзываем
	newRefresh, err := crypto.NewRefreshToken()
	if err != nil {
		return TokenPair{}, serr.ErrInternal
	}
	newHash := crypto.HashRefreshToken(newRefresh)

	newID, err := s.sessions.Create(ctx, userID, newHash, now.Add(s.refreshTTL))
	if err != nil {
		return TokenPair{}, err
	}

	// пометить старый как revoked и связать с новым
	if err := s.sessions.RevokeAndReplace(ctx, sessID, newID); err != nil {
		return TokenPair{}, err
	}

	_ = replacedBy // не используем, но оставляем, если захочешь логировать

	return TokenPair{AccessToken: access, RefreshToken: newRefresh}, nil
}
