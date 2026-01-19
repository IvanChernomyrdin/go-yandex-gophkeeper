package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/memory"
)

// SecretSync создаёт CLI-команду для синхронизации локальных секретов с сервером.
//
// Команда запрашивает у сервера полный список секретов текущего пользователя
// и сохраняет их локально только в зашифрованном виде (ciphertext).
// Расшифровка payload выполняется отдельно командой:
//
//	gophkeeper get <id> --decrypt
//
// Требования:
//   - пользователь должен быть залогинен (access token сохранён локально).
//
// Поведение:
//  1. выполняет запрос Sync к серверу с access token;
//  2. преобразует ответ сервера в локальные записи memory.Secret;
//  3. перезаписывает локальный secrets store (ReplaceAll);
//  4. сохраняет secrets store в файл;
//  5. выводит: "synced N secrets (ciphertext stored locally)".
//
// Защита от несовпадения моделей:
// если сервер вернул элемент без ID (пустая строка), команда завершится ошибкой
// вида "sync: server returned secret with empty id..." — это помогает быстро поймать
// рассинхрон JSON-модели между сервером и клиентом.
//
// Пример:
//
//	gophkeeper sync
func SecretSync(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Синхронизация секретов с сервером",
		Long: `Синхронизация локальных секретов с сервером.

Загружает все секреты и сохраняет их локально только в зашифрованном виде (ciphertext).
Расшифровка выполняется отдельно: gophkeeper get <id> --decrypt

Пример:
  gophkeeper sync
`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.Creds == nil || app.Creds.AccessToken == "" {
				return fmt.Errorf("no access_token, run: gophkeeper login")
			}

			c := NewAPIClient(app.ServerURL)
			result, err := c.Sync(app.Creds.AccessToken)
			if err != nil {
				return err
			}

			secrets := make([]memory.Secret, 0, len(result.Secrets))
			for i, s := range result.Secrets {
				// Стоп-кран: если ID пустой — значит модель ответа не совпала с JSON
				if s.ID == "" {
					return fmt.Errorf("sync: server returned secret with empty id at index %d (model mismatch)", i)
				}

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

			app.Secrets.ReplaceAll(secrets)

			if err := SaveSecretsToFile(app.SecretsPath, app.Secrets); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "synced %d secrets (ciphertext stored locally)\n", len(secrets))
			return nil
		},
	}

	return cmd
}

// readMasterPassword читает master password для шифрования/расшифровки.
//
// Режимы:
//   - fromStdin=true: читает пароль из STDIN полностью (удобно для скриптов/CI);
//   - fromStdin=false: читает пароль интерактивно из терминала со скрытым вводом.
//
// Важно:
//   - если fromStdin=false, но stdin не является терминалом, функция вернёт ошибку
//     "stdin is not a terminal; use --master-password-stdin".
//   - пустой пароль считается ошибкой.
func readMasterPassword(cmd *cobra.Command, fromStdin bool) (string, error) {
	if fromStdin {
		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", fmt.Errorf("read master password from stdin: %w", err)
		}
		pw := bytes.TrimRight(b, "\r\n")
		if len(pw) == 0 {
			return "", errors.New("empty master password on stdin")
		}
		return string(pw), nil
	}

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("stdin is not a terminal; use --master-password-stdin")
	}

	fmt.Fprint(cmd.ErrOrStderr(), "Master password: ")
	pwBytes, err := term.ReadPassword(fd)
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", fmt.Errorf("read master password: %w", err)
	}

	pw := strings.TrimSpace(string(pwBytes))
	if pw == "" {
		return "", errors.New("empty master password")
	}
	return pw, nil
}
