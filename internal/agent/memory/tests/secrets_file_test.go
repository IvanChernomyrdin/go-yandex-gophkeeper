package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/memory"
)

func TestDefaultSecretsPath_ReturnsHomeDotGophkeeper(t *testing.T) {
	p, err := memory.DefaultSecretsPath()
	if err != nil {
		t.Fatalf("DefaultSecretsPath error: %v", err)
	}
	if p == "" {
		t.Fatalf("expected non-empty path")
	}

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".gophkeeper", "secrets.json")
	if filepath.Clean(p) != filepath.Clean(want) {
		t.Fatalf("unexpected path: got=%q want=%q", p, want)
	}
}

func TestSaveToFile_CreatesDirAndWritesJSON_AndLoadFromFile_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nested", "secrets.json")

	store := memory.NewSecrets()
	now := time.Now().UTC()

	meta := `{"env":"test"}`
	store.ReplaceAll([]memory.Secret{
		{
			ID:        "id1",
			Type:      "text",
			Title:     "t1",
			Payload:   "cipher1",
			Meta:      &meta,
			Version:   2,
			UpdatedAt: now,
			CreatedAt: now,
		},
		{
			ID:        "id2",
			Type:      "login_password",
			Title:     "t2",
			Payload:   "cipher2",
			Meta:      nil,
			Version:   1,
			UpdatedAt: now.Add(-time.Minute),
			CreatedAt: now.Add(-time.Minute),
		},
	})

	if err := memory.SaveToFile(path, store); err != nil {
		t.Fatalf("SaveToFile error: %v", err)
	}

	// файл должен существовать
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("expected non-empty file")
	}

	// проверим, что JSON парсится
	var dump memory.SecretsDump
	if err := json.Unmarshal(b, &dump); err != nil {
		t.Fatalf("saved JSON invalid: %v", err)
	}
	if len(dump.Secrets) != 2 {
		t.Fatalf("expected 2 secrets in file, got %d", len(dump.Secrets))
	}

	// round-trip load
	store2 := memory.NewSecrets()
	if err := memory.LoadFromFile(path, store2); err != nil {
		t.Fatalf("LoadFromFile error: %v", err)
	}
	if len(store2.List()) != 2 {
		t.Fatalf("expected 2 secrets after load, got %d", len(store2.List()))
	}

	// точечно проверим один секрет
	got, err := store2.Get("id1")
	if err != nil {
		t.Fatalf("Get after load error: %v", err)
	}
	if got.Payload != "cipher1" || got.Type != "text" || got.Version != 2 {
		t.Fatalf("unexpected loaded secret: %+v", got)
	}
	if got.Meta == nil || *got.Meta != meta {
		t.Fatalf("expected meta=%q, got %+v", meta, got.Meta)
	}
}

func TestLoadFromFile_NotExists_ReturnsNil(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nope.json")

	store := memory.NewSecrets()
	if err := memory.LoadFromFile(path, store); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// store не должен измениться
	if len(store.List()) != 0 {
		t.Fatalf("expected empty store")
	}
}

func TestLoadFromFile_BadJSON_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.json")

	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatalf("write bad file: %v", err)
	}

	store := memory.NewSecrets()
	if err := memory.LoadFromFile(path, store); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSaveToFile_Permissions_BestEffort(t *testing.T) {
	// На Windows chmod семантика другая, этот тест пропускаем.
	if runtime.GOOS == "windows" {
		t.Skip("permissions are not reliably testable on Windows")
	}

	tmp := t.TempDir()
	path := filepath.Join(tmp, "nested", "secrets.json")

	store := memory.NewSecrets()
	store.ReplaceAll([]memory.Secret{
		{ID: "id1", Type: "text", Title: "t", Payload: "p", Version: 1, UpdatedAt: time.Now(), CreatedAt: time.Now()},
	})

	if err := memory.SaveToFile(path, store); err != nil {
		t.Fatalf("SaveToFile error: %v", err)
	}

	// проверка прав директории
	dir := filepath.Dir(path)
	dinfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if dinfo.Mode().Perm() != 0o700 {
		t.Fatalf("expected dir perm 0700, got %o", dinfo.Mode().Perm())
	}

	// проверка прав файла
	finfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if finfo.Mode().Perm() != 0o600 {
		t.Fatalf("expected file perm 0600, got %o", finfo.Mode().Perm())
	}
}
