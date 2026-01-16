package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/api"
)

// NewRegisterCmd создаёт CLI-команду для регистрации нового пользователя.
//
// Команда выполняет регистрацию пользователя на сервере GophKeeper
// с использованием email и пароля. Для выполнения команды необходимо
// указать обязательные флаги --email и --password.
//
// Пример использования:
//
//	gophkeeper register --email test@example.com --password StrongPass123
//
// В случае успешной регистрации пользователю выводится сообщение
// об успешном завершении операции.
func NewRegisterCmd(app *App) *cobra.Command {
	var email, password string

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Регистрация нового пользователя",
		Long: `Регистрация нового пользователя на сервере.

Пример:
  gophkeeper register --email test@example.com --password StrongPass123
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := api.NewClient(app.ServerURL)
			// выполняет добавление нового пользователя в бд
			_, err := c.Register(email, password)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "registration successful")
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "email for registration")
	cmd.Flags().StringVar(&password, "password", "", "password for registration")
	cmd.MarkFlagRequired("email")
	cmd.MarkFlagRequired("password")

	return cmd
}
