package tests

import (
	"errors"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/crypto"
)

func TestEncryptDecrypt_RoundTrip_Success(t *testing.T) {
	pw := "StrongPass123!"
	plain := []byte(`{"text":"hello"}`)

	blob, err := crypto.EncryptPayload(pw, plain)
	if err != nil {
		t.Fatalf("EncryptPayload error: %v", err)
	}

	got, err := crypto.DecryptPayload(pw, blob)
	if err != nil {
		t.Fatalf("DecryptPayload error: %v", err)
	}

	if string(got) != string(plain) {
		t.Fatalf("plaintext mismatch: got=%q want=%q", string(got), string(plain))
	}
}

func TestEncryptPayload_Format_MagicAndMinSize(t *testing.T) {
	pw := "pw"
	plain := []byte("x")

	blob, err := crypto.EncryptPayload(pw, plain)
	if err != nil {
		t.Fatalf("EncryptPayload error: %v", err)
	}

	// magic
	if string(blob[:len(crypto.FormatMagic)]) != crypto.FormatMagic {
		t.Fatalf("expected magic %q, got %q", crypto.FormatMagic, string(blob[:len(crypto.FormatMagic)]))
	}

	p := crypto.DefaultKDFParams()
	minLen := len(crypto.FormatMagic) + p.SaltLen + crypto.NonceSize + 1
	if len(blob) < minLen {
		t.Fatalf("expected blob len >= %d, got %d", minLen, len(blob))
	}
}

func TestDecryptPayload_WrongPassword_ReturnsErrAuthFailed(t *testing.T) {
	pw := "pw"
	plain := []byte("secret")

	blob, err := crypto.EncryptPayload(pw, plain)
	if err != nil {
		t.Fatalf("EncryptPayload error: %v", err)
	}

	_, err = crypto.DecryptPayload("wrong", blob)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, crypto.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

func TestDecryptPayload_InvalidMagic_ReturnsErrInvalidFormat(t *testing.T) {
	pw := "pw"
	plain := []byte("secret")

	blob, err := crypto.EncryptPayload(pw, plain)
	if err != nil {
		t.Fatalf("EncryptPayload error: %v", err)
	}

	// портим magic
	blob[0] = 'X'

	_, err = crypto.DecryptPayload(pw, blob)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errorsIs(err, crypto.ErrInvalidFormat) {
		t.Fatalf("expected ErrInvalidFormat, got %v", err)
	}
}

func TestDecryptPayload_TooShort_ReturnsErrCiphertextShort(t *testing.T) {
	_, err := crypto.DecryptPayload("pw", []byte("gk1"))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errorsIs(err, crypto.ErrCiphertextShort) {
		t.Fatalf("expected ErrCiphertextShort, got %v", err)
	}
}

func TestDecryptPayload_EmptyCiphertext_ReturnsErrCiphertextShort(t *testing.T) {
	p := crypto.DefaultKDFParams()

	// blob = magic + salt + nonce, ciphertext отсутствует
	blob := make([]byte, 0, len(crypto.FormatMagic)+p.SaltLen+crypto.NonceSize)
	blob = append(blob, []byte(crypto.FormatMagic)...)
	blob = append(blob, make([]byte, p.SaltLen)...)
	blob = append(blob, make([]byte, crypto.NonceSize)...)

	_, err := crypto.DecryptPayload("pw", blob)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, crypto.ErrCiphertextShort) {
		t.Fatalf("expected ErrCiphertextShort, got %v", err)
	}
}

func errorsIs(err, target error) bool {
	if err == nil || target == nil {
		return false
	}
	if err == target {
		return true
	}
	return errors.Is(err, target)
}
