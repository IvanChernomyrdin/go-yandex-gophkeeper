package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/config"
)

func TestDefaultPath_ReturnsPathInHomeDir(t *testing.T) {
	p, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath returned error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir returned error: %v", err)
	}

	want := filepath.Join(home, ".gophkeeper", "credentials.json")
	if p != want {
		t.Fatalf("expected %q, got %q", want, p)
	}
}

func TestLoad_FileNotExists_ReturnsEmptyCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	p := filepath.Join(tmpDir, "no-such-file.json")

	creds, err := config.Load(p)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if creds == nil {
		t.Fatalf("expected non-nil creds")
	}
	if creds.AccessToken != "" || creds.RefreshToken != "" {
		t.Fatalf("expected empty creds, got %+v", *creds)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	p := filepath.Join(tmpDir, "a", "credentials.json") // вложенная директория

	want := &config.Credentials{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
	}

	if err := config.Save(p, want); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	got, err := config.Load(p)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got.AccessToken != want.AccessToken {
		t.Fatalf("expected AccessToken=%q, got %q", want.AccessToken, got.AccessToken)
	}
	if got.RefreshToken != want.RefreshToken {
		t.Fatalf("expected RefreshToken=%q, got %q", want.RefreshToken, got.RefreshToken)
	}

	// проверим права файла только на linux, на винде он гарантирует эти права.
	if runtime.GOOS != "windows" {
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("Stat returned error: %v", err)
		}
		perm := st.Mode().Perm()

		// ожидаем, что группа/остальные не имеют доступа
		if perm&0o077 != 0 {
			t.Fatalf("expected no group/other permissions, got %o", perm)
		}
	}
}

func TestLoad_BadJSON_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	p := filepath.Join(tmpDir, "credentials.json")

	if err := os.WriteFile(p, []byte("{bad-json"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := config.Load(p)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
