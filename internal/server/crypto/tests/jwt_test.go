package tests

import (
	"testing"
	"time"

	crypt "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/crypto"
	"github.com/golang-jwt/jwt/v5"
)

func TestNewAccessToken_Success(t *testing.T) {
	t.Parallel()
	cfg := crypt.JWTConfig{
		Issuer:     "gophkeeper",
		Audience:   "gophkeeper-cli",
		SigningKey: "supersecretkeysupersecretkey123456",
		AccessTTL:  5 * time.Minute,
	}

	userID := "user-123"

	tokenStr, err := crypt.NewAccessToken(userID, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	// Парсим и валидируем токен
	parsed, err := jwt.ParseWithClaims(
		tokenStr,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (any, error) {
			// Проверяем алгоритм
			if token.Method != jwt.SigningMethodHS256 {
				t.Fatalf("unexpected signing method: %v", token.Method)
			}
			return []byte(cfg.SigningKey), nil
		},
	)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	if !parsed.Valid {
		t.Fatal("token is not valid")
	}

	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok {
		t.Fatal("claims type assertion failed")
	}

	if claims.Subject != userID {
		t.Fatalf("expected subject %q, got %q", userID, claims.Subject)
	}
	if claims.Issuer != cfg.Issuer {
		t.Fatalf("expected issuer %q, got %q", cfg.Issuer, claims.Issuer)
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != cfg.Audience {
		t.Fatalf("expected audience %q, got %v", cfg.Audience, claims.Audience)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil")
	}
	if time.Until(claims.ExpiresAt.Time) <= 0 {
		t.Fatal("token already expired")
	}
}

func TestNewAccessToken_EmptySigningKey_TokenDoesNotValidateWithNonEmptyKey(t *testing.T) {
	cfg := crypt.JWTConfig{
		Issuer:     "issuer",
		Audience:   "aud",
		SigningKey: "", // пустой ключ
		AccessTTL:  time.Minute,
	}

	tokenStr, err := crypt.NewAccessToken("user", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	// Пытаемся валидировать НЕ тем ключом — должно упасть.
	parsed, err := jwt.ParseWithClaims(
		tokenStr,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (any, error) {
			return []byte("non-empty-key"), nil
		},
	)

	if err == nil && parsed != nil && parsed.Valid {
		t.Fatal("expected token to be invalid with different key")
	}
}

func TestNewAccessToken_ExpirationRespected(t *testing.T) {
	cfg := crypt.JWTConfig{
		Issuer:     "issuer",
		Audience:   "aud",
		SigningKey: "supersecretkeysupersecretkey123456",
		AccessTTL:  1 * time.Second,
	}

	tokenStr, err := crypt.NewAccessToken("user", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(2 * time.Second)

	parsed, err := jwt.ParseWithClaims(
		tokenStr,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (any, error) {
			return []byte(cfg.SigningKey), nil
		},
	)

	// jwt.ParseWithClaims вернёт ошибку по exp
	if err == nil && parsed.Valid {
		t.Fatal("expected token to be expired")
	}
}
