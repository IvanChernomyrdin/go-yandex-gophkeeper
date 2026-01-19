package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/agent/api"
	"github.com/stretchr/testify/require"
)

func TestClient_Register_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req api.RegisterRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "test@example.com", req.Email)
		require.Equal(t, "StrongPass123", req.Password)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.RegisterResponse{UserID: "u1"})
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	resp, err := c.Register("test@example.com", "StrongPass123")
	require.NoError(t, err)
	require.Equal(t, "u1", resp.UserID)
}

func TestClient_Login_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req api.LoginRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "test@example.com", req.Email)
		require.Equal(t, "StrongPass123", req.Password)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.LoginResponse{
			AccessToken:  "access-1",
			RefreshToken: "refresh-1",
		})
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	resp, err := c.Login("test@example.com", "StrongPass123")
	require.NoError(t, err)
	require.Equal(t, "access-1", resp.AccessToken)
	require.Equal(t, "refresh-1", resp.RefreshToken)
}

func TestClient_Refresh_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req api.RefreshRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "refresh-1", req.RefreshToken)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.RefreshResponse{
			AccessToken:  "access-2",
			RefreshToken: "refresh-2",
		})
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	resp, err := c.Refresh("refresh-1")
	require.NoError(t, err)
	require.Equal(t, "access-2", resp.AccessToken)
	require.Equal(t, "refresh-2", resp.RefreshToken)
}

func TestClient_Me_Success_UsesBearerToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "Bearer access-1", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.MeResponse{UserID: "u1"})
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	resp, err := c.Me("access-1")
	require.NoError(t, err)
	require.Equal(t, "u1", resp.UserID)
}

func TestClient_Non2xx_ReturnsBodyAsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "invalid credentials")
	})

	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL)

	_, err := c.Login("test@example.com", "wrong")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "invalid credentials"))
}
