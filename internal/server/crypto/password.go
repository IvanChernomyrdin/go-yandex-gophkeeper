// Хэширование паролей
package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2Params struct {
	Time      uint32
	MemoryKiB uint32
	Threads   uint8
	KeyLen    uint32
	SaltLen   uint32
}

// HashPassword возвращает строку формата:
// argon2id$v=19$m=65536,t=3,p=2$<salt_b64>$<hash_b64>
func HashPassword(password string, p Argon2Params) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", errors.New("empty password")
	}

	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, p.Time, p.MemoryKiB, p.Threads, p.KeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		p.MemoryKiB, p.Time, p.Threads,
		b64Salt, b64Hash,
	)
	return encoded, nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 {
		return false, errors.New("invalid hash format")
	}

	// parts[0] = argon2id
	// parts[1] = v=19
	// parts[2] = m=...,t=...,p=...
	// parts[3] = salt
	// parts[4] = hash

	var memory uint32
	var time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, errors.New("invalid params format")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, errors.New("invalid salt")
	}

	wantHash, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, errors.New("invalid hash")
	}

	got := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(wantHash)))
	return subtle.ConstantTimeCompare(got, wantHash) == 1, nil
}
