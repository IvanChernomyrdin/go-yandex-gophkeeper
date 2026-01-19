package cli

import (
	"encoding/base64"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

// SecretGet создаёт CLI-команду для просмотра локально сохранённых секретов.
//
// Команда работает только с локальным хранилищем (secrets-файл) и не обращается к серверу.
//
// Режимы работы:
//   - без аргументов печатает список секретов: ID, type, title, version, updated_at;
//   - с аргументом <id> печатает подробную информацию об одном секрете.
//
// По умолчанию payload выводится как ciphertext (base64-строка), ровно в том виде,
// как он хранится на сервере и локально (E2E).
//
// Если указан флаг --decrypt, команда расшифрует payload и выведет plaintext.
// Для расшифровки требуется master password:
//   - по умолчанию запрашивается интерактивно (скрытый ввод);
//   - для скриптов/CI доступен режим чтения пароля из STDIN через --master-password-stdin.
//
// Важно:
//
//	sec.Payload хранится как base64-строка, поэтому перед расшифровкой выполняется base64 decode.
//
// В случае некорректного base64 вернётся ошибка.
//
// Примеры:
//
//	# список локальных секретов
//	gophkeeper get
//
//	# один секрет по ID (ciphertext)
//	gophkeeper get <uuid>
//
//	# один секрет по ID (plaintext)
//	gophkeeper get <uuid> --decrypt
//
//	# для скриптов (пароль из STDIN)
//	echo "MASTER_PASS" | gophkeeper get <uuid> --decrypt --master-password-stdin
func SecretGet(app *App) *cobra.Command {
	var decrypt bool
	var passwordFromStdin bool

	cmd := &cobra.Command{
		Use:   "get [id]",
		Short: "Получить локальные секреты (список или один по ID)",
		Long: `Показывает локально сохранённые секреты.

Без аргументов печатает список (ID, type, title, version, updated_at).
С ID печатает один секрет. По умолчанию payload выводится как ciphertext (base64 string),
как он хранится на сервере (E2E).
Если указать --decrypt, payload будет расшифрован (попросит master password).

Примеры:
  gophkeeper get
  gophkeeper get <uuid>
  gophkeeper get <uuid> --decrypt
`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				items := app.Secrets.List()
				sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

				if len(items) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "no local secrets (run: gophkeeper sync)")
					return nil
				}

				for _, s := range items {
					fmt.Fprintf(cmd.OutOrStdout(),
						"%s\t%s\t%s\tv%d\t%s\n",
						s.ID, s.Type, s.Title, s.Version, s.UpdatedAt.Format("2006-01-02 15:04:05"),
					)
				}
				return nil
			}

			id := args[0]
			sec, err := app.Secrets.Get(id)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(),
				"ID: %s\nType: %s\nTitle: %s\nVersion: %d\nUpdatedAt: %s\nCreatedAt: %s\n",
				sec.ID, sec.Type, sec.Title, sec.Version,
				sec.UpdatedAt.Format("2006-01-02 15:04:05"),
				sec.CreatedAt.Format("2006-01-02 15:04:05"),
			)
			if sec.Meta != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Meta: %s\n", *sec.Meta)
			}

			if !decrypt {
				fmt.Fprintf(cmd.OutOrStdout(), "Payload(ciphertext base64): %s\n", sec.Payload)
				return nil
			}

			pw, err := ReadMasterPassword(cmd, passwordFromStdin)
			if err != nil {
				return err
			}

			blob, err := base64.StdEncoding.DecodeString(sec.Payload)
			if err != nil {
				return fmt.Errorf("payload is not valid base64: %w", err)
			}

			plain, err := DecryptPayload(pw, blob)
			if err != nil {
				return fmt.Errorf("decrypt secret %s failed: %w", sec.ID, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Payload(plaintext): %s\n", string(plain))
			return nil
		},
	}

	cmd.Flags().BoolVar(&decrypt, "decrypt", false, "decrypt payload before printing (asks for master password)")
	cmd.Flags().BoolVar(&passwordFromStdin, "master-password-stdin", false, "read master password from STDIN (for scripts)")
	return cmd
}
