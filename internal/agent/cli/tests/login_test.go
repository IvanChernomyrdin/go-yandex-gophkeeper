package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/cli"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/config"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

func TestNewLoginCmd_Success_SavesTokensAndPrintsMessage(t *testing.T) {
	// HTTPS тестовый сервер
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Email != "test@example.com" {
			t.Fatalf("expected email test@example.com, got %q", req.Email)
		}
		if req.Password != "StrongPass123" {
			t.Fatalf("expected password StrongPass123, got %q", req.Password)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  "access-1",
			"refresh_token": "refresh-1",
		})
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	// временный путь под креды
	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, "creds.json")

	app := &cli.App{
		ServerURL: srv.URL,
		CredsPath: credsPath,
		Creds:     &config.Credentials{},
	}

	cmd := cli.NewLoginCmd(app)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	cmd.SetArgs([]string{
		"--email", "test@example.com",
		"--password", "StrongPass123",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if got := out.String(); !strings.Contains(got, "login ok (tokens saved)") {
		t.Fatalf("unexpected output: %q", got)
	}

	// проверим, что токены реально сохранились в файл
	loaded, err := config.Load(credsPath)
	if err != nil {
		t.Fatalf("load creds: %v", err)
	}
	if loaded.AccessToken != "access-1" {
		t.Fatalf("expected AccessToken=access-1, got %q", loaded.AccessToken)
	}
	if loaded.RefreshToken != "refresh-1" {
		t.Fatalf("expected RefreshToken=refresh-1, got %q", loaded.RefreshToken)
	}
}

func TestNewLoginCmd_MissingRequiredFlags_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, "creds.json")

	app := &cli.App{
		ServerURL: "https://127.0.0.1:8080",
		CredsPath: credsPath,
		Creds:     &config.Credentials{},
	}

	cmd := cli.NewLoginCmd(app)
	cmd.SetArgs([]string{
		"--email", "test@example.com",
		// --password пропущен
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("%s, got nil", serr.ErrExpectedError.Error())
	}

	// Cobra обычно пишет "required flag(s) \"password\" not set"
	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("%s: %v", serr.ErrUnexpectedError.Error(), err)
	}
}

func TestNewLoginCmd_ServerReturnsError_DoesNotWriteCredsFile(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(serr.ErrInvalidCredentials.Error()))
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, "creds.json")

	app := &cli.App{
		ServerURL: srv.URL,
		CredsPath: credsPath,
		Creds:     &config.Credentials{},
	}

	cmd := cli.NewLoginCmd(app)
	cmd.SetArgs([]string{
		"--email", "test@example.com",
		"--password", "wrong",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("%s, got nil", serr.ErrExpectedError.Error())
	}
	if !strings.Contains(err.Error(), serr.ErrInvalidCredentials.Error()) {
		t.Fatalf("%s: %v", serr.ErrUnexpectedError.Error(), err)
	}

	// файл не обязан существовать (и лучше, чтобы не появлялся)
	if _, statErr := os.Stat(credsPath); statErr == nil {
		// Если всё же создался — это плохо: токены не должны сохраняться при ошибке логина
		t.Fatalf("creds file should not be created on login error")
	}
}
