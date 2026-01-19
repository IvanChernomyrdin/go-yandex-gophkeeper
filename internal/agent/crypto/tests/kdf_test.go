package tests

import (
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/crypto"
)

func TestDefaultKDFParams_HasExpectedSizes(t *testing.T) {
	p := crypto.DefaultKDFParams()
	if p.SaltLen != crypto.SaltSize {
		t.Fatalf("expected SaltLen=%d, got %d", crypto.SaltSize, p.SaltLen)
	}
	if p.KeyLen != crypto.KeySize {
		t.Fatalf("expected KeyLen=%d, got %d", crypto.KeySize, p.KeyLen)
	}
	if p.Time == 0 || p.Memory == 0 || p.Threads == 0 {
		t.Fatalf("expected non-zero KDF params, got %+v", p)
	}
}

func TestNewSalt_LengthAndRandomness(t *testing.T) {
	s1, err := crypto.NewSalt(16)
	if err != nil {
		t.Fatalf("NewSalt error: %v", err)
	}
	s2, err := crypto.NewSalt(16)
	if err != nil {
		t.Fatalf("NewSalt error: %v", err)
	}
	if len(s1) != 16 || len(s2) != 16 {
		t.Fatalf("unexpected salt lengths: %d, %d", len(s1), len(s2))
	}

	// очень грубая проверка случайности: два соли не должны совпасть
	eq := true
	for i := range s1 {
		if s1[i] != s2[i] {
			eq = false
			break
		}
	}
	if eq {
		t.Fatalf("expected salts to differ, got identical")
	}
}

func TestDeriveKey_DeterministicForSameInput(t *testing.T) {
	p := crypto.DefaultKDFParams()
	salt := []byte("0123456789abcdef") // 16 bytes

	k1 := crypto.DeriveKey("pw", salt, p)
	k2 := crypto.DeriveKey("pw", salt, p)

	if len(k1) != int(p.KeyLen) || len(k2) != int(p.KeyLen) {
		t.Fatalf("unexpected key lengths: %d, %d", len(k1), len(k2))
	}

	for i := range k1 {
		if k1[i] != k2[i] {
			t.Fatalf("expected deterministic keys, mismatch at %d", i)
		}
	}
}

func TestDeriveKey_DifferentPasswordOrSalt_ProducesDifferentKey(t *testing.T) {
	p := crypto.DefaultKDFParams()
	salt1 := []byte("0123456789abcdef")
	salt2 := []byte("fedcba9876543210")

	k1 := crypto.DeriveKey("pw1", salt1, p)
	k2 := crypto.DeriveKey("pw2", salt1, p)
	k3 := crypto.DeriveKey("pw1", salt2, p)

	if bytesEqual(k1, k2) {
		t.Fatalf("expected different keys for different password")
	}
	if bytesEqual(k1, k3) {
		t.Fatalf("expected different keys for different salt")
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
