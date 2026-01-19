// Package api содержит HTTP-клиент для взаимодействия с сервером GophKeeper.
//
// Клиент инкапсулирует базовый URL сервера и настроенный http.Client,
// предоставляя удобные методы для отправки JSON-запросов (POST/GET/PUT/DELETE)
// с авторизацией через Bearer токен.
//
// Особенности:
//   - baseURL нормализуется (обрезаются завершающие "/").
//   - По умолчанию добавляется заголовок Accept: application/json.
//   - Заголовок Content-Type: application/json добавляется только при наличии тела запроса.
//   - При ответах 204 No Content тело не читается и это считается успехом.
//   - Пустое тело ответа (EOF при декодировании) не считается ошибкой.
//   - При ошибочных ответах (не 2xx) возвращается ошибка с текстом тела ответа
//     (если тело пустое — используется res.Status).
//
// ВНИМАНИЕ: NewClient включает InsecureSkipVerify=true (TLS сертификат не проверяется).
// Это допустимо только для разработки и локального окружения. Для production следует
// включать проверку сертификата и/или использовать доверенный CA/сертификаты.
package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client реализует HTTP-клиент для общения с сервером GophKeeper.
//
// Поля:
//   - baseURL: базовый адрес сервера без завершающего слэша.
//   - http: настроенный http.Client (таймаут, транспорт, TLS).
//
// Client предоставляет методы PostJSON/GetJSON/PutJSON/DeleteJSON,
// которые отправляют HTTP-запросы и (при необходимости) декодируют JSON-ответ.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient создаёт новый HTTP-клиент для общения с сервером.
//
// Параметры:
//   - baseURL: базовый адрес сервера (например: "https://127.0.0.1:8080").
//
// Поведение:
//   - обрезает завершающий "/" у baseURL;
//   - создаёт http.Client с таймаутом 10 секунд;
//
// ВНИМАНИЕ: InsecureSkipVerify=true отключает проверку сертификата и делает TLS
// уязвимым для MITM. Использовать только для локальной разработки/тестов.
func NewClient(baseURL string) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // только для dev
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout:   10 * time.Second,
			Transport: tr,
		},
	}
}

// readAPIErrorBody читает тело ответа сервера и возвращает ошибку с текстом тела.
//
// Используется в случае HTTP-ошибок (не 2xx).
//
// Поведение:
//   - читает res.Body полностью;
//   - если тело непустое — возвращает error с этим текстом (trim пробелов);
//   - если тело пустое — возвращает error со строкой res.Status.
func readAPIErrorBody(res *http.Response) error {
	raw, _ := io.ReadAll(res.Body)
	msg := strings.TrimSpace(string(raw))
	if msg == "" {
		msg = res.Status
	}
	return errors.New(msg)
}

// decodeJSONOrOK декодирует JSON из r в resp.
//
// Параметры:
//   - r: источник данных (обычно res.Body);
//   - resp: указатель на структуру/объект для декодирования.
//     Если resp == nil — функция ничего не делает и возвращает nil.
//
// Особенность:
//   - Если тело ответа пустое и json.Decoder вернул io.EOF,
//     это НЕ считается ошибкой и возвращается nil.
//     Это полезно для эндпоинтов, которые могут возвращать пустое тело.
func decodeJSONOrOK(r io.Reader, resp any) error {
	if resp == nil {
		return nil
	}
	err := json.NewDecoder(r).Decode(resp)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

// PostJSON выполняет POST-запрос к серверу, сериализуя req в JSON.
//
// Параметры:
//   - path: путь относительно baseURL (например: "/auth/login").
//   - req: объект для сериализации в JSON. Если req == nil, тело не отправляется
//     и Content-Type не устанавливается.
//   - resp: указатель на структуру/объект для декодирования JSON-ответа.
//     Если resp == nil, тело ответа не декодируется.
//   - authToken: access токен. Если непустой, добавляется заголовок:
//     Authorization: Bearer <token>.
//
// Заголовки:
//   - всегда: Accept: application/json
//   - если req != nil: Content-Type: application/json
//
// Обработка ответа:
//   - 2xx: успех
//   - 204 No Content: успех без попытки декодирования тела
//   - прочие 2xx: декодирует JSON в resp (если resp != nil); EOF не ошибка
//   - не 2xx: возвращает ошибку с текстом тела ответа (или res.Status)
func (c *Client) PostJSON(path string, req any, resp any, authToken string) error {
	var buf bytes.Buffer
	if req != nil {
		if err := json.NewEncoder(&buf).Encode(req); err != nil {
			return err
		}
	}

	r, err := http.NewRequest(http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return err
	}
	r.Header.Set("Accept", "application/json")
	if req != nil {
		r.Header.Set("Content-Type", "application/json")
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
		return readAPIErrorBody(res)
	}

	// 204/пустое тело — ок
	if res.StatusCode == http.StatusNoContent {
		return nil
	}

	return decodeJSONOrOK(res.Body, resp)
}

// GetJSON выполняет GET-запрос к серверу и (опционально) декодирует JSON-ответ.
//
// Параметры:
//   - path: путь относительно baseURL (например: "/secrets").
//   - resp: указатель на структуру/объект для декодирования JSON-ответа.
//     Если resp == nil, тело ответа не декодируется.
//   - authToken: access токен. Если непустой, добавляется заголовок:
//     Authorization: Bearer <token>.
//
// Заголовки:
//   - всегда: Accept: application/json
//
// Обработка ответа:
//   - 2xx: успех
//   - 204 No Content: успех без попытки декодирования тела
//   - прочие 2xx: декодирует JSON в resp (если resp != nil); EOF не ошибка
//   - не 2xx: возвращает ошибку с текстом тела ответа (или res.Status)
func (c *Client) GetJSON(path string, resp any, authToken string) error {
	r, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	r.Header.Set("Accept", "application/json")
	if authToken != "" {
		r.Header.Set("Authorization", "Bearer "+authToken)
	}

	res, err := c.http.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return readAPIErrorBody(res)
	}

	if res.StatusCode == http.StatusNoContent {
		return nil
	}

	return decodeJSONOrOK(res.Body, resp)
}

// PutJSON выполняет PUT-запрос к серверу, сериализуя req в JSON.
//
// Параметры:
//   - path: путь относительно baseURL (например: "/secrets/{id}").
//   - req: объект для сериализации в JSON. Если req == nil, тело не отправляется
//     и Content-Type не устанавливается.
//   - resp: указатель на структуру/объект для декодирования JSON-ответа.
//     Если resp == nil, тело ответа не декодируется.
//   - authToken: access токен. Если непустой, добавляется заголовок:
//     Authorization: Bearer <token>.
//
// Заголовки:
//   - всегда: Accept: application/json
//   - если req != nil: Content-Type: application/json
//
// Обработка ответа:
//   - 2xx: успех
//   - 204 No Content: успех без попытки декодирования тела
//     (важно для эндпоинтов типа UpdateSecret, которые возвращают 204)
//   - прочие 2xx: декодирует JSON в resp (если resp != nil); EOF не ошибка
//   - не 2xx: возвращает ошибку с текстом тела ответа (или res.Status)
func (c *Client) PutJSON(path string, req any, resp any, authToken string) error {
	var buf bytes.Buffer
	if req != nil {
		if err := json.NewEncoder(&buf).Encode(req); err != nil {
			return err
		}
	}

	r, err := http.NewRequest(http.MethodPut, c.baseURL+path, &buf)
	if err != nil {
		return err
	}
	r.Header.Set("Accept", "application/json")
	if req != nil {
		r.Header.Set("Content-Type", "application/json")
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
		return readAPIErrorBody(res)
	}

	// КЛЮЧ: твой UpdateSecret возвращает 204
	if res.StatusCode == http.StatusNoContent {
		return nil
	}

	return decodeJSONOrOK(res.Body, resp)
}

// DeleteJSON выполняет DELETE-запрос к серверу и (опционально) декодирует JSON-ответ.
//
// Параметры:
//   - path: путь относительно baseURL (например: "/secrets/{id}?version=N").
//   - resp: указатель на структуру/объект для декодирования JSON-ответа.
//     Если resp == nil, тело ответа не декодируется.
//   - authToken: access токен. Если непустой, добавляется заголовок:
//     Authorization: Bearer <token>.
//
// Заголовки:
//   - всегда: Accept: application/json
//
// Обработка ответа:
//   - 2xx: успех
//   - 204 No Content: успех без попытки декодирования тела
//   - прочие 2xx: декодирует JSON в resp (если resp != nil); EOF не ошибка
//   - не 2xx: возвращает ошибку с текстом тела ответа (или res.Status)
func (c *Client) DeleteJSON(path string, resp any, authToken string) error {
	r, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	r.Header.Set("Accept", "application/json")
	if authToken != "" {
		r.Header.Set("Authorization", "Bearer "+authToken)
	}

	res, err := c.http.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return readAPIErrorBody(res)
	}

	if res.StatusCode == http.StatusNoContent {
		return nil
	}

	return decodeJSONOrOK(res.Body, resp)
}
