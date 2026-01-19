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
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/memory"
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

	Secrets     *memory.SecretsStore
	SecretsPath string
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
		Long: `GophKeeper CLI — менеджер приватных данных.
Команды аутентификации:
  register    Регистрация нового пользователя
  login       Логин (получить access/refresh токены)
  refresh     Обновить access токен по refresh токену
  version     Версия и дата сборки

Команды работы с секретами:
  sync        Синхронизация локальных секретов с сервером
  get         Получить список всех локальных секретов
  get <id>    Получить секрет по ID
  set         Создать новый секрет
  update <id> Обновить существующий секрет по ID
  delete <id> Удалить секрет по ID

Описание команд:

Регистрация:
  Регистрирует нового пользователя в системе.
  gophkeeper register --email test@example.com --password StrongPass123

Логин:
  Выполняет аутентификацию и сохраняет access/refresh токены в локальном конфиге.
  gophkeeper login --email test@example.com --password StrongPass123

Refresh:
  Обновляет access токен, используя refresh токен из локального конфига.
  gophkeeper refresh

Version:
  Отображает версию и дату сборки клиента.
  gophkeeper version

Sync:
  Загружает все секреты с сервера и сохраняет их локально.
  gophkeeper sync

Get:
  Отображает все локально сохранённые секреты.
  gophkeeper get

Get <id>:
  Отображает один секрет по его ID.
  gophkeeper get 1

Set:
  Создаёт новый секрет на сервере.

  Типы секретов:
    login_password — логин и пароль для сайта, БД, VPN и т.п.
    text           — произвольный текст (API token, JWT, SSH private key, заметки)
    binary         — бинарные данные (в base64)
    bank_card      — платёжные данные
    otp            — одноразовые коды

  Общий формат:
    gophkeeper set --type <type> --title <title> --payload '<json>' [--meta '<json>']

  Примеры:
    gophkeeper set --type login_password --title OZON --payload '{"login":"ivan","password":"secret123","url":"https://ozon.ru"}'
    gophkeeper set --type text --title "GitHub token" --payload '{"text":"ghp_xxxxxxxxxxxxxxxxxxxx"}'
    gophkeeper set --type binary --title "TLS cert" --payload '{"data":"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCg==","encoding":"base64"}'
    gophkeeper set --type bank_card --title "Visa personal" --payload '{"card_number":"4111111111111111","holder":"IVAN IVANOV","exp_month":12,"exp_year":2027,"cvv":"123"}'
    gophkeeper set --type otp --title "Google OTP" --payload '{"issuer":"Google","account":"ivan@gmail.com","secret":"JBSWY3DPEHPK3PXP","digits":6,"period":30}'
    gophkeeper set --type text --title "Server note" --payload '{"text":"root access"}' --meta '{"env":"prod","owner":"devops"}'

Update <id>:
  Обновляет существующий секрет по ID.
  Можно передавать только те поля, которые требуется изменить.
  
  Формат:
    gophkeeper update <id> [--type <type> --title <title> --payload '<json>' --meta '<json>']

  Пример:
    gophkeeper update 1 --title "yandex my love"

Delete <id>:
  Удаляет секрет по ID.
  gophkeeper delete 1
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

			app.Secrets = memory.NewSecrets()

			sp, err := memory.DefaultSecretsPath()
			if err != nil {
				return err
			}
			app.SecretsPath = sp

			// загрузим локальные secrets (если файл есть)
			if err := memory.LoadFromFile(app.SecretsPath, app.Secrets); err != nil {
				return err
			}

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

	cmd.AddCommand(SecretSync(app))
	cmd.AddCommand(SecretGet(app))
	cmd.AddCommand(SecretCreate(app))
	cmd.AddCommand(SecretUpdate(app))
	cmd.AddCommand(SecretDelete(app))

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
