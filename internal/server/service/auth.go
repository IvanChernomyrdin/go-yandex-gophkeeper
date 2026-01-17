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

// AuthService реализует бизнес-логику аутентификации и управления сессиями.
//
// Ответственность:
//   - регистрация пользователей
//   - аутентификация (логин)
//   - выпуск access / refresh токенов
//   - обновление access токенов по refresh
//   - rotation refresh токенов
//   - reuse detection (защита от повторного использования refresh)
type AuthService struct {
	users    UsersRepo
	sessions SessionsRepo

	pass crypto.Argon2Params
	jwt  crypto.JWTConfig

	refreshTTL     time.Duration
	rotateRefresh  bool
	reuseDetection bool
}

// TokenPair представляет пару access / refresh токенов.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// NewAuthService создаёт AuthService с зависимостями и настройками из конфига.
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

// Register регистрирует нового пользователя.
//
// Валидация:
//   - email обязателен и должен быть валидным
//   - пароль обязателен и длиной >= 8 символов
//
// Возвращает:
//   - id пользователя
//   - ErrInvalidInput при некорректных данных или ErrAlreadyExists если email уже зарегистрирован
func (s *AuthService) Register(ctx context.Context, email, password string) (uuid.UUID, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)

	if email == "" || password == "" || !regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`).MatchString(email) || len(password) < 8 {
		return uuid.Nil, serr.ErrInvalidInput
	}

	hash, err := crypto.HashPassword(password, s.pass)
	if err != nil {
		return uuid.Nil, serr.ErrInternal
	}
	return s.users.Create(ctx, email, hash)
}

// Login аутентифицирует пользователя и выдаёт пару токенов.
//
// Поведение:
//   - не раскрывает факт существования email
//   - при успехе создаёт refresh-сессию
//
// Ошибки:
//   - ErrInvalidInput
//   - ErrInvalidCredentials
func (s *AuthService) Login(ctx context.Context, email, password string) (TokenPair, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return TokenPair{}, serr.ErrInvalidInput
	}
	// получаем юзера по email
	userID, hash, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// не палим существование email
		if errors.Is(err, serr.ErrNotFound) {
			return TokenPair{}, serr.ErrInvalidCredentials
		}
		return TokenPair{}, err
	}
	// проверяем пароль
	ok, err := crypto.VerifyPassword(password, hash)
	if err != nil {
		return TokenPair{}, serr.ErrInternal
	}
	if !ok {
		return TokenPair{}, serr.ErrInvalidCredentials
	}
	// создаём новый access окен
	access, err := crypto.NewAccessToken(userID.String(), s.jwt)
	if err != nil {
		return TokenPair{}, serr.ErrInternal
	}
	// создаём новый refresh токен
	refresh, err := crypto.NewRefreshToken()
	if err != nil {
		return TokenPair{}, serr.ErrInternal
	}
	// хэш для refresh токена
	refreshHash := crypto.HashRefreshToken(refresh)
	// создаём сессию
	_, err = s.sessions.Create(ctx, userID, refreshHash, time.Now().Add(s.refreshTTL))
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

// Refresh обновляет access токен по refresh токену.
//
// Поддерживает:
//   - rotation refresh токенов
//   - reuse detection (отзыв всех сессий при атаке)
//
// Ошибки:
//   - ErrInvalidInput
//   - ErrUnauthorized
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return TokenPair{}, serr.ErrInvalidInput
	}

	hash := crypto.HashRefreshToken(refreshToken)

	sessID, userID, expiresAt, revokedAt, _, err := s.sessions.GetByRefreshHash(ctx, hash) // пропускаю на что поменялась сессия т.к. логировать не собираюсь
	if err != nil {
		return TokenPair{}, err
	}

	now := time.Now()
	if expiresAt.Before(now) {
		return TokenPair{}, serr.ErrUnauthorized
	}

	// если токен уже отозван — значит кто-то пытается переиспользовать
	if revokedAt != nil {
		if s.reuseDetection {
			if err := s.sessions.RevokeAllForUser(ctx, userID); err != nil {
				return TokenPair{}, err
			}
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

	return TokenPair{AccessToken: access, RefreshToken: newRefresh}, nil
}
