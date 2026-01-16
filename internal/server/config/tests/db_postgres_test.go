package tests

import (
	"database/sql"
	"os"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
)

func TestGetDB_ReturnsGlobalDB(t *testing.T) {
	orig := config.DB
	defer func() { config.DB = orig }()

	// подменяем глобальную переменную
	config.DB = &sql.DB{}
	if config.GetDB() != config.DB {
		t.Fatalf("GetDB should return global DB pointer")
	}
}

// Чтобы этот тест работал, нужено поднять Postgres и задать DSN, плюс чтобы путь file://migrations/postgres был доступен из рабочей директории тестов.
func TestInit_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set; skipping integration test")
	}

	orig := config.DB
	defer func() { config.DB = orig }()

	if err := config.Init(dsn); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if config.GetDB() == nil {
		t.Fatalf("expected DB to be initialized")
	}

	config.GetDB().Close()
	config.DB = nil
}
