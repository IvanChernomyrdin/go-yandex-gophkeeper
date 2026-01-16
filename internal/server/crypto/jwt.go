// Package crypto содержит криптографические примитивы,
// используемые сервером GophKeeper.
//
// В частности, пакет отвечает за:
//   - генерацию и подпись JWT access-токенов;
//   - настройку параметров токенов (issuer, audience, TTL);
//   - соблюдение требований безопасности (HS256, срок жизни).
package crypto

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig описывает параметры генерации JWT access-токена.
type JWTConfig struct {
	// Issuer — значение поля iss (кто выдал токен).
	Issuer string
	// Audience — значение поля aud (для кого предназначен токен).
	Audience string
	// SigningKey — секретный ключ для подписи токена (HS256).
	// Должен быть достаточно длинным и случайным.
	SigningKey string
	// AccessTTL — срок жизни access-токена.
	AccessTTL time.Duration
}

// NewAccessToken создаёт и подписывает JWT access-токен для пользователя.
//
// Токен содержит стандартные RegisteredClaims:
//   - iss (Issuer)
//   - aud (Audience)
//   - sub (userID)
//   - iat (IssuedAt)
//   - exp (ExpiresAt)
//
// Используется алгоритм подписи HS256.
// В случае ошибки подписи возвращается непустая ошибка.
func NewAccessToken(userID string, cfg JWTConfig) (string, error) {
	now := time.Now()

	claims := jwt.RegisteredClaims{
		Issuer:    cfg.Issuer,
		Audience:  []string{cfg.Audience},
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(cfg.AccessTTL)),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(cfg.SigningKey))
}
