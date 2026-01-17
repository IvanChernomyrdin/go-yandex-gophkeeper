// Package api реализует HTTP-слой сервера GophKeeper.
//
// Пакет отвечает за:
//   - регистрацию HTTP-маршрутов и настройку роутера (chi);
//   - обработку входящих запросов и формирование ответов (JSON, статусы);
//   - маппинг доменных ошибок (service/repository) в HTTP-коды и сообщения;
//   - подключение middleware (логирование, проверка JWT и т.д.).
package api

import (
	"net/http"

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
func NewRouter(h *Handler) http.Handler {
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
		// CRUD запросы для секретов
		r.Route("/secrets", func(r chi.Router) {
			r.Post("/", h.CreateSecret) // Создание секрета
			r.Get("/", h.ListSecrets)   // Получение все секретов
			//     r.Get("/{id}", h.GetSecret)
			//     r.Put("/{id}", h.UpdateSecret)
			//     r.Delete("/{id}", h.DeleteSecret)
		})
	})

	return r
}
