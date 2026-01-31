// Package http реализует маршрутизацию HTTP-слоя сервера GophKeeper.
//
// Пакет отвечает за:
//   - регистрацию HTTP-маршрутов и настройку роутера (chi);
//   - логирование выполнения HTTP-запросов;
//   - выполняет проверку JWT access-токенов;
package http

import (
	"net/http"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/middleware"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter создаёт и настраивает HTTP-роутер сервера.
//
// Роутер использует chi.Router и регистрирует:
//   - публичные эндпоинты аутентификации под префиксом /auth;
//   - middleware логирования для всех запросов;
//   - группу защищённых JWT эндпоинтов (пока без маршрутов secrets).
func NewRouter(h *api.Handler) http.Handler {
	r := chi.NewRouter()
	// логирование всех запросов
	r.Use(middleware.LoggerMiddleware())

	// добавляем swagger
	r.Get("/swagger/*", httpSwagger.WrapHandler)
	// Публичные пути
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
	})
	// защищены пути
	r.Group(func(r chi.Router) {
		// проверка access токена
		r.Use(h.Verifier.AuthMiddleware())
		// запросы для секретов
		r.Route("/secrets", func(r chi.Router) {
			r.Post("/", h.CreateSecret) // Создание секрета
			r.Get("/", h.ListSecrets)   // Получение все секретов на клиенте делается каманда sync
			// r.Get("/{id}", h.GetSecret) // реализуется на клиенте
			r.Put("/{id}", h.UpdateSecret)    // обновляем, передаём id в параметрах и данные секрета в теле
			r.Delete("/{id}", h.DeleteSecret) // удаляем секрет по id и по ?version
		})
	})

	return r
}
