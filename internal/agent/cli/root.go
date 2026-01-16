// Package cli реализует командный интерфейс (CLI) клиентского приложения GophKeeper.
//
// Пакет отвечает за:
//   - определение root-команды и набора подкоманд;
//   - разбор аргументов и флагов командной строки;
//   - загрузку локальных учётных данных (access/refresh токены) из конфигурационного файла;
//   - выполнение команд и вывод результата пользователю.
//
// Точка входа пакета — функция Execute.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/config"
)

// App содержит состояние CLI-приложения, разделяемое между командами.
//
// В структуре хранятся параметры подключения к серверу и загруженные учётные данные.
// Экземпляр App создаётся при построении root-команды и передаётся в подкоманды.
type App struct {
	// ServerURL — базовый URL сервера GophKeeper (например, "https://127.0.0.1:8080").
	ServerURL string

	// CredsPath — путь к файлу с сохранёнными учётными данными (access/refresh токены).
	CredsPath string
	// Creds — загруженные учётные данные из файла конфигурации.
	// Может быть nil, если загрузка не выполнялась или завершилась ошибкой.
	Creds *config.Credentials
}

// NewRootCmd создаёт root-команду CLI и регистрирует подкоманды.
//
// buildVersion и buildDate используются для вывода информации о сборке (команда version).
// В PersistentPreRunE выполняется инициализация состояния приложения:
// определяется путь к файлу учётных данных и загружаются сохранённые токены.
func NewRootCmd(buildVersion, buildDate string) *cobra.Command {
	app := &App{
		ServerURL: "https://127.0.0.1:8080",
	}

	cmd := &cobra.Command{
		Use:   "gophkeeper",
		Short: "GophKeeper CLI — менеджер приватных данных (passwords/secrets)",
		Long: `GophKeeper CLI.

Команды:
  register  Регистрация нового пользователя
  login     Логин (получить access/refresh)
  refresh   Обновить access по refresh токену
  version   Версия и дата сборки

Примеры:

Регистрация:
  gophkeeper register --email test@example.com --password StrongPass123

Логин:
  gophkeeper login --email test@example.com --password StrongPass123
  (сохраняет access и refresh токены в локальном конфиге)

Refresh:
  gophkeeper refresh
	(обновляет access токен используя refresh_token из локального конфига)

Проверка токена:
  gophkeeper me
`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.DefaultPath()
			if err != nil {
				return err
			}
			app.CredsPath = p

			creds, err := config.Load(app.CredsPath)
			if err != nil {
				return err
			}
			app.Creds = creds
			return nil
		},
	}

	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	cmd.PersistentFlags().StringVar(&app.ServerURL, "server", "https://127.0.0.1:8080", "server base URL")

	cmd.AddCommand(NewRegisterCmd(app))
	cmd.AddCommand(NewLoginCmd(app))
	cmd.AddCommand(NewRefreshCmd(app))
	cmd.AddCommand(NewVersionCmd(buildVersion, buildDate))

	return cmd
}

// Execute запускает обработку CLI-команд.
//
// При ошибке выполнения команды сообщение выводится в stderr, после чего процесс
// завершается с кодом 1 (os.Exit(1)).
func Execute(buildVersion, buildDate string) {
	if err := NewRootCmd(buildVersion, buildDate).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
