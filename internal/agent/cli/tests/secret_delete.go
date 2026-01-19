package tests

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/api"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/cli"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/config"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/memory"
)

func withDeleteDeps(t *testing.T, fn func()) {
	t.Helper()

	origNew := cli.NewAPIClient
	origSave := cli.SaveSecretsToFile

	t.Cleanup(func() {
		cli.NewAPIClient = origNew
		cli.SaveSecretsToFile = origSave
	})

	fn()
}

func TestSecretDelete_Success_DeletesOnServerAndLocalAndSaves(t *testing.T) {
	withDeleteDeps(t, func() {
		const (
			secretID = "11111111-1111-1111-1111-111111111111"
			version  = int(7)
		)

		// fake server: expects DELETE /secrets/{id}?version=7
		var gotPath string
		var gotAuth string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Fatalf("expected DELETE, got %s", r.Method)
			}
			gotPath = r.URL.Path + "?" + r.URL.RawQuery
			gotAuth = r.Header.Get("Authorization")

			// emulate server success: 204 No Content
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		cli.NewAPIClient = func(_ string) *api.Client { return api.NewClient(srv.URL) }

		saved := false
		cli.SaveSecretsToFile = func(_ string, _ *memory.SecretsStore) error {
			saved = true
			return nil
		}

		// app with one local secret
		store := memory.NewSecrets()
		store.ReplaceAll([]memory.Secret{
			{ID: secretID, Version: version},
		})

		app := &cli.App{
			ServerURL:   srv.URL,
			SecretsPath: filepath.Join(t.TempDir(), "secrets.json"),
			Secrets:     store,
			Creds:       &config.Credentials{AccessToken: "token"},
		}

		cmd := cli.SecretDelete(app)
		cmd.SetArgs([]string{secretID})

		var out bytes.Buffer
		cmd.SetOut(&out)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		// request assertions
		if gotAuth != "Bearer token" {
			t.Fatalf("unexpected auth header: %q", gotAuth)
		}
		wantPath := fmt.Sprintf("/secrets/%s?version=%d", secretID, version)
		if gotPath != wantPath {
			t.Fatalf("unexpected path: got %q want %q", gotPath, wantPath)
		}

		// local store updated
		if _, err := app.Secrets.Get(secretID); err == nil {
			t.Fatalf("expected secret deleted locally")
		}

		// saved
		if !saved {
			t.Fatalf("expected SaveToFile called")
		}

		// output
		if !strings.Contains(out.String(), fmt.Sprintf("deleted secret %s (version=%d)", secretID, version)) {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})
}

func TestSecretDelete_Fails_NoAccessToken(t *testing.T) {
	withDeleteDeps(t, func() {
		app := &cli.App{
			ServerURL: "http://example",
			Secrets:   memory.NewSecrets(),
			Creds:     &config.Credentials{AccessToken: ""},
		}
		cmd := cli.SecretDelete(app)
		cmd.SetArgs([]string{"some-id"})
		if err := cmd.Execute(); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestSecretDelete_Fails_NotFoundLocally(t *testing.T) {
	withDeleteDeps(t, func() {
		app := &cli.App{
			ServerURL: "http://example",
			Secrets:   memory.NewSecrets(),
			Creds:     &config.Credentials{AccessToken: "token"},
		}
		cmd := cli.SecretDelete(app)
		cmd.SetArgs([]string{"missing-id"})
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not found locally") {
			t.Fatalf("expected not found locally error, got: %v", err)
		}
	})
}

func TestSecretDelete_Fails_ServerError_DoesNotDeleteLocalOrSave(t *testing.T) {
	withDeleteDeps(t, func() {
		const (
			secretID = "11111111-1111-1111-1111-111111111111"
			version  = int(1)
		)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"conflict"}`))
		}))
		defer srv.Close()

		cli.NewAPIClient = func(_ string) *api.Client { return api.NewClient(srv.URL) }

		saved := false
		cli.SaveSecretsToFile = func(_ string, _ *memory.SecretsStore) error {
			saved = true
			return nil
		}

		store := memory.NewSecrets()
		store.ReplaceAll([]memory.Secret{{ID: secretID, Version: version}})

		app := &cli.App{
			ServerURL:   srv.URL,
			SecretsPath: filepath.Join(t.TempDir(), "secrets.json"),
			Secrets:     store,
			Creds:       &config.Credentials{AccessToken: "token"},
		}

		cmd := cli.SecretDelete(app)
		cmd.SetArgs([]string{secretID})

		err := cmd.Execute()
		if err == nil {
			t.Fatalf("expected error")
		}

		// local must remain
		if _, getErr := app.Secrets.Get(secretID); getErr != nil {
			t.Fatalf("expected secret still exists locally on server error")
		}

		// save must NOT be called
		if saved {
			t.Fatalf("did not expect SaveToFile called on server error")
		}
	})
}
