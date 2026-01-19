package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/cli"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/memory"
	"github.com/spf13/cobra"
)

func withDeps(t *testing.T, fn func()) {
	t.Helper()

	origNew := cli.NewAPIClient
	origEnc := cli.EncryptPayload
	origRead := cli.ReadMasterPassword
	origSave := cli.SaveSecretsToFile

	t.Cleanup(func() {
		cli.NewAPIClient = origNew
		cli.EncryptPayload = origEnc
		cli.ReadMasterPassword = origRead
		cli.SaveSecretsToFile = origSave
	})

	fn()
}

func TestSecretCreate_Success_NoMeta(t *testing.T) {
	withDeps(t, func() {
		// перехватим входящий JSON запроса
		var got map[string]any

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/secrets" {
				t.Fatalf("expected /secrets, got %s", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatalf("decode request: %v", err)
			}

			// ВАЖНО: отдаём сырой JSON, а не SecretResponse struct literal
			now := time.Now().Format(time.RFC3339Nano)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"id":"11111111-1111-1111-1111-111111111111",
				"type":"text",
				"title":"E2E text",
				"payload":"CIPHERTEXT",
				"version":1,
				"updated_at":"` + now + `",
				"created_at":"` + now + `"
			}`))
		}))
		defer srv.Close()

		cli.NewAPIClient = func(_ string) *api.Client { return api.NewClient(srv.URL) }
		cli.ReadMasterPassword = func(_ *cobra.Command, _ bool) (string, error) { return "pw", nil }
		cli.EncryptPayload = func(_ string, _ []byte) ([]byte, error) { return []byte("CIPHERTEXT"), nil }

		saved := false
		cli.SaveSecretsToFile = func(_ string, _ *memory.SecretsStore) error {
			saved = true
			return nil
		}

		app := &cli.App{
			ServerURL:   srv.URL,
			SecretsPath: filepath.Join(t.TempDir(), "secrets.json"),
			Secrets:     memory.NewSecrets(),
			Creds:       &config.Credentials{AccessToken: "token"},
		}

		cmd := cli.SecretCreate(app)
		cmd.SetArgs([]string{"--type", "text", "--title", "E2E text", "--payload", `{"text":"hello"}`})

		var out bytes.Buffer
		cmd.SetOut(&out)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		// Проверяем, что payload ушёл именно "зашифрованный"
		if got["payload"] != "CIPHERTEXT" {
			t.Fatalf("payload mismatch, got=%v", got["payload"])
		}

		// meta не должен улетать, если флаг не указан
		if _, ok := got["meta"]; ok {
			t.Fatalf("meta should not be present in request")
		}

		if !saved {
			t.Fatalf("expected SaveToFile called")
		}

		if !strings.Contains(out.String(), "created secret 11111111-1111-1111-1111-111111111111 (v1)") {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})
}
