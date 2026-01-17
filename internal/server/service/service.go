// Package service содержит бизнес-логику приложения gophkeeper.
//
// Service-слой:
//   - не знает про HTTP, JSON, роутеры
//   - не знает про конкретную БД
//   - работает только с интерфейсами репозиториев
//
// Он инкапсулирует правила:
//   - аутентификации
//   - авторизации
//   - управления сессиями
//
// Архитектурно:
//
//	api --> service --> repository
package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
)

// Repositories — набор интерфейсов, которые сервисный слой ожидает от слоя repository.
//
// Используется для явного внедрения зависимостей (dependency injection)
// при сборке сервисов приложения.
type Repositories struct {
	Users    UsersRepo
	Sessions SessionsRepo
	Secrets  SecretsRepo
}

// Services — агрегатор всех сервисов приложения.
type Services struct {
	Auth    *AuthService
	Secrets *SecretsService
}

// NewServices собирает все сервисы приложения.
// cfg используется сервисами для:
//   - параметров хеширования паролей
//   - JWT-настроек
//   - TTL токенов и сессий.
func NewServices(repos Repositories, cfg *config.Config) *Services {
	return &Services{
		Auth:    NewAuthService(repos.Users, repos.Sessions, cfg),
		Secrets: NewSecretsService(repos.Secrets, cfg.Secrets),
	}
}

// UsersRepo — репозиторий описываеющий операции с пользователями (нужен для auth/register/login).
type UsersRepo interface {
	Create(ctx context.Context, email, passwordHash string) (uuid.UUID, error)
	GetByEmail(ctx context.Context, email string) (uuid.UUID, string, error)
}

// SessionsRepo описывает работу с refresh-сессиями.
//
// Используется для:
//   - refresh access-токенов
//   - ротации refresh-токенов
//   - детекта повторного использования
type SessionsRepo interface {
	Create(ctx context.Context, userID uuid.UUID, refreshHash []byte, expiresAt time.Time) (uuid.UUID, error)
	GetByRefreshHash(ctx context.Context, refreshHash []byte) (id uuid.UUID, userID uuid.UUID, expiresAt time.Time, revokedAt *time.Time, replacedBy *uuid.UUID, err error)
	RevokeAndReplace(ctx context.Context, oldID, newID uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// SecretType тип секрета
type SecretType string

const (
	SecretLoginPassword SecretType = "login_password"
	SecretText          SecretType = "text"
	SecretBinary        SecretType = "binary"
	SecretBankCard      SecretType = "bank_card"
	SecretOTP           SecretType = "otp"
)

type SecretsRepo interface {
	Create(
		ctx context.Context,
		userID uuid.UUID,
		typ SecretType,
		title string,
		payload string,
		meta *string,
	) (id uuid.UUID, version int, updatedAt time.Time, err error)
}
