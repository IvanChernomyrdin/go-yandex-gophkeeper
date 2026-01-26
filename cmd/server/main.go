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
// @BasePath  /
// @schemes https

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

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/middleware"
	h "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/net/http"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/repository"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/service"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/logger"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"

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
	secretsRepo := repository.NewSecretsRepository(db)
	// складываем в репозиторий
	repos := service.Repositories{
		Users:    usersRepo,
		Sessions: sessionsRepo,
		Secrets:  secretsRepo,
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
	router := h.NewRouter(handler)
	//создаём сервер
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// создаём контекст и errgroup
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	g, ctx := errgroup.WithContext(ctx)

	// запускаем сервер
	g.Go(func() error {
		sugar.Infof("server started on %s", addr)

		if err := server.ListenAndServeTLS(
			cfg.TLS.CertFile,
			cfg.TLS.KeyFile,
		); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// graceful shutdown с таймаутом из конфига
	g.Go(func() error {
		<-ctx.Done()

		sugar.Info("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(
			ctx,
			cfg.Server.ShutdownTimeout,
		)
		defer cancel()

		return server.Shutdown(shutdownCtx)
	})

	// ожидание и единная обработка ошибок
	if err := g.Wait(); err != nil {
		sugar.Fatalf("server stopped with error: %v", err)
	}
	sugar.Info("server gracefully stopped")
}
