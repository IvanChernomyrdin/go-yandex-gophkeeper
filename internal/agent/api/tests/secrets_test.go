package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/api"
	sharedModels "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/models"
)

func TestClient_Sync_CallsGETSecrets_AndDecodes(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/secrets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer token-1" {
			t.Fatalf("expected Authorization Bearer token-1, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"secrets":[{"id":"s1","type":"text","title":"t","payload":"p","version":1,"updated_at":"2026-01-19T12:00:00Z","created_at":"2026-01-19T12:00:00Z"}]}`)
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	resp, err := c.Sync("token-1")
	if err != nil {
		t.Fatalf("Sync error: %v", err)
	}
	if len(resp.Secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(resp.Secrets))
	}
	if resp.Secrets[0].ID != "s1" {
		t.Fatalf("expected id s1, got %q", resp.Secrets[0].ID)
	}
}

func TestClient_CreateSecret_POSTSecrets_AndDecodes(t *testing.T) {
	var got map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/secrets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer token-1" {
			t.Fatalf("expected Authorization Bearer token-1, got %q", auth)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if got["type"] != "text" || got["title"] != "T" || got["payload"] != "CIPH" {
			t.Fatalf("unexpected request: %#v", got)
		}

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"s1","version":1,"updated_at":"2026-01-19T12:00:00Z"}`)
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	resp, err := c.CreateSecret("token-1", sharedModels.CreateSecretRequest{
		Type:    "text",
		Title:   "T",
		Payload: "CIPH",
		Meta:    nil,
	})
	if err != nil {
		t.Fatalf("CreateSecret error: %v", err)
	}
	if resp.ID != "s1" {
		t.Fatalf("expected id s1, got %q", resp.ID)
	}
	if resp.Version != 1 {
		t.Fatalf("expected version 1, got %d", resp.Version)
	}
}

func TestClient_UpdateSecret_PUTSecretsID_204NoContent_IsOK(t *testing.T) {
	var got map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/secrets/s1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer token-1" {
			t.Fatalf("expected Authorization Bearer token-1, got %q", auth)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if got["version"] != float64(7) {
			t.Fatalf("expected version=7 in request, got %#v", got["version"])
		}

		// update endpoint returns 204 in твоём сервере
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	resp, err := c.UpdateSecret("token-1", "s1", sharedModels.UpdateSecretRequest{
		Title:   ptr("NEW"),
		Version: 7,
	})
	if err != nil {
		t.Fatalf("UpdateSecret error: %v", err)
	}
	// при 204 resp будет zero-value — это ок
	if resp.Secret.ID != "" {
		t.Fatalf("expected zero-value response on 204, got id=%q", resp.Secret.ID)
	}
}

func TestClient_DeleteSecret_FormatsVersionQuery(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/secrets/s1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.RawQuery != "version=3" {
			t.Fatalf("expected query version=3, got %q", r.URL.RawQuery)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer token-1" {
			t.Fatalf("expected Authorization Bearer token-1, got %q", auth)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	if err := c.DeleteSecret("token-1", "s1", 3); err != nil {
		t.Fatalf("DeleteSecret error: %v", err)
	}
}

func TestClient_UpdateSecret_Non2xx_ReturnsErrorBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/secrets/s1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		io.WriteString(w, `{"error":"conflict"}`)
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	_, err := c.UpdateSecret("token-1", "s1", sharedModels.UpdateSecretRequest{Version: 1})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func ptr(s string) *string { return &s }
