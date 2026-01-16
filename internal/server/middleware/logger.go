// Логирование HTTP-запросов
package middleware

import (
	"net/http"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/logger"
)

type ResponseWriter struct {
	http.ResponseWriter
	Status int
	Size   int
}

func (w *ResponseWriter) WriteHeader(Status int) {
	w.Status = Status
	w.ResponseWriter.WriteHeader(Status)
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
	if w.Status == 0 {
		w.Status = http.StatusOK
	}
	Size, err := w.ResponseWriter.Write(b)
	w.Size += Size
	return Size, err
}

func LoggerMiddleware() func(http.Handler) http.Handler {
	loggerHTTP := logger.NewHTTPLogger()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wr := &ResponseWriter{ResponseWriter: w}
			next.ServeHTTP(wr, r)

			duration := time.Since(start).Seconds() * 1000
			loggerHTTP.LogRequest(r.Method, r.RequestURI, wr.Status, wr.Size, duration)
		})
	}
}
