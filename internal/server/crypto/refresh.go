package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// NewRefreshToken генерирует криптографически стойкий refresh-токен.
//
// Токен:
//   - имеет длину 256 бит
//   - генерируется с использованием crypto/rand
//   - кодируется в Base64 URL-safe формате
//
// Возвращает этот токен
func NewRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashRefreshToken хэширует refresh-токен с использованием SHA-256.
//
// Используется для безопасного хранения refresh-токена в БД.
// Оригинальный токен:
//   - никогда не сохраняется
//   - используется только для сравнения при обновлении сессии
//
// Возвращает 32-байтовый хэш.
func HashRefreshToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
