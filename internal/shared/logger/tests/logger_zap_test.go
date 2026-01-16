package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/logger"
)

func TestNewHTTPLogger_CreatesLogFileAndWrites(t *testing.T) {
	// ВАЖНО: тест не параллелим, т.к. путь общий.
	logPath := filepath.Join("runtime", "logs", "http.log")

	// подчистим старый файл (если есть)
	os.Remove(logPath)

	l := logger.NewHTTPLogger()
	// пишем лог
	l.Info("test message")
	// закрываем буферы zap
	_ = l.Sync()

	// проверяем, что файл создан
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected log file to exist at %q, got error: %v", logPath, err)
	}

	// читаем содержимое
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(b)

	if len(s) == 0 {
		t.Fatalf("expected non-empty log file")
	}
	if !regexp.MustCompile(`\btest message\b`).MatchString(s) {
		t.Fatalf("expected log to contain message, got: %q", s)
	}

	// проверяем формат времени: "HH:MM:SS DD.MM.YYYY"
	// пример: 11:57:16 16.01.2026
	timeRe := regexp.MustCompile(`\b\d{2}:\d{2}:\d{2} \d{2}\.\d{2}\.\d{4}\b`)
	if !timeRe.MatchString(s) {
		t.Fatalf("expected custom time format (HH:MM:SS DD.MM.YYYY), got: %q", s)
	}

	// cleanup (может не получиться на Windows, если файл ещё держится системой — это ок)
	os.Remove(logPath)
}

func TestHTTPLogger_LogRequest_WritesStructuredFields(t *testing.T) {
	logPath := filepath.Join("runtime", "logs", "http.log")
	os.Remove(logPath)

	l := logger.NewHTTPLogger()
	l.LogRequest("POST", "/auth/login", 401, 20, 158.5463)
	l.Sync()

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(b)

	// проверяем наличие ключевых полей
	mustContain := []string{
		"HTTP request",
		"method", "POST",
		"uri", "/auth/login",
		"status", "401",
		"response_size", "20",
		"duration_ms",
	}
	for _, sub := range mustContain {
		if !regexp.MustCompile(regexp.QuoteMeta(sub)).MatchString(s) {
			t.Fatalf("expected log to contain %q, got: %q", sub, s)
		}
	}

	os.Remove(logPath)
}
