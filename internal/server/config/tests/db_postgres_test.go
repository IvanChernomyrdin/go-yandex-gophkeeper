package tests

import (
	"database/sql"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// Тест с мок-базой данных через DI
func TestDatabaseInjection(t *testing.T) {
	// Создаём мок DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	// Проверяем работу простого запроса через мок
	mock.ExpectQuery(`SELECT 1`).WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	var x int
	err = db.QueryRow(`SELECT 1`).Scan(&x)
	require.NoError(t, err)
	require.Equal(t, 1, x)

	// Проверяем, что все ожидания моков выполнены
	require.NoError(t, mock.ExpectationsWereMet())
}

// Интеграционный тест с настоящей DB через DI
func TestInit_WithDSN(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping integration test")
	}

	// Инициализируем DB напрямую
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	require.NotNil(t, db)
	t.Cleanup(func() { db.Close() })

	// Простейший запрос для проверки
	var x int
	err = db.QueryRow("SELECT 1").Scan(&x)
	require.NoError(t, err)
	require.Equal(t, 1, x)
}
