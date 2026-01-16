// Package api содержит клиент для взаимодействия с HTTP API сервера GophKeeper.
package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// Client реализует HTTP-клиент для общения с сервером GophKeeper.
//
// Клиент инкапсулирует baseURL и настроенный http.Client.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient создаёт новый HTTP-клиент для общения с сервером.
//
// baseURL — базовый адрес сервера (например, "https://127.0.0.1:8080").
// Клиент использует таймаут запросов 10 секунд.
// В режиме разработки отключена проверка TLS-сертификата (InsecureSkipVerify=true).
func NewClient(baseURL string) *Client {
	// в dev отключаем verify
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // только для dev
	}

	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout:   10 * time.Second,
			Transport: tr,
		},
	}
}

// PostJSON выполняет POST-запрос к серверу, сериализуя тело запроса в JSON.
//
// Параметры:
//   - path: путь относительно baseURL (например, "/auth/login").
//   - req: структура/значение, которое будет закодировано в JSON и отправлено в теле запроса.
//   - resp: указатель на структуру для декодирования JSON-ответа (может быть nil).
//   - authToken: access-токен для заголовка Authorization; если пустой, заголовок не устанавливается.
//
// Поведение:
//   - всегда устанавливает заголовок "Content-Type: application/json";
//   - если authToken не пустой, добавляет "Authorization: Bearer <token>";
//   - при ответе не из диапазона 2xx возвращает ошибку с текстом тела ответа;
//   - при resp == nil не пытается декодировать тело ответа.
func (c *Client) PostJSON(path string, req any, resp any, authToken string) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return err
	}

	r, err := http.NewRequest(http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		r.Header.Set("Authorization", "Bearer "+authToken)
	}

	res, err := c.http.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var b bytes.Buffer
		b.ReadFrom(res.Body)
		return errors.New(b.String())
	}

	if resp != nil {
		return json.NewDecoder(res.Body).Decode(resp)
	}
	return nil
}

// GetJSON выполняет GET-запрос к серверу и декодирует JSON-ответ в resp.
//
// Параметры:
//   - path: путь относительно baseURL (например, "/me").
//   - resp: указатель на структуру для декодирования JSON-ответа (не должен быть nil).
//   - authToken: access-токен для заголовка Authorization; если пустой, заголовок не устанавливается.
//
// Поведение:
//   - если authToken не пустой, добавляет "Authorization: Bearer <token>";
//   - при ответе не из диапазона 2xx возвращает ошибку с текстом тела ответа;
//   - при успешном ответе декодирует JSON-тело в resp.
func (c *Client) GetJSON(path string, resp any, authToken string) error {
	r, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if authToken != "" {
		r.Header.Set("Authorization", "Bearer "+authToken)
	}

	res, err := c.http.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var b bytes.Buffer
		b.ReadFrom(res.Body)
		return errors.New(b.String())
	}

	return json.NewDecoder(res.Body).Decode(resp)
}
