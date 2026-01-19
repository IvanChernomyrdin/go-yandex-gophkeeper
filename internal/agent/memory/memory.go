package memory

import (
	"sync"
	"time"

	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

// Secret — локальная модель секрета, хранимая в памяти агентом.
//
// Поля соответствуют данным, которые приходят от сервера при sync.
// Payload хранится в виде ciphertext (обычно base64-строка).
type Secret struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Payload   string    `json:"payload"`
	Meta      *string   `json:"meta,omitempty"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

// SecretsStore — потокобезопасное in-memory хранилище секретов.
//
// Используется CLI/агентом для:
//   - выдачи секретов по ID (Get)
//   - получения списка локальных секретов (List)
//   - полной замены локального состояния после sync (ReplaceAll)
//   - локального обновления полей по данным из БД/сервера (UpdateFromDB)
//   - удаления секрета (Delete)
type SecretsStore struct {
	mu      sync.RWMutex
	secrets map[string]Secret
}

// NewSecrets создаёт пустое хранилище секретов.
func NewSecrets() *SecretsStore {
	return &SecretsStore{
		secrets: make(map[string]Secret),
	}
}

// Get возвращает секрет по ID.
//
// Если секрет отсутствует — возвращает serr.ErrSecretNotFound
func (s *SecretsStore) Get(id string) (Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result, ok := s.secrets[id]
	if !ok {
		return Secret{}, serr.ErrSecretNotFound
	}
	return result, nil
}

// ReplaceAll полностью заменяет содержимое стора переданным списком.
//
// Используется после sync, чтобы локальное состояние строго соответствовало серверу.
// Если в списке есть дубликаты по ID, последнее значение перезапишет предыдущее.
func (s *SecretsStore) ReplaceAll(secrets []Secret) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.secrets = make(map[string]Secret, len(secrets))
	for _, sec := range secrets {
		s.secrets[sec.ID] = sec
	}
}

// List возвращает список всех секретов из стора.
//
// Порядок элементов не гарантируется (map).
func (s *SecretsStore) List() []Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Secret, 0, len(s.secrets))
	for _, secret := range s.secrets {
		result = append(result, secret)
	}
	return result
}

// UpdateFromDB обновляет поля локального секрета по ID.
//
// Обновляются только те поля, для которых переданы непустые указатели.
// Также всегда обновляется UpdatedAt на time.Now().
//
// Если секрет отсутствует — возвращает serr.ErrSecretNotFound.
func (s *SecretsStore) UpdateFromDB(id string, secreType *string, payload *string, meta *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sec, ok := s.secrets[id]
	if !ok {
		return serr.ErrSecretNotFound
	}

	if secreType != nil {
		sec.Type = *secreType
	}
	if payload != nil {
		sec.Payload = *payload
	}
	if meta != nil {
		sec.Meta = meta
	}

	sec.UpdatedAt = time.Now()
	s.secrets[id] = sec
	return nil
}

// Delete удаляет секрет по ID.
//
// Если секрет отсутствует — возвращает serr.ErrSecretNotFound.
func (s *SecretsStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.secrets[id]; !ok {
		return serr.ErrSecretNotFound
	}
	delete(s.secrets, id)
	return nil
}
