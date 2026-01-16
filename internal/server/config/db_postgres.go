// Package config содержит инициализацию подключения к базе данных сервера
// и доступ к глобальному экземпляру *sql.DB.
//
// Пакет выполняет:
//   - открытие соединения с PostgreSQL (через драйвер pgx);
//   - проверку доступности базы (Ping);
//   - запуск миграций (golang-migrate) при старте сервера.
//
// Примечание: пакет использует глобальную переменную DB. Инициализация должна
// выполняться один раз при запуске сервера.
package config

import (
	"database/sql"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/logger"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
)

// DB — глобальный экземпляр подключения к базе данных.
//
// Инициализируется функцией Init и используется другими пакетами через GetDB.
var DB *sql.DB

// Init открывает подключение к базе данных по DSN, проверяет его доступность
// и применяет миграции.
//
// databaseDSN — строка подключения к PostgreSQL.
// Миграции запускаются из каталога file://migrations/postgres.
// Если миграции уже применены, ошибка migrate.ErrNoChange не считается ошибкой.
func Init(databaseDSN string) error {
	customLog := logger.NewHTTPLogger().Logger.Sugar()

	var err error
	DB, err = sql.Open("pgx", databaseDSN)

	if err != nil {
		customLog.Errorf("error to connect db: %v", err)
		return err
	}

	if err = DB.Ping(); err != nil {
		customLog.Errorf("error check db connection: %v", err)
		return err
	}

	// Запуск миграций
	driver, err := postgres.WithInstance(DB, &postgres.Config{})
	if err != nil {
		customLog.Errorf("error creating migration driver: %v", err)
		return err
	}

	// создаём миграции с выбранным драйвером
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations/postgres",
		"postgres", driver)
	if err != nil {
		customLog.Errorf("error creating migrations: %v", err)
		return err
	}

	// запускаем создание миграций
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		customLog.Errorf("error applying migrations: %v", err)
		return err
	}

	customLog.Info("migrations applied successfully")
	return nil
}

// GetDB возвращает текущий глобальный экземпляр *sql.DB.
//
// Возвращаемое значение может быть nil, если Init ещё не вызывался
// или завершился ошибкой.
func GetDB() *sql.DB {
	return DB
}
