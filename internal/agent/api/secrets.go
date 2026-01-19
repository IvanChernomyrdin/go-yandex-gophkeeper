package api

import (
	"fmt"

	sharedModels "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/models"
)

// Sync загружает все секреты пользователя с сервера.
//
// Выполняет запрос:
//
//	GET /secrets
//
// Параметры:
//   - accessToken: access-токен пользователя. Передаётся в заголовке Authorization: Bearer <token>.
//
// Возвращает:
//   - sharedModels.GetAllSecretsResponse (обычно содержит массив secrets)
//   - ошибку, если запрос завершился неуспешно (не 2xx) или ответ не удалось декодировать.
func (c *Client) Sync(accessToken string) (sharedModels.GetAllSecretsResponse, error) {
	var resp sharedModels.GetAllSecretsResponse
	err := c.GetJSON("/secrets", &resp, accessToken)
	return resp, err
}

// CreateSecret создаёт новый секрет на сервере.
//
// Выполняет запрос:
//
//	POST /secrets
//
// Тело запроса сериализуется в JSON из sharedModels.CreateSecretRequest.
// Обычно payload уже зашифрован на клиенте и передаётся как ciphertext (строка).
//
// Параметры:
//   - accessToken: access-токен пользователя (Authorization: Bearer <token>)
//   - req: данные создаваемого секрета (type/title/payload и опционально meta)
//
// Возвращает:
//   - sharedModels.CreateSecretResponse (ID, version, timestamps и др.)
//   - ошибку, если запрос завершился неуспешно (не 2xx) или ответ не удалось декодировать.
func (c *Client) CreateSecret(accessToken string, req sharedModels.CreateSecretRequest) (sharedModels.CreateSecretResponse, error) {
	var resp sharedModels.CreateSecretResponse
	err := c.PostJSON("/secrets", req, &resp, accessToken)
	return resp, err
}

// UpdateSecret обновляет существующий секрет на сервере по ID.
//
// Выполняет запрос:
//   PUT /secrets/{id}
//
// Тело запроса сериализуется в JSON из sharedModels.UpdateSecretRequest.
// Для partial update могут передаваться только изменяемые поля.
// Обычно используется optimistic locking: версия передаётся в req.Version.
//
// Важно: сервер может отвечать 204 No Content. В этом случае метод возвращает
// zero-value sharedModels.SecretResponse и nil error.
//
// Параметры:
//   - accessToken: access-токен пользователя (Authorization: Bearer <token>)
//   - id: идентификатор секрета (uuid)
//   - req: патч-данные обновления (type/title/payload/meta/version)
//
// Возвращает:
//   - sharedModels.SecretResponse (если сервер возвращает тело) либо zero-value при 204
//   - ошибку при неуспешном статусе (не 2xx) или ошибке декодирования JSON.

func (c *Client) UpdateSecret(accessToken, id string, req sharedModels.UpdateSecretRequest) (sharedModels.SecretResponse, error) {
	var resp sharedModels.SecretResponse
	err := c.PutJSON(fmt.Sprintf("/secrets/%s", id), req, &resp, accessToken)
	return resp, err
}

// DeleteSecret удаляет секрет на сервере по ID с учётом версии.
//
// Выполняет запрос:
//   DELETE /secrets/{id}?version=N
//
// Используется optimistic locking: сервер удалит секрет только если версия совпадает.
// В случае конфликта версия/состояние могут отличаться, сервер вернёт ошибку.
//
// Параметры:
//   - accessToken: access-токен пользователя (Authorization: Bearer <token>)
//   - id: идентификатор секрета (uuid)
//   - version: ожидаемая версия секрета для удаления (optimistic locking)
//
// Возвращает:
//   - nil при успешном удалении (2xx, включая 204 No Content)
//   - ошибку при неуспешном статусе (не 2xx).

func (c *Client) DeleteSecret(accessToken, id string, version int) error {
	path := fmt.Sprintf("/secrets/%s?version=%d", id, version)
	return c.DeleteJSON(path, nil, accessToken)
}
