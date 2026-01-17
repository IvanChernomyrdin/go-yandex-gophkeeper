package tests

import (
	"testing"

	crypt "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/crypto"
)

func defaultParams() crypt.Argon2Params {
	return crypt.Argon2Params{
		Time:      1,
		MemoryKiB: 32 * 1024,
		Threads:   1,
		KeyLen:    32,
		SaltLen:   16,
	}
}

// Хэширование и успешная проверка
func TestHashAndVerifyPassword_OK(t *testing.T) {
	params := defaultParams()
	password := "super-secret-password"

	hash, err := crypt.HashPassword(password, params)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	ok, err := crypt.VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}

	if !ok {
		t.Fatal("expected password to be valid")
	}
}

// Неверный пароль
func TestVerifyPassword_InvalidPassword(t *testing.T) {
	params := defaultParams()

	hash, err := crypt.HashPassword("correct-password", params)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	ok, err := crypt.VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}

	if ok {
		t.Fatal("expected password to be invalid")
	}
}

// Пустой пароль
func TestHashPassword_EmptyPassword(t *testing.T) {
	_, err := crypt.HashPassword("", defaultParams())
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}

// Битый формат хэша
func TestVerifyPassword_InvalidFormat(t *testing.T) {
	_, err := crypt.VerifyPassword("password", "not-a-valid-hash")
	if err == nil {
		t.Fatal("expected error for invalid hash format")
	}
}

// Проверка: соль разная (хэши разные)
func TestHashPassword_DifferentSalt(t *testing.T) {
	params := defaultParams()
	password := "same-password"

	h1, _ := crypt.HashPassword(password, params)
	h2, _ := crypt.HashPassword(password, params)

	if h1 == h2 {
		t.Fatal("expected different hashes for same password")
	}
}
