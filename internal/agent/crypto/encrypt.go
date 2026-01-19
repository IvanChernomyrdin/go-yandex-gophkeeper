package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

var (
	// ErrInvalidKey возвращается, если DeriveKey вернул ключ неправильной длины.
	// Для AES-256 ожидается 32 байта.
	ErrInvalidKey = errors.New("invalid key length")
)

const (
	// FormatMagic — сигнатура формата ciphertext blob.
	// Нужна для быстрой проверки, что данные действительно от GophKeeper.
	FormatMagic = "gk1"

	// NonceSize — размер nonce для AES-GCM.
	// 12 байт — стандартный размер для GCM.
	NonceSize = 12
)

// EncryptPayload шифрует payload мастер-паролем и возвращает единый бинарный blob.
//
// Формат выходных данных:
//
//	"gk1" + salt(16) + nonce(12) + ciphertext
//
// Где:
//   - "gk1" — магическая сигнатура формата,
//   - salt — случайная соль для Argon2id,
//   - nonce — случайный nonce для AES-GCM,
//   - ciphertext — результат AES-GCM.Seal.
//
// Важно:
//   - payload шифруется локально, мастер-пароль на сервер не передаётся.
//   - blob предназначен для хранения/передачи как есть (часто дополнительно кодируется в base64).
//
// Ошибки:
//   - ErrInvalidKey если derivation ключа дал неправильную длину,
//   - ошибки генерации соли/nonce,
//   - ошибки инициализации AES/GCM.
func EncryptPayload(masterPassword string, payload []byte) ([]byte, error) {
	params := DefaultKDFParams()

	salt, err := NewSalt(params.SaltLen)
	if err != nil {
		return nil, err
	}
	key := DeriveKey(masterPassword, salt, params)
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("rand nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, payload, nil)

	out := make([]byte, 0, len(FormatMagic)+len(salt)+len(nonce)+len(ciphertext))
	out = append(out, []byte(FormatMagic)...)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)

	return out, nil
}
