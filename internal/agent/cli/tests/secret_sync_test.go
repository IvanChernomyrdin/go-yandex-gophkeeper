package tests

import (
	"bytes"
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
)

func withSyncDeps(t *testing.T, fn func()) {
	t.Helper()

	origNew := cli.NewAPIClient
	origSave := cli.SaveSecretsToFile

	t.Cleanup(func() {
		cli.NewAPIClient = origNew
		cli.SaveSecretsToFile = origSave
	})

	fn()
}

func TestSecretSync_Success_SavesAndReplacesLocalStore(t *testing.T) {
	withSyncDeps(t, func() {
		// сервер отдаёт 2 секрета
		now := time.Now().Format(time.RFC3339Nano)

		var gotAuth string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/secrets" {
				t.Fatalf("expected /secrets, got %s", r.URL.Path)
			}
			gotAuth = r.Header.Get("Authorization")

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"secrets":[
					{"id":"a","type":"text","title":"A","payload":"P1","version":1,"updated_at":"` + now + `","created_at":"` + now + `"},
					{"id":"b","type":"text","title":"B","payload":"P2","version":2,"updated_at":"` + now + `","created_at":"` + now + `"}
				]
			}`))
		}))
		defer srv.Close()

		cli.NewAPIClient = func(_ string) *api.Client { return api.NewClient(srv.URL) }

		saved := false
		cli.SaveSecretsToFile = func(_ string, _ *memory.SecretsStore) error {
			saved = true
			return nil
		}

		// локально изначально что-то лежит — должно перезаписаться
		store := memory.NewSecrets()
		store.ReplaceAll([]memory.Secret{{ID: "old", Type: "text", Title: "OLD", Payload: "X", Version: 9}})

		app := &cli.App{
			ServerURL:   srv.URL,
			SecretsPath: filepath.Join(t.TempDir(), "secrets.json"),
			Secrets:     store,
			Creds:       &config.Credentials{AccessToken: "token"},
		}

		cmd := cli.SecretSync(app)
		var out bytes.Buffer
		cmd.SetOut(&out)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		if gotAuth != "Bearer token" {
			t.Fatalf("unexpected auth: %q", gotAuth)
		}
		if !saved {
			t.Fatalf("expected SaveToFile called")
		}

		// store должен содержать только a и b
		items := app.Secrets.List()
		if len(items) != 2 {
			t.Fatalf("expected 2 secrets in store, got %d", len(items))
		}
		// проверим, что "old" пропал
		if _, err := app.Secrets.Get("old"); err == nil {
			t.Fatalf("expected old secret to be replaced")
		}

		if !strings.Contains(out.String(), "synced 2 secrets") {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})
}

func TestSecretSync_Fails_NoAccessToken(t *testing.T) {
	withSyncDeps(t, func() {
		app := &cli.App{
			ServerURL: "http://example",
			Secrets:   memory.NewSecrets(),
			Creds:     &config.Credentials{AccessToken: ""},
		}

		cmd := cli.SecretSync(app)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "no access_token") {
			t.Fatalf("expected no access_token error, got: %v", err)
		}
	})
}

func TestSecretSync_Fails_ModelMismatch_EmptyID(t *testing.T) {
	withSyncDeps(t, func() {
		now := time.Now().Format(time.RFC3339Nano)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// id пустой -> должен сработать стоп-кран
			_, _ = w.Write([]byte(`{
				"secrets":[
					{"id":"","type":"text","title":"A","payload":"P1","version":1,"updated_at":"` + now + `","created_at":"` + now + `"}
				]
			}`))
		}))
		defer srv.Close()

		cli.NewAPIClient = func(_ string) *api.Client { return api.NewClient(srv.URL) }
		cli.SaveSecretsToFile = func(_ string, _ *memory.SecretsStore) error {
			t.Fatalf("SaveToFile must not be called on model mismatch")
			return nil
		}

		app := &cli.App{
			ServerURL:   srv.URL,
			SecretsPath: filepath.Join(t.TempDir(), "secrets.json"),
			Secrets:     memory.NewSecrets(),
			Creds:       &config.Credentials{AccessToken: "token"},
		}

		cmd := cli.SecretSync(app)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "empty id") {
			t.Fatalf("expected empty id error, got: %v", err)
		}
	})
}
