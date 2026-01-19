package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
)

var (
	// ErrInvalidFormat возвращается, если blob не соответствует ожидаемому формату
	// ("gk1"+salt+nonce+ciphertext).
	ErrInvalidFormat = errors.New("invalid ciphertext format")

	// ErrAuthFailed возвращается, если расшифровка не удалась:
	// неверный пароль или данные повреждены.
	// Детали ошибки намеренно скрываются.
	ErrAuthFailed = errors.New("decryption failed (wrong password or corrupted data)")

	// ErrCiphertextShort возвращается, если blob слишком короткий и не может
	// содержать magic+salt+nonce+минимальный ciphertext.
	ErrCiphertextShort = errors.New("ciphertext too short")
)

// DecryptPayload расшифровывает blob, сформированный EncryptPayload.
//
// Ожидаемый формат blob:
//
//	"gk1" + salt(16) + nonce(12) + ciphertext
//
// Алгоритм:
//  1. проверяет минимальную длину и сигнатуру формата;
//  2. извлекает salt и nonce;
//  3. выводит ключ Argon2id(masterPassword, salt, params);
//  4. выполняет AES-GCM.Open.
//
// Ошибки:
//   - ErrCiphertextShort если blob слишком короткий,
//   - ErrInvalidFormat если сигнатура/структура некорректна,
//   - ErrAuthFailed если неверный пароль или данные повреждены,
//   - обёрнутые ошибки инициализации AES/GCM.
func DecryptPayload(masterPassword string, blob []byte) ([]byte, error) {
	params := DefaultKDFParams()

	minLen := len(FormatMagic) + params.SaltLen + NonceSize + 1
	if len(blob) < minLen {
		return nil, ErrCiphertextShort
	}

	if string(blob[:len(FormatMagic)]) != FormatMagic {
		return nil, ErrInvalidFormat
	}

	off := len(FormatMagic)
	salt := blob[off : off+params.SaltLen]
	off += params.SaltLen

	nonce := blob[off : off+NonceSize]
	off += NonceSize

	ciphertext := blob[off:]
	if len(ciphertext) == 0 {
		return nil, ErrInvalidFormat
	}

	key := DeriveKey(masterPassword, salt, params)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrAuthFailed
	}
	return plain, nil
}
