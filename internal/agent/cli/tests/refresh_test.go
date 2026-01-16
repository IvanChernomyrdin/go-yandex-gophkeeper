package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/cli"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/config"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

func TestNewRefreshCmd_NoRefreshToken_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, "creds.json")

	app := &cli.App{
		ServerURL: "https://127.0.0.1:8080",
		CredsPath: credsPath,
		Creds:     &config.Credentials{AccessToken: "access-old", RefreshToken: ""},
	}

	cmd := cli.NewRefreshCmd(app)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("%s, got nil", serr.ErrExpectedError.Error())
	}
	if !strings.Contains(err.Error(), "no refresh_token in config") {
		t.Fatalf("%s: %v", serr.ErrUnexpectedError.Error(), err)
	}
}

func TestNewRefreshCmd_Success_UpdatesTokensAndSaves(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ct)
		}

		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.RefreshToken != "refresh-old" {
			t.Fatalf("expected refresh-old, got %q", req.RefreshToken)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  "access-new",
			"refresh_token": "refresh-new",
		})
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, "creds.json")

	app := &cli.App{
		ServerURL: srv.URL,
		CredsPath: credsPath,
		Creds:     &config.Credentials{AccessToken: "access-old", RefreshToken: "refresh-old"},
	}

	cmd := cli.NewRefreshCmd(app)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if got := out.String(); !strings.Contains(got, "refresh ok (tokens updated)") {
		t.Fatalf("unexpected output: %q", got)
	}

	loaded, err := config.Load(credsPath)
	if err != nil {
		t.Fatalf("load creds: %v", err)
	}
	if loaded.AccessToken != "access-new" {
		t.Fatalf("expected AccessToken=access-new, got %q", loaded.AccessToken)
	}
	if loaded.RefreshToken != "refresh-new" {
		t.Fatalf("expected RefreshToken=refresh-new, got %q", loaded.RefreshToken)
	}
}

func TestNewRefreshCmd_ServerReturnsError_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid refresh token"))
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, "creds.json")

	app := &cli.App{
		ServerURL: srv.URL,
		CredsPath: credsPath,
		Creds:     &config.Credentials{AccessToken: "access-old", RefreshToken: "refresh-old"},
	}

	cmd := cli.NewRefreshCmd(app)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("%s, got nil", serr.ErrExpectedError.Error())
	}
	if !strings.Contains(err.Error(), "invalid refresh token") {
		t.Fatalf("%s: %v", serr.ErrUnexpectedError.Error(), err)
	}
}
