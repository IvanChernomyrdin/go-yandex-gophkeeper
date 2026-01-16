package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/api"
)

func TestClient_postJSON_SetsHeaders_AndDecodesResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/x", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected method POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ct)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer token-1" {
			t.Fatalf("expected Authorization Bearer token-1, got %q", auth)
		}

		var got map[string]any
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if got["a"] != float64(1) { // json numbers decode as float64 into map
			t.Fatalf("expected a=1, got %#v", got["a"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	var resp map[string]any
	err := c.PostJSON("/x", map[string]any{"a": 1}, &resp, "token-1")
	if err != nil {
		t.Fatalf("postJSON returned error: %v", err)
	}
	if resp["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", resp["ok"])
	}
}

func TestClient_postJSON_WithoutAuth_DoesNotSetAuthorization(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/x", func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Fatalf("expected empty Authorization, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	var resp map[string]any
	err := c.PostJSON("/x", map[string]any{"a": 1}, &resp, "")
	if err != nil {
		t.Fatalf("postJSON returned error: %v", err)
	}
}

func TestClient_postJSON_Non2xx_ReturnsBodyAsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/x", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "bad request: invalid input")
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	err := c.PostJSON("/x", map[string]any{"a": 1}, nil, "")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "bad request: invalid input") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_postJSON_respNil_DoesNotDecode(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/x", func(w http.ResponseWriter, r *http.Request) {
		// вернём не-JSON, но при resp=nil клиент не должен пытаться декодировать
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "not a json")
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	if err := c.PostJSON("/x", map[string]any{"a": 1}, nil, ""); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestClient_getJSON_SetsAuthorization_AndDecodesResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected method GET, got %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer token-1" {
			t.Fatalf("expected Authorization Bearer token-1, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"user_id": "u1"})
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	var resp map[string]any
	if err := c.GetJSON("/me", &resp, "token-1"); err != nil {
		t.Fatalf("getJSON returned error: %v", err)
	}
	if resp["user_id"] != "u1" {
		t.Fatalf("expected user_id=u1, got %#v", resp["user_id"])
	}
}

func TestClient_getJSON_WithoutAuth_DoesNotSetAuthorization(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Fatalf("expected empty Authorization, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"user_id": "u1"})
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	var resp map[string]any
	if err := c.GetJSON("/me", &resp, ""); err != nil {
		t.Fatalf("getJSON returned error: %v", err)
	}
}

func TestClient_getJSON_Non2xx_ReturnsBodyAsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "unauthorized")
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	var resp map[string]any
	err := c.GetJSON("/me", &resp, "token-1")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_postJSON_BadRequestEncoding_ReturnsError(t *testing.T) {
	// json.Encoder не умеет кодировать func
	bad := func() {}

	srv := httptest.NewTLSServer(http.NewServeMux())
	defer srv.Close()

	c := api.NewClient(srv.URL)

	var resp bytes.Buffer
	err := c.PostJSON("/x", bad, &resp, "")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
