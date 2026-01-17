package tests

import (
	"encoding/base64"
	"testing"

	crypt "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/crypto"
)

// Генерация токена
func TestNewRefreshToken_OK(t *testing.T) {
	token, err := crypt.NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken error: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Проверяем, что это валидный base64-url
	if _, err := base64.RawURLEncoding.DecodeString(token); err != nil {
		t.Fatalf("token is not valid base64-url: %v", err)
	}
}

// Уникальность токена
func TestNewRefreshToken_Unique(t *testing.T) {
	t1, _ := crypt.NewRefreshToken()
	t2, _ := crypt.NewRefreshToken()

	if t1 == t2 {
		t.Fatal("expected tokens to be unique")
	}
}

// Хэширование refresh токена
func TestHashRefreshToken_OK(t *testing.T) {
	token := "some-refresh-token"

	hash := crypt.HashRefreshToken(token)

	if len(hash) != 32 {
		t.Fatalf("expected hash length 32, got %d", len(hash))
	}
}

// Разные токены следовательно разные хэши
func TestHashRefreshToken_DifferentTokens(t *testing.T) {
	h1 := crypt.HashRefreshToken("token-1")
	h2 := crypt.HashRefreshToken("token-2")

	if string(h1) == string(h2) {
		t.Fatal("expected different hashes for different tokens")
	}
}
