package crypto

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	// SaltSize — размер соли по умолчанию (байты).
	// Используется для derivation ключа из мастер-пароля.
	SaltSize = 16

	// KeySize — размер симметричного ключа (байты).
	// Для AES-256 используется 32 байта.
	KeySize = 32
)

// KDFParams описывает параметры derivation ключа (KDF) на базе Argon2id.
//
// Поля:
//   - Time: количество итераций (CPU cost)
//   - Memory: объём памяти в KiB (memory cost)
//   - Threads: число потоков
//   - KeyLen: длина выводимого ключа в байтах
//   - SaltLen: длина соли в байтах
//
// В рамках MVP параметры могут быть захардкожены, но структура оставлена
// для возможной настройки в будущем (например, миграции параметров).
type KDFParams struct {
	Time    uint32 // iterations
	Memory  uint32 // KiB
	Threads uint8
	KeyLen  uint32
	SaltLen int
}

// DefaultKDFParams возвращает параметры KDF по умолчанию.
//
// Параметры подобраны как разумный компромисс для CLI (MVP):
// достаточно дорого для атак перебором, но ещё приемлемо по времени/памяти
// на обычной машине пользователя.
func DefaultKDFParams() KDFParams {
	return KDFParams{
		Time:    2,
		Memory:  64 * 1024, // 64 MiB
		Threads: 2,
		KeyLen:  KeySize,
		SaltLen: SaltSize,
	}
}

// NewSalt генерирует криптографически стойкую соль длиной n байт.
//
// Возвращает:
//   - []byte длиной n
//   - ошибку при проблемах с генератором случайных чисел.
func NewSalt(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("rand salt: %w", err)
	}
	return b, nil
}

// DeriveKey выводит симметричный ключ из masterPassword и salt с использованием Argon2id.
//
// Важно:
//   - Для безопасности salt должен быть уникальным и случайным для каждого шифрования.
//   - Параметры p должны соответствовать тем, что использовались при EncryptPayload,
//     иначе DecryptPayload не сможет восстановить тот же ключ.
func DeriveKey(masterPassword string, salt []byte, p KDFParams) []byte {
	return argon2.IDKey(
		[]byte(masterPassword),
		salt,
		p.Time,
		p.Memory,
		p.Threads,
		p.KeyLen,
	)
}
