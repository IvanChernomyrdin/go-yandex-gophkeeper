// Package config содержит функции для работы с локальной конфигурацией CLI-клиента.
//
// Конфигурация хранит учётные данные (access/refresh токены) и размещается
// в домашней директории пользователя в файле:
//
//	~/.gophkeeper/credentials.json
//
// Пакет предоставляет функции для получения пути по умолчанию, загрузки и сохранения
// конфигурации в JSON формате.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Credentials содержит учётные данные, используемые CLI-клиентом.
//
// AccessToken применяется для авторизации запросов к серверу.
// RefreshToken применяется для обновления пары токенов.
type Credentials struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// DefaultPath возвращает путь к конфигурационному файлу в домашней директории пользователя.
//
// Формат пути:
//
//	<home>/.gophkeeper/credentials.json
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gophkeeper", "credentials.json"), nil
}

// Load загружает конфигурацию из указанного файла.
//
// Если файл не существует, возвращает пустую конфигурацию без ошибки.
// Если файл существует, но содержит некорректный JSON, возвращает ошибку.
func Load(path string) (*Credentials, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// дефолтный конфиг, если файла нет
			return &Credentials{}, nil
		}
		return nil, err
	}
	var c Credentials
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Save сохраняет конфигурацию в указанный файл в JSON формате.
//
// При необходимости создаёт директорию назначения с правами 0700.
// Файл конфигурации записывается с правами 0600.
func Save(path string, c *Credentials) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
