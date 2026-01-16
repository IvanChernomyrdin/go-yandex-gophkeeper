package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/config"
)

// NewRefreshCmd создаёт CLI-команду для обновления пары токенов.
//
// Команда использует сохранённый refresh токен для получения
// новой пары access/refresh токенов с сервера GophKeeper.
// Обновлённые токены сохраняются в локальный конфигурационный файл.
//
// Команда не принимает аргументов. Перед выполнением требуется,
// чтобы refresh токен уже был сохранён (например, после выполнения команды login).
//
// Пример использования:
//
//	gophkeeper refresh
//
// Если refresh токен отсутствует в конфигурации, команда завершится
// с ошибкой и предложит выполнить повторный вход (login).
func NewRefreshCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Обновить access токен по refresh токену",
		Long: `Обновляет access token по refresh token.

Пример:
  gophkeeper refresh
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.Creds.RefreshToken == "" {
				return fmt.Errorf("no refresh_token in config, run: gophkeeper login")
			}

			c := api.NewClient(app.ServerURL)
			// генерирует новый jwt по refresh
			resp, err := c.Refresh(app.Creds.RefreshToken)
			if err != nil {
				return err
			}
			// сохраняет в структуру
			app.Creds.AccessToken = resp.AccessToken
			app.Creds.RefreshToken = resp.RefreshToken
			// сохраняет локально
			if err := config.Save(app.CredsPath, app.Creds); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "refresh ok (tokens updated)")
			return nil
		},
	}

	return cmd
}
