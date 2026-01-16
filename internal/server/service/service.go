// Package service содержит бизнес-логику приложения (gophkeeper).
// Это прослойка между HTTP-обработчиками (api) и хранилищем данных (repository).
package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
)

// Repositories — набор интерфейсов, которые сервисный слой ожидает от слоя repository.
type Repositories struct {
	Users    UsersRepo
	Secrets  SecretsRepo
	Sessions SessionsRepo
}

// Services — агрегатор всех сервисов приложения.
type Services struct {
	Auth *AuthService
	// Secrets *SecretsService
}

// NewServices собирает все сервисы приложения.
// cfg нужен AuthService (параметры хеширования пароля).
func NewServices(repos Repositories, cfg *config.Config) *Services {
	return &Services{
		Auth: NewAuthService(repos.Users, repos.Sessions, cfg),
		// Secrets: NewSecretsService(repos.Secrets),
	}
}

// HealthRepo — минимально нужное для health-check.
type HealthRepo interface {
	Ping(ctx context.Context) error
}

// UsersRepo — репозиторий пользователей (нужен для auth/register/login).
type UsersRepo interface {
	Create(ctx context.Context, email, passwordHash string) (uuid.UUID, error)
	GetByEmail(ctx context.Context, email string) (uuid.UUID, string, error)
}

// SecretsRepo — репозиторий секретов (CRUD + version).
type SecretsRepo interface {
	// потом добавишь методы
}

type SessionsRepo interface {
	Create(ctx context.Context, userID uuid.UUID, refreshHash []byte, expiresAt time.Time) (uuid.UUID, error)
	GetByRefreshHash(ctx context.Context, refreshHash []byte) (id uuid.UUID, userID uuid.UUID, expiresAt time.Time, revokedAt *time.Time, replacedBy *uuid.UUID, err error)
	RevokeAndReplace(ctx context.Context, oldID, newID uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}
