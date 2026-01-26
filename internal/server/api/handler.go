// Package api реализует HTTP-слой сервера GophKeeper.
//
// Пакет отвечает за:
//   - регистрацию HTTP-маршрутов и настройку роутера (chi);
//   - обработку входящих запросов и формирование ответов (JSON, статусы);
//   - маппинг доменных ошибок (service/repository) в HTTP-коды и сообщения;
//   - подключение middleware (логирование, проверка JWT и т.д.).
package api

import (
	"encoding/json"
	"net/http"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/middleware"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/logger"
)

// Handler агрегирует зависимости HTTP-слоя и предоставляет методы-хендлеры.
//
// Handler содержит:
//   - Svc: сервисный слой (бизнес-логика);
//   - Log: логгер для записи событий и ошибок;
//   - Verifier: компонент проверки JWT и middleware авторизации.
//
// Методы Handler используются роутером для обработки HTTP-запросов.
type Handler struct {
	Svc      *service.Services
	Log      *logger.HTTPLogger
	Verifier *middleware.JWTVerifier
}

// NewHandler создаёт экземпляр Handler с переданными зависимостями.
//
// svc — набор сервисов приложения,
// log — логгер,
// verifier — JWT-проверка и middleware авторизации.
func NewHandler(svc *service.Services, log *logger.HTTPLogger, verifier *middleware.JWTVerifier) *Handler {
	return &Handler{
		Svc:      svc,
		Log:      log,
		Verifier: verifier,
	}
}

// Вспомогательная функция вывода ошибки
func WriteError(w http.ResponseWriter, status int, err error) {
	w.Header().Set(ContentType, JsonContentType)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: err.Error(),
	})
}
