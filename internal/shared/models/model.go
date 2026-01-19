package models

import "time"

// Secret — плоская модель секрета, используемая в HTTP API.
//
// Secret хранит метаданные секрета и его зашифрованный payload.
// Payload всегда передаётся/хранится в зашифрованном виде (ciphertext),
// обычно как base64-строка (но формат строки определяет клиент).
//
// Поля:
//   - ID: уникальный идентификатор секрета (UUID в виде строки)
//   - Type: тип секрета (text, login_password, binary, bank_card, otp и т.д.)
//   - Title: пользовательское название секрета
//   - Payload: зашифрованные данные секрета (ciphertext)
//   - Meta: опциональная мета-информация (JSON/строка), не шифруется сервером
//   - Version: версия записи для optimistic locking (инкрементируется на сервере)
//   - UpdatedAt: время последнего изменения секрета (серверное)
//   - CreatedAt: время создания секрета (серверное)
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

// GetAllSecretsResponse — ответ эндпоинта получения всех секретов пользователя.
//
// Используется в:
//   GET /secrets
//
// Содержит массив секретов.
type GetAllSecretsResponse struct {
	Secrets []Secret `json:"secrets"`
}

// SecretResponse — обёртка для ответа, если сервер возвращает секрет вложенным объектом.
//
// Используется, если контракт API предполагает формат:
//   {"secret":{...}}
//
// Примечание: в твоём текущем сервере некоторые методы могут возвращать 204 No Content,
// поэтому SecretResponse может не использоваться на PUT/DELETE.
type SecretResponse struct {
	Secret Secret `json:"secret"`
}

// CreateSecretRequest — запрос на создание нового секрета.
//
// Используется в:
//   POST /secrets
//
// Поля:
//   - Type/Title обязательны
//   - Payload должен быть уже подготовлен клиентом (обычно ciphertext/base64)
//   - Meta опциональна и передаётся как есть (часто JSON-строка)
type CreateSecretRequest struct {
	Type    string  `json:"type"`
	Title   string  `json:"title"`
	Payload string  `json:"payload"`
	Meta    *string `json:"meta,omitempty"`
}

// UpdateSecretRequest — запрос на обновление секрета (partial update) по ID.
//
// Используется в:
//   PUT /secrets/{id}
//
// Поля Type/Title/Payload/Meta — указатели, чтобы можно было передавать
// только изменяемые поля (omitempty работает корректно).
//
// Version — обязательное поле для optimistic locking:
// клиент передаёт текущую локальную версию, сервер обновляет запись только
// если version совпадает, иначе возвращает conflict.
type UpdateSecretRequest struct {
	Type    *string `json:"type,omitempty"`
	Title   *string `json:"title,omitempty"`
	Payload *string `json:"payload,omitempty"`
	Meta    *string `json:"meta,omitempty"`
	Version int     `json:"version"`
}

// DeleteSecretResponse — простой ответ на удаление, если сервер возвращает JSON.
//
// Возможный контракт:
//   {"ok": true}
//
// Примечание: в твоей реализации сервер может отвечать 204 No Content вместо JSON.
type DeleteSecretResponse struct {
	OK bool `json:"ok"`
}

// CreateSecretResponse — ответ на создание секрета.
//
// Используется в:
//   POST /secrets
//
// Возвращает:
//   - ID созданного секрета
//   - Version (обычно 1 на старте)
//   - UpdatedAt (время создания/обновления на сервере)
type CreateSecretResponse struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}
