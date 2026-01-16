package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/cli"
	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/config"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

func TestNewRegisterCmd_Success_PrintsMessage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ct)
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
		json.NewEncoder(w).Encode(map[string]string{"user_id": "u1"})
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	app := &cli.App{
		ServerURL: srv.URL,
		// для register эти поля не используются, но App должен быть валидным
		Creds: &config.Credentials{},
	}

	cmd := cli.NewRegisterCmd(app)

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

	if got := out.String(); got != "registration successful" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestNewRegisterCmd_MissingRequiredFlags_ReturnsError(t *testing.T) {
	app := &cli.App{ServerURL: "https://127.0.0.1:8080", Creds: &config.Credentials{}}

	cmd := cli.NewRegisterCmd(app)

	// не передаём --password
	cmd.SetArgs([]string{"--email", "test@example.com"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("%s, got nil", serr.ErrExpectedError.Error())
	}

	// Cobra обычно пишет "required flag(s) \"password\" not set"
	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("%s: %v", serr.ErrUnexpectedError.Error(), err)
	}
}

func TestNewRegisterCmd_ServerReturnsError_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(serr.ErrAlreadyExists.Error()))
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	app := &cli.App{
		ServerURL: srv.URL,
		Creds:     &config.Credentials{},
	}

	cmd := cli.NewRegisterCmd(app)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	cmd.SetArgs([]string{
		"--email", "test@example.com",
		"--password", "StrongPass123",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("%s, got nil", serr.ErrExpectedError.Error())
	}
	if !strings.Contains(err.Error(), serr.ErrAlreadyExists.Error()) {
		t.Fatalf("%s: %v", serr.ErrUnexpectedError.Error(), err)
	}
}
