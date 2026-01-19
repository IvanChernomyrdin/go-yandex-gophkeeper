package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// SecretDelete создаёт CLI-команду для удаления секрета на сервере и локально.
//
// Команда удаляет секрет по ID на сервере, а затем удаляет его из локального
// хранилища и сохраняет обновлённый secrets-файл.
//
// Для удаления используется optimistic locking:
// версия (Version) берётся из локально сохранённого секрета и отправляется на сервер
// в запросе DELETE /secrets/{id}?version=N.
// Если локальная версия устарела (секрет был изменён на сервере), сервер вернёт conflict.
//
// Требования:
//   - пользователь должен быть залогинен (access token сохранён локально);
//   - секрет должен быть синхронизирован локально (иначе команда попросит выполнить sync).
//
// Пример использования:
//
//	gophkeeper delete 7a0a4a6a-a7bf-42c0-8cdf-2be8583d180e
//
// В случае успешного выполнения команда:
//  1. удаляет секрет на сервере;
//  2. удаляет секрет из локального стора;
//  3. сохраняет локальный файл secrets;
//  4. выводит сообщение вида: "deleted secret <id> (version=<N>)".
func SecretDelete(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Удалить секрет на сервере и локально",
		Long: `Удаляет секрет по ID на сервере и в локальном хранилище.

Версия берётся из локально сохранённого секрета (optimistic locking):
  DELETE /secrets/{id}?version=N

Пример:
  gophkeeper delete <uuid>
(если секрета нет локально — сначала сделай: gophkeeper sync)
`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.Creds == nil || app.Creds.AccessToken == "" {
				return fmt.Errorf("no access_token, run: gophkeeper login")
			}

			id := args[0]

			sec, err := app.Secrets.Get(id)
			if err != nil {
				return fmt.Errorf("secret %s not found locally (run: gophkeeper sync): %w", id, err)
			}

			c := NewAPIClient(app.ServerURL)
			if err := c.DeleteSecret(app.Creds.AccessToken, id, sec.Version); err != nil {
				return err
			}

			if err := app.Secrets.Delete(id); err != nil {
				return err
			}
			if err := SaveSecretsToFile(app.SecretsPath, app.Secrets); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "deleted secret %s (version=%d)\n", id, sec.Version)
			return nil
		},
	}
	return cmd
}
