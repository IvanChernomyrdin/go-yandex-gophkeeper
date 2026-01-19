package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/memory"
	sharedModels "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/models"
)

// SecretCreate создаёт CLI-команду для создания нового секрета на сервере и
// сохранения его локально в зашифрованном виде (ciphertext).
//
// Команда отправляет на сервер тип, заголовок и зашифрованный payload.
// Шифрование выполняется на клиенте с использованием master password.
// Master password не передаётся серверу и не должен передаваться флагом,
// чтобы не утекать в shell history.
//
// По умолчанию master password запрашивается интерактивно (скрытый ввод).
// Для скриптов/CI доступен режим чтения пароля из STDIN через флаг
// --master-password-stdin.
//
// Обязательные флаги:
//
//	--type     — тип секрета (например: text, login_password, binary, bank_card, otp)
//	--title    — название секрета
//	--payload  — исходные данные секрета (JSON/строка), которые будут зашифрованы
//
// Необязательные флаги:
//
//	--meta     — дополнительная мета-информация (JSON/строка), передаётся на сервер как есть
//	--master-password-stdin — читать master password из STDIN (удобно для автоматизации)
//
// Примеры использования:
//
//	# Произвольный текст (API token, заметка и т.п.)
//	gophkeeper set --type text --title "GitHub token" --payload '{"text":"ghp_xxx"}'
//
//	# Логин/пароль (например, для сайта)
//	gophkeeper set --type login_password --title "OZON" --payload '{"login":"ivan","password":"secret","url":"https://ozon.ru"}'
//
//	# С использованием meta (например, теги окружения/владельца)
//	gophkeeper set --type text --title "Server note" --payload '{"text":"root access"}' --meta '{"env":"prod","owner":"devops"}'
//
//	# Для скриптов (пароль читается из STDIN)
//	echo "MASTER_PASS" | gophkeeper set --type text --title "note" --payload '{"text":"hello"}' --master-password-stdin
//
// В случае успешного выполнения команда:
//  1. получает от сервера ID, version и timestamps;
//  2. сохраняет секрет локально (payload в виде ciphertext) в файл secrets;
//  3. выводит сообщение вида: "created secret <id> (v<version>)".
func SecretCreate(app *App) *cobra.Command {
	var (
		typ               string
		title             string
		payloadStr        string
		meta              string
		passwordFromStdin bool
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Создать новый секрет на сервере и сохранить локально (ciphertext)",
		Long: `Создаёт новый секрет на сервере и сохраняет его локально (ciphertext).

Payload шифруется на клиенте с использованием master password.
Master password не передаётся флагом (чтобы не утекать в history).
По умолчанию пароль запрашивается интерактивно (скрытый ввод).
Для скриптов: --master-password-stdin читает пароль из STDIN.

Примеры:
  gophkeeper set --type text --title "GitHub token" --payload '{"text":"ghp_xxx"}'
  gophkeeper set --type login_password --title "OZON" --payload '{"login":"ivan","password":"secret","url":"https://ozon.ru"}'
`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.Creds == nil || app.Creds.AccessToken == "" {
				return fmt.Errorf("no access_token, run: gophkeeper login")
			}
			if typ == "" || title == "" || payloadStr == "" {
				return fmt.Errorf("--type, --title and --payload are required")
			}

			pw, err := ReadMasterPassword(cmd, passwordFromStdin)
			if err != nil {
				return err
			}

			cipherBytes, err := EncryptPayload(pw, []byte(payloadStr))
			if err != nil {
				return fmt.Errorf("encrypt payload: %w", err)
			}
			cipherStr := string(cipherBytes)

			var metaPtr *string
			if cmd.Flags().Changed("meta") {
				metaPtr = &meta
			}

			c := NewAPIClient(app.ServerURL)

			created, err := c.CreateSecret(app.Creds.AccessToken, sharedModels.CreateSecretRequest{
				Type:    typ,
				Title:   title,
				Payload: cipherStr,
				Meta:    metaPtr,
			})
			if err != nil {
				return err
			}
			if created.ID == "" {
				return fmt.Errorf("server returned empty id on create")
			}

			local := memory.Secret{
				ID:        created.ID,
				Type:      typ,
				Title:     title,
				Payload:   cipherStr,
				Meta:      metaPtr,
				Version:   created.Version,
				UpdatedAt: created.UpdatedAt,
				CreatedAt: created.UpdatedAt,
			}

			app.Secrets.ReplaceAll(append(app.Secrets.List(), local))

			if err := SaveSecretsToFile(app.SecretsPath, app.Secrets); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "created secret %s (v%d)\n", created.ID, created.Version)
			return nil
		},
	}

	cmd.Flags().StringVar(&typ, "type", "", "secret type")
	cmd.Flags().StringVar(&title, "title", "", "secret title")
	cmd.Flags().StringVar(&payloadStr, "payload", "", "payload JSON/string (will be encrypted)")
	cmd.Flags().StringVar(&meta, "meta", "", "optional meta JSON/string")
	cmd.Flags().BoolVar(&passwordFromStdin, "master-password-stdin", false, "read master password from STDIN (for scripts)")

	return cmd
}
