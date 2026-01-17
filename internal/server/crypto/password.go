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

// Argon2Params описывает параметры алгоритма Argon2id.
//
// Параметры должны подбираться с учётом производительности сервера
// и требований к защите от перебора.
type Argon2Params struct {
	Time      uint32
	MemoryKiB uint32
	Threads   uint8
	KeyLen    uint32
	SaltLen   uint32
}

// HashPassword хэширует пароль с использованием Argon2id.
//
// Возвращает строку, содержащую:
//   - идентификатор алгоритма
//   - версию
//   - параметры
//   - соль (Base64)
//   - хэш (Base64)
//
// Формат:
//
//	argon2id$v=19$m=65536,t=3,p=2$<salt_b64>$<hash_b64>
//
// Ошибки:
//   - если пароль пустой
//   - если не удалось сгенерировать соль
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

// VerifyPassword проверяет, соответствует ли пароль ранее сохранённому хэшу.
//
// Функция:
//   - извлекает параметры Argon2 из encoded-строки
//   - повторно вычисляет хэш
//   - сравнивает его с сохранённым в constant-time
//
// Возвращает:
//   - true, если пароль корректен
//   - false, если пароль неверен
//
// Ошибка возвращается только при некорректном формате хэша.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 {
		return false, errors.New("invalid hash format")
	}

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
