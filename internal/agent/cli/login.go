package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/config"
)

// NewLoginCmd создаёт CLI-команду для входа пользователя в систему.
//
// Команда выполняет аутентификацию пользователя на сервере GophKeeper,
// получает пару access/refresh токенов и сохраняет их в локальный
// конфигурационный файл.
//
// Для выполнения команды требуется указать обязательные флаги
// --email и --password.
//
// Пример использования:
//
//	gophkeeper login --email test@example.com --password StrongPass123
//
// В случае успешного выполнения токены сохраняются локально, а пользователю
// выводится сообщение об успешном входе.
func NewLoginCmd(app *App) *cobra.Command {
	var email, password string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Логин пользователя (получить access/refresh токены)",
		Long: `Логин пользователя.

Пример:
  gophkeeper login --email test@example.com --password StrongPass123
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// создаём API-клиент для общения с сервером
			c := api.NewClient(app.ServerURL)
			// выполняем логин пользователя
			resp, err := c.Login(email, password)
			if err != nil {
				return err
			}

			// сохраняем полученные токены в состоянии приложения
			app.Creds.AccessToken = resp.AccessToken
			app.Creds.RefreshToken = resp.RefreshToken

			// сохраняем токены в локальный конфигурационный файл
			if err := config.Save(app.CredsPath, app.Creds); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "login ok (tokens saved)")
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "email for login")
	cmd.Flags().StringVar(&password, "password", "", "password for login")
	cmd.MarkFlagRequired("email")
	cmd.MarkFlagRequired("password")

	return cmd
}
