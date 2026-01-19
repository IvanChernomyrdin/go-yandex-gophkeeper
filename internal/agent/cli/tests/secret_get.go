package tests

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/cli"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/memory"
	"github.com/spf13/cobra"
)

func withGetDeps(t *testing.T, fn func()) {
	t.Helper()

	origRead := cli.ReadMasterPassword
	origDec := cli.DecryptPayload

	t.Cleanup(func() {
		cli.ReadMasterPassword = origRead
		cli.DecryptPayload = origDec
	})

	fn()
}

func TestSecretGet_List_Empty(t *testing.T) {
	withGetDeps(t, func() {
		app := &cli.App{
			Secrets: memory.NewSecrets(),
		}

		cmd := cli.SecretGet(app)
		cmd.SetArgs([]string{})

		var out bytes.Buffer
		cmd.SetOut(&out)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		if !strings.Contains(out.String(), "no local secrets") {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})
}

func TestSecretGet_List_SortedByID(t *testing.T) {
	withGetDeps(t, func() {
		now := time.Date(2026, 1, 19, 15, 0, 0, 0, time.UTC)

		store := memory.NewSecrets()
		store.ReplaceAll([]memory.Secret{
			{ID: "b", Type: "text", Title: "B", Version: 1, UpdatedAt: now},
			{ID: "a", Type: "text", Title: "A", Version: 2, UpdatedAt: now},
		})

		app := &cli.App{Secrets: store}

		cmd := cli.SecretGet(app)
		cmd.SetArgs([]string{})

		var out bytes.Buffer
		cmd.SetOut(&out)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d: %q", len(lines), out.String())
		}
		if !strings.HasPrefix(lines[0], "a\t") {
			t.Fatalf("expected first line to be id=a, got: %q", lines[0])
		}
		if !strings.HasPrefix(lines[1], "b\t") {
			t.Fatalf("expected second line to be id=b, got: %q", lines[1])
		}
	})
}

func TestSecretGet_One_NoDecrypt_PrintsCiphertext(t *testing.T) {
	withGetDeps(t, func() {
		now := time.Date(2026, 1, 19, 15, 0, 0, 0, time.UTC)

		store := memory.NewSecrets()
		store.ReplaceAll([]memory.Secret{
			{
				ID:        "id1",
				Type:      "text",
				Title:     "T",
				Payload:   "BASE64CIPHERTEXT",
				Version:   3,
				UpdatedAt: now,
				CreatedAt: now,
			},
		})

		app := &cli.App{Secrets: store}

		cmd := cli.SecretGet(app)
		cmd.SetArgs([]string{"id1"})

		var out bytes.Buffer
		cmd.SetOut(&out)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		if !strings.Contains(out.String(), "Payload(ciphertext base64): BASE64CIPHERTEXT") {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})
}

func TestSecretGet_One_Decrypt_Success(t *testing.T) {
	withGetDeps(t, func() {
		now := time.Date(2026, 1, 19, 15, 0, 0, 0, time.UTC)

		blob := []byte("gk1BLOB")
		b64 := base64.StdEncoding.EncodeToString(blob)

		cli.ReadMasterPassword = func(_ *cobra.Command, _ bool) (string, error) {
			return "pw", nil
		}
		cli.DecryptPayload = func(pw string, raw []byte) ([]byte, error) {
			if pw != "pw" {
				t.Fatalf("unexpected pw: %q", pw)
			}
			if string(raw) != string(blob) {
				t.Fatalf("unexpected raw blob: %q", string(raw))
			}
			return []byte(`{"text":"hello"}`), nil
		}

		store := memory.NewSecrets()
		store.ReplaceAll([]memory.Secret{
			{
				ID:        "id1",
				Type:      "text",
				Title:     "T",
				Payload:   b64,
				Version:   1,
				UpdatedAt: now,
				CreatedAt: now,
			},
		})

		app := &cli.App{Secrets: store}

		cmd := cli.SecretGet(app)
		cmd.SetArgs([]string{"id1", "--decrypt"})

		var out bytes.Buffer
		cmd.SetOut(&out)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}

		if !strings.Contains(out.String(), `Payload(plaintext): {"text":"hello"}`) {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})
}

func TestSecretGet_One_Decrypt_Fails_InvalidBase64(t *testing.T) {
	withGetDeps(t, func() {
		now := time.Date(2026, 1, 19, 15, 0, 0, 0, time.UTC)

		store := memory.NewSecrets()
		store.ReplaceAll([]memory.Secret{
			{
				ID:        "id1",
				Type:      "text",
				Title:     "T",
				Payload:   "NOT_BASE64!!!",
				Version:   1,
				UpdatedAt: now,
				CreatedAt: now,
			},
		})

		cli.ReadMasterPassword = func(_ *cobra.Command, _ bool) (string, error) {
			return "pw", nil
		}
		cli.DecryptPayload = func(_ string, _ []byte) ([]byte, error) {
			t.Fatalf("DecryptPayload must not be called on invalid base64")
			return nil, nil
		}

		app := &cli.App{Secrets: store}

		cmd := cli.SecretGet(app)
		cmd.SetArgs([]string{"id1", "--decrypt"})

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "payload is not valid base64") {
			t.Fatalf("expected base64 error, got: %v", err)
		}
	})
}
