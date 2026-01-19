package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SecretsDump — формат файла локального хранилища секретов.
//
// Файл содержит объект вида:
//   { "secrets": [ ... ] }
//
// Payload хранится в зашифрованном виде (ciphertext), как строка (обычно base64).

type SecretsDump struct {
	Secrets []Secret `json:"secrets"`
}

// DefaultSecretsPath возвращает путь по умолчанию для локального файла секретов.
//
// Путь формируется как:
//
//	$HOME/.gophkeeper/secrets.json
//
// Ошибки:
//   - возвращает ошибку, если не удаётся определить домашнюю директорию пользователя.
func DefaultSecretsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gophkeeper", "secrets.json"), nil
}

// SaveToFile сериализует текущее состояние store в JSON и сохраняет в файл по пути path.
//
// Поведение:
//   - читает store под RLock (потокобезопасно);
//   - создаёт директорию для файла (MkdirAll) с правами 0700;
//   - сохраняет файл с правами 0600;
//   - формат JSON: SecretsDump{Secrets:[...]} с отступами (MarshalIndent).
//
// Важно:
//   - порядок секретов в JSON не гарантируется (map).
func SaveToFile(path string, store *SecretsStore) error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	out := SecretsDump{Secrets: make([]Secret, 0, len(store.secrets))}
	for _, s := range store.secrets {
		out.Secrets = append(out.Secrets, s)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// LoadFromFile загружает secrets из JSON-файла в store.
//
// Поведение:
//   - если файл не существует — возвращает nil (это нормальная ситуация при первом запуске);
//   - если JSON некорректный — возвращает ошибку Unmarshal;
//   - при успешной загрузке полностью заменяет содержимое store (ReplaceAll semantics).
//
// Важно:
//   - операция замены выполняется под Lock (потокобезопасно).

func LoadFromFile(path string, store *SecretsStore) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var dump SecretsDump
	if err := json.Unmarshal(b, &dump); err != nil {
		return err
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	// заменяем полностью — после sync это удобно
	store.secrets = make(map[string]Secret, len(dump.Secrets))
	for _, s := range dump.Secrets {
		store.secrets[s.ID] = s
	}

	return nil
}
