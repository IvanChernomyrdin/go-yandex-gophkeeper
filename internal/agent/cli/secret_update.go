// internal/agent/cli/secret_update.go
package cli

import (
	"encoding/base64"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/memory"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/models"
)

// SecretUpdate создаёт CLI-команду для обновления секрета на сервере и локально.
//
// Команда обновляет секрет по ID на сервере и синхронизирует локальное хранилище.
// Обновлять можно только выбранные поля: type, title, payload, meta.
// Payload перед отправкой шифруется на клиенте с использованием master password
// и отправляется как base64(ciphertext).
//
// Используется optimistic locking:
// версия берётся из локального стора и отправляется в запросе.
// Если локальная версия устарела, сервер вернёт conflict.
//
// Локальное обновление выполняется в два шага:
//  1. частично обновляет локальный secret через UpdateFromDB (type/title/payload);
//  2. выполняет sync и ReplaceAll, чтобы версия/updated_at/meta точно совпали с сервером.
//
// Требования:
//   - пользователь должен быть залогинен (access token сохранён локально);
//   - секрет должен быть синхронизирован локально (иначе команда попросит выполнить sync);
//   - должен быть указан хотя бы один флаг обновления: --type/--title/--payload/--meta.
//
// Примеры:
//
//	gophkeeper update <uuid> --title "new title"
//	gophkeeper update <uuid> --payload '{"text":"new"}'
//	gophkeeper update <uuid> --title "t" --payload '{"text":"x"}'
//
// В случае успеха выводит: "updated secret <id>".
func SecretUpdate(app *App) *cobra.Command {
	var (
		typ        string
		title      string
		payloadStr string
		meta       string

		setType, setTitle, setPayload, setMeta bool
		passwordFromStdin                      bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Обновить секрет на сервере и локально (ciphertext)",
		Long: `Обновляет секрет по ID на сервере и обновляет локальное хранилище.

Optimistic locking:
  версия берётся из локального стора и отправляется в запросе.
  Если версия устарела — сервер вернёт conflict.

Примеры:
  gophkeeper update <uuid> --title "new title"
  gophkeeper update <uuid> --payload '{"text":"new"}'
  gophkeeper update <uuid> --title "t" --payload '{"text":"x"}'
`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.Creds == nil || app.Creds.AccessToken == "" {
				return fmt.Errorf("no access_token, run: gophkeeper login")
			}
			id := args[0]

			// Берём локальный секрет, чтобы взять Version
			sec, err := app.Secrets.Get(id)
			if err != nil {
				return fmt.Errorf("secret %s not found locally (run: gophkeeper sync): %w", id, err)
			}

			// PATCH поля
			var (
				typePtr    *string
				titlePtr   *string
				payloadPtr *string
				metaPtr    *string
			)

			if setType {
				typePtr = &typ
			}
			if setTitle {
				titlePtr = &title
			}
			if setMeta {
				metaPtr = &meta
			}
			if setPayload {
				pw, err := ReadMasterPassword(cmd, passwordFromStdin)
				if err != nil {
					return err
				}

				blob, err := EncryptPayload(pw, []byte(payloadStr))
				if err != nil {
					return fmt.Errorf("encrypt payload: %w", err)
				}
				b64 := base64.StdEncoding.EncodeToString(blob)
				payloadPtr = &b64
			}

			if !setType && !setTitle && !setPayload && !setMeta {
				return fmt.Errorf("nothing to update: set at least one flag")
			}

			// Запрос на сервер
			c := NewAPIClient(app.ServerURL)
			if _, err := c.UpdateSecret(app.Creds.AccessToken, id, models.UpdateSecretRequest{
				Type:    typePtr,
				Title:   titlePtr,
				Payload: payloadPtr,
				Meta:    metaPtr,
				Version: sec.Version,
			}); err != nil {
				return err
			}

			// Локально обновляем только то, что умеет UpdateFromDB (4 аргумента).
			// META сюда не пихаем — её подтянем sync'ом (или добавишь отдельный метод позже).
			if err := app.Secrets.UpdateFromDB(id, typePtr, titlePtr, payloadPtr); err != nil {
				return err
			}

			// Чтобы версия/updated_at/meta всегда совпали с сервером — делаем sync.
			// (Это убирает пляски с ReplaceOne, time.Now и т.д.)
			synced, err := c.Sync(app.Creds.AccessToken)
			if err != nil {
				return fmt.Errorf("update ok, but sync failed: %w", err)
			}

			secrets := make([]memory.Secret, 0, len(synced.Secrets))
			for _, s := range synced.Secrets {
				secrets = append(secrets, memory.Secret{
					ID:        s.ID,
					Type:      s.Type,
					Title:     s.Title,
					Payload:   s.Payload,
					Meta:      s.Meta,
					Version:   s.Version,
					UpdatedAt: s.UpdatedAt,
					CreatedAt: s.CreatedAt,
				})
			}
			// синхронизируем мапу
			app.Secrets.ReplaceAll(secrets)

			if err := SaveSecretsToFile(app.SecretsPath, app.Secrets); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "updated secret %s\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&typ, "type", "", "new type")
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&payloadStr, "payload", "", "new payload JSON/string (will be encrypted)")
	cmd.Flags().StringVar(&meta, "meta", "", "new meta JSON/string")
	cmd.Flags().BoolVar(&passwordFromStdin, "master-password-stdin", false, "read master password from STDIN (for scripts)")

	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		setType = cmd.Flags().Changed("type")
		setTitle = cmd.Flags().Changed("title")
		setPayload = cmd.Flags().Changed("payload")
		setMeta = cmd.Flags().Changed("meta")
	}

	return cmd
}
