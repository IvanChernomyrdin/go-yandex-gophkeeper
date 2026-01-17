// @title           GophKeeper API
// @version         1.0
// @description     Secure password manager backend (GophKeeper).
// @description     Provides user authentication and secret storage.
// @termsOfService  https://example.com/terms

// @contact.name   Ivan Chernomyrdin
// @contact.url    https://github.com/IvanChernomyrdin
// @contact.email  ivan@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
//
// Package main содержит точку входа серверного приложения GophKeeper.
//
// Пакет отвечает за инициализацию и жизненный цикл HTTP(S)-сервера, а именно:
//   - загрузку переменных окружения из файла .env (если он присутствует);
//   - загрузку конфигурации сервера из файла ./configs/server.yaml;
//   - обязательную проверку включённого TLS (сервер работает только по HTTPS);
//   - инициализацию подключения к базе данных и управление его жизненным циклом;
//   - создание репозиториев, сервисов, middleware и HTTP-обработчиков;
//   - настройку и запуск HTTPS-сервера с заданными таймаутами;
//   - обработку системных сигналов завершения (SIGINT, SIGTERM, SIGQUIT);
//   - корректное (graceful) завершение работы сервера с таймаутом.
//
// Пакет не содержит бизнес-логики и не предназначен для unit-тестирования.
// HTTP API сервера реализовано в пакете internal/server/api и документируется с помощью OpenAPI (Swagger).
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/middleware"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/repository"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/logger"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/joho/godotenv"

	_ "github.com/IvanChernomyrdin/go-yandex-gophkeeper/swagger/docs"
)

func main() {
	sugar := logger.NewHTTPLogger().Logger.Sugar()
	httpLogger := logger.NewHTTPLogger()

	if err := godotenv.Load(); err != nil {
		sugar.Warnf("no .env file loaded, error: %v", err)
	}

	cfg, err := config.Load("./configs/server.yaml")
	if err != nil {
		sugar.Fatal(err)
	}
	// хочу только https
	if !cfg.TLS.Enabled {
		sugar.Fatal("tls must be enabled")
	}
	// подключаем базу данных
	if err := config.Init(cfg.DB.DSN); err != nil {
		sugar.Fatal(err)
	}
	// возвращаем указатель на db
	db := config.GetDB()
	// делаем отложенное закрытие бд
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	// создаём репы
	usersRepo := repository.NewUsersRepository(db)
	sessionsRepo := repository.NewSessionsRepository(config.GetDB())
	// складываем в репозиторий
	repos := service.Repositories{
		Users:    usersRepo,
		Sessions: sessionsRepo,
	}
	// создаём сервис
	svc := service.NewServices(repos, cfg)
	// создаём jwt
	verifier := middleware.NewJWTVerifier(
		cfg.Auth.JWT.SigningKey,
		cfg.Auth.Issuer,
		cfg.Auth.Audience,
	)
	// создаём хандлер
	handler := api.NewHandler(svc, httpLogger, verifier)
	// создаём роутер
	router := api.NewRouter(handler)
	//создаём сервер
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
	// канал для ошибок
	serverErrChan := make(chan error, 1)

	// запускаем сервер
	go func() {
		sugar.Infof("server started on %s", addr)

		err := server.ListenAndServeTLS(cfg.TLS.CertFile, cfg.TLS.KeyFile)

		// err всегда вернёт ошибку при остановке. Стандартное завершение - ErrServerClosed
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrChan <- err
			return
		}
		serverErrChan <- nil
	}()

	// ждём сигнал или остановку сервака
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	select {
	case <-signalCtx.Done():
		sugar.Info("shutdown signal received")
	case err := <-serverErrChan:
		if err != nil {
			sugar.Fatalf("server error: %v", err)
		}
		sugar.Infof("server stopped")
		return
	}

	// graceful shutdown с таймаутом из конфига
	shutdownTimeout := cfg.Server.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		sugar.Errorf("graceful shutdown failed: %v", err)
		// закрываем принудительно
		server.Close()
	}

	sugar.Info("server gracefully stopped")
}
