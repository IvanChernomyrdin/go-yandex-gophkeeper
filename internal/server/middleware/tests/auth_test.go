package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Вспомогательная функция для JWT
func makeToken(t *testing.T, key, sub, iss, aud string, exp time.Time) string {
	t.Helper()

	claims := jwt.RegisteredClaims{
		Subject:   sub,
		Issuer:    iss,
		Audience:  []string{aud},
		ExpiresAt: jwt.NewNumericDate(exp),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(key))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

// Успех
func TestAuthMiddleware_OK(t *testing.T) {
	key := "secret"
	v := middleware.NewJWTVerifier(key, "issuer", "aud")

	userID := uuid.New()

	token := makeToken(
		t,
		key,
		userID.String(),
		"issuer",
		"aud",
		time.Now().Add(time.Minute),
	)

	called := false
	handler := v.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		uid, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("user id not found in context")
		}

		if uid != userID {
			t.Fatalf("unexpected user id: %v", uid)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

// Нет токена
func TestAuthMiddleware_MissingToken(t *testing.T) {
	v := middleware.NewJWTVerifier("key", "", "")

	handler := v.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

// Токен истёк
func TestAuthMiddleware_Expired(t *testing.T) {
	key := "secret"
	v := middleware.NewJWTVerifier(key, "", "")

	token := makeToken(
		t,
		key,
		"user",
		"",
		"",
		time.Now().Add(-time.Minute),
	)

	handler := v.AuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

// Проверка форматов принимаемого токена
func TestExtractBearer(t *testing.T) {
	tests := []struct {
		hdr  string
		want string
	}{
		{"Bearer token", "token"},
		{"bearer token", "token"},
		{"Bearer    token", "token"},
		{"Token token", ""},
		{"", ""},
	}

	for _, tt := range tests {
		if got := middleware.ExtractBearer(tt.hdr); got != tt.want {
			t.Errorf("ExtractBearer(%q) = %q, want %q", tt.hdr, got, tt.want)
		}
	}
}
