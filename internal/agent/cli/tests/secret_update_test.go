package tests

import (
	"bytes"
	"encoding/base64"
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

func withUpdateDeps(t *testing.T, fn func()) {
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

func TestSecretUpdate_Fails_NoAccessToken(t *testing.T) {
	withUpdateDeps(t, func() {
		app := &cli.App{
			ServerURL: "http://example",
			Secrets:   memory.NewSecrets(),
			Creds:     &config.Credentials{AccessToken: ""},
		}
		cmd := cli.SecretUpdate(app)
		cmd.SetArgs([]string{"id1", "--title", "x"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "no access_token") {
			t.Fatalf("expected no access_token error, got: %v", err)
		}
	})
}

func TestSecretUpdate_Fails_NotFoundLocally(t *testing.T) {
	withUpdateDeps(t, func() {
		app := &cli.App{
			ServerURL: "http://example",
			Secrets:   memory.NewSecrets(),
			Creds:     &config.Credentials{AccessToken: "token"},
		}
		cmd := cli.SecretUpdate(app)
		cmd.SetArgs([]string{"missing-id", "--title", "x"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not found locally") {
			t.Fatalf("expected not found locally error, got: %v", err)
		}
	})
}

func TestSecretUpdate_Fails_NothingToUpdate(t *testing.T) {
	withUpdateDeps(t, func() {
		store := memory.NewSecrets()
		store.ReplaceAll([]memory.Secret{{ID: "id1", Version: 1}})

		app := &cli.App{
			ServerURL: "http://example",
			Secrets:   store,
			Creds:     &config.Credentials{AccessToken: "token"},
		}

		cmd := cli.SecretUpdate(app)
		cmd.SetArgs([]string{"id1"}) // no flags
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "nothing to update") {
			t.Fatalf("expected nothing to update error, got: %v", err)
		}
	})
}

func TestSecretUpdate_Success_TitleOnly_UpdatesLocalAndSyncs(t *testing.T) {
	withUpdateDeps(t, func() {
		// --- fake server ---
		// 1) PUT /secrets/id1 -> 204
		// 2) GET /secrets -> returns list
		now := time.Now().Format(time.RFC3339Nano)

		putCalled := 0
		getCalled := 0

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/secrets/id1":
				putCalled++
				w.WriteHeader(http.StatusNoContent)
				return
			case r.Method == http.MethodGet && r.URL.Path == "/secrets":
				getCalled++
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"secrets":[
						{"id":"id1","type":"text","title":"NEW","payload":"P","meta":null,"version":2,"updated_at":"` + now + `","created_at":"` + now + `"}
					]
				}`))
				return
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}))
		defer srv.Close()

		cli.NewAPIClient = func(_ string) *api.Client { return api.NewClient(srv.URL) }

		saved := false
		cli.SaveSecretsToFile = func(_ string, _ *memory.SecretsStore) error { saved = true; return nil }

		// local store has old version=1
		store := memory.NewSecrets()
		store.ReplaceAll([]memory.Secret{{ID: "id1", Type: "text", Title: "OLD", Payload: "P", Version: 1}})

		app := &cli.App{
			ServerURL:   srv.URL,
			SecretsPath: filepath.Join(t.TempDir(), "secrets.json"),
			Secrets:     store,
			Creds:       &config.Credentials{AccessToken: "token"},
		}

		cmd := cli.SecretUpdate(app)
		cmd.SetArgs([]string{"id1", "--title", "NEW"})

		var out bytes.Buffer
		cmd.SetOut(&out)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		if putCalled != 1 {
			t.Fatalf("expected PUT called once, got %d", putCalled)
		}
		if getCalled != 1 {
			t.Fatalf("expected GET /secrets called once (sync), got %d", getCalled)
		}
		if !saved {
			t.Fatalf("expected SaveToFile called")
		}

		// after sync ReplaceAll, store should contain id1 title NEW version 2
		sec, err := app.Secrets.Get("id1")
		if err != nil {
			t.Fatalf("expected secret in store, err=%v", err)
		}
		if sec.Title != "NEW" {
			t.Fatalf("expected title NEW, got %q", sec.Title)
		}
		if sec.Version != 2 {
			t.Fatalf("expected version 2, got %d", sec.Version)
		}

		if !strings.Contains(out.String(), "updated secret id1") {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})
}

func TestSecretUpdate_Success_Payload_EncryptsToBase64(t *testing.T) {
	withUpdateDeps(t, func() {
		now := time.Now().Format(time.RFC3339Nano)

		// capture request body of PUT
		var gotBody map[string]any

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/secrets/id1":
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
				w.WriteHeader(http.StatusNoContent)
				return
			case r.Method == http.MethodGet && r.URL.Path == "/secrets":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"secrets":[{"id":"id1","type":"text","title":"OLD","payload":"P","version":2,"updated_at":"` + now + `","created_at":"` + now + `"}]}`))
				return
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}))
		defer srv.Close()

		cli.NewAPIClient = func(_ string) *api.Client { return api.NewClient(srv.URL) }
		cli.ReadMasterPassword = func(_ *cobra.Command, _ bool) (string, error) { return "pw", nil }

		// encrypt returns raw blob; update should base64 encode it
		rawBlob := []byte("RAWBLOB")
		cli.EncryptPayload = func(pw string, pt []byte) ([]byte, error) {
			if pw != "pw" {
				t.Fatalf("unexpected pw: %q", pw)
			}
			if string(pt) != `{"text":"new"}` {
				t.Fatalf("unexpected plaintext: %q", string(pt))
			}
			return rawBlob, nil
		}

		cli.SaveSecretsToFile = func(_ string, _ *memory.SecretsStore) error { return nil }

		store := memory.NewSecrets()
		store.ReplaceAll([]memory.Secret{{ID: "id1", Type: "text", Title: "OLD", Payload: "P", Version: 1}})

		app := &cli.App{
			ServerURL:   srv.URL,
			SecretsPath: filepath.Join(t.TempDir(), "secrets.json"),
			Secrets:     store,
			Creds:       &config.Credentials{AccessToken: "token"},
		}

		cmd := cli.SecretUpdate(app)
		cmd.SetArgs([]string{"id1", "--payload", `{"text":"new"}`, "--master-password-stdin"})

		// master password stubbed, stdin flag just marks passwordFromStdin=true in your code
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		wantB64 := base64.StdEncoding.EncodeToString(rawBlob)
		if gotBody["payload"] != wantB64 {
			t.Fatalf("expected payload base64 %q, got %v", wantB64, gotBody["payload"])
		}
	})
}

func TestSecretUpdate_Fails_UpdateOkButSyncFails(t *testing.T) {
	withUpdateDeps(t, func() {
		// PUT ok, GET fails
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut && r.URL.Path == "/secrets/id1" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if r.Method == http.MethodGet && r.URL.Path == "/secrets" {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"boom"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		cli.NewAPIClient = func(_ string) *api.Client { return api.NewClient(srv.URL) }
		cli.SaveSecretsToFile = func(_ string, _ *memory.SecretsStore) error {
			t.Fatalf("SaveToFile must not be called if sync failed")
			return nil
		}

		store := memory.NewSecrets()
		store.ReplaceAll([]memory.Secret{{ID: "id1", Version: 1}})

		app := &cli.App{
			ServerURL:   srv.URL,
			SecretsPath: filepath.Join(t.TempDir(), "secrets.json"),
			Secrets:     store,
			Creds:       &config.Credentials{AccessToken: "token"},
		}

		cmd := cli.SecretUpdate(app)
		cmd.SetArgs([]string{"id1", "--title", "X"})

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "update ok, but sync failed") {
			t.Fatalf("expected sync failed error, got: %v", err)
		}
	})
}
