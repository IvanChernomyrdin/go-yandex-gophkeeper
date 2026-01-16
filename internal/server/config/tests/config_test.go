package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/server/config"
	serr "github.com/IvanChernomyrdin/go-yandex-gophkeeper/internal/shared/errors"
)

func TestExpandEnvStrict_ReplacesExistingEnv(t *testing.T) {
	t.Setenv("JWT_SIGNING_KEY", "supersecretkeysupersecretkey123456")

	in := `signing_key: "${JWT_SIGNING_KEY}"`
	out := config.ExpandEnvStrict(in)

	if out == in {
		t.Fatalf("expected env to be expanded, got unchanged string: %q", out)
	}
	if out != `signing_key: "supersecretkeysupersecretkey123456"` &&
		out != `signing_key: supersecretkeysupersecretkey123456` {
		// YAML допускает без кавычек, проверим просто наличие значения
		if wantSub := "supersecretkeysupersecretkey123456"; !contains(out, wantSub) {
			t.Fatalf("expected output to contain %q, got %q", wantSub, out)
		}
	}
}

func TestExpandEnvStrict_LeavesUnknownEnvAsIs(t *testing.T) {
	in := `signing_key: "${MISSING_ENV}"`
	out := config.ExpandEnvStrict(in)

	if out != in {
		t.Fatalf("expected unknown env placeholder to remain unchanged, got %q", out)
	}
}

func TestApplyDefaults_SetsExpectedDefaults(t *testing.T) {
	cfg := &config.Config{}
	config.ApplyDefaults(cfg)

	if cfg.Env != "dev" {
		t.Fatalf("expected Env=dev, got %q", cfg.Env)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("expected Server.Port=8080, got %d", cfg.Server.Port)
	}
	if cfg.Auth.JWT.Algorithm != "HS256" {
		t.Fatalf("expected Auth.JWT.Algorithm=HS256, got %q", cfg.Auth.JWT.Algorithm)
	}
	if cfg.Log.Level != "info" {
		t.Fatalf("expected Log.Level=info, got %q", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Fatalf("expected Log.Format=json, got %q", cfg.Log.Format)
	}
	if cfg.Security.RateLimit.Key != "ip" {
		t.Fatalf("expected Security.RateLimit.Key=ip, got %q", cfg.Security.RateLimit.Key)
	}
}

func TestValidate_ServerHostRequired(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Server.Host = ""

	if err := cfg.Validate(); err == nil {
		t.Fatalf("%s, got nil", serr.ErrExpectedError.Error())
	}
}

func TestValidate_TLSRequiresCertAndKey(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.TLS.Enabled = true
	cfg.TLS.CertFile = ""
	cfg.TLS.KeyFile = ""

	if err := cfg.Validate(); err == nil {
		t.Fatalf("%s, got nil", serr.ErrExpectedError.Error())
	}
}

func TestValidate_JWTSigningKeyMustBeLong(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Auth.JWT.SigningKey = "short-key"

	if err := cfg.Validate(); err == nil {
		t.Fatalf("%s, got nil", serr.ErrExpectedError.Error())
	}
}

func TestValidate_RejectsUnexpandedEnvInSigningKey(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Auth.JWT.SigningKey = "${JWT_SIGNING_KEY}"

	if err := cfg.Validate(); err == nil {
		t.Fatalf("%s, got nil", serr.ErrExpectedError.Error())
	}
}

func TestApplyEnvOverrides_ServerPort(t *testing.T) {
	cfg := minimalValidConfig()
	cfg.Server.Port = 8080

	t.Setenv("SERVER_PORT", "9090")
	cfg.ApplyEnvOverrides()

	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port=9090, got %d", cfg.Server.Port)
	}
}

func TestLoad_ExpandsEnv_AppliesDefaults_AndValidates(t *testing.T) {
	// ВАЖНО: этот тест пройдёт только если в прод-коде стоит корректный YAML-парсер
	// (обычно gopkg.in/yaml.v3).
	t.Setenv("JWT_SIGNING_KEY", "supersecretkeysupersecretkey123456")

	yml := `
env: dev
server:
  host: "127.0.0.1"
  port: 0
tls:
  enabled: false
db:
  dsn: "postgres://user:pass@localhost:5432/db?sslmode=disable"
auth:
  issuer: "gophkeeper"
  audience: "gophkeeper-cli"
  access_ttl: 1h
  refresh_ttl: 24h
  jwt:
    algorithm: ""
    signing_key: "${JWT_SIGNING_KEY}"
  sessions:
    store: "db"
    rotate_refresh: true
    reuse_detection: true
    max_sessions_per_user: 5
password:
  hasher: "bcrypt"
  bcrypt:
    cost: 10
concurrency:
  strategy: "optimistic_lock"
  conflict_policy: "reject"
security:
  rate_limit:
    enabled: false
log:
  level: ""
  format: ""
`

	tmpDir := t.TempDir()
	p := filepath.Join(tmpDir, "server.yaml")
	if err := os.WriteFile(p, []byte(yml), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := config.Load(p)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// проверяем дефолты
	if cfg.Server.Port != 8080 {
		t.Fatalf("expected default port=8080, got %d", cfg.Server.Port)
	}
	if cfg.Auth.JWT.Algorithm != "HS256" {
		t.Fatalf("expected default jwt algorithm HS256, got %q", cfg.Auth.JWT.Algorithm)
	}
	if cfg.Log.Level != "info" {
		t.Fatalf("expected default log level info, got %q", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Fatalf("expected default log format json, got %q", cfg.Log.Format)
	}

	// проверяем, что env подставился (не остался ${...})
	if contains(cfg.Auth.JWT.SigningKey, "${") {
		t.Fatalf("expected signing key to be expanded, got %q", cfg.Auth.JWT.SigningKey)
	}
}

// --- helpers ---

func minimalValidConfig() *config.Config {
	return &config.Config{
		Env: "dev",
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
		TLS: config.TLSConfig{
			Enabled: false,
		},
		DB: config.DBConfig{
			DSN: "postgres://example",
		},
		Auth: config.AuthConfig{
			JWT: config.JWTConfig{
				Algorithm:  "HS256",
				SigningKey: "supersecretkeysupersecretkey123456",
			},
			Sessions: config.SessionsConfig{
				Store:              "db",
				MaxSessionsPerUser: 5,
			},
		},
		Password: config.PasswordConfig{
			Hasher: "bcrypt",
			Bcrypt: config.BcryptConfig{Cost: 10},
		},
		Concurrency: config.ConcurrencyConfig{
			Strategy:       "optimistic_lock",
			ConflictPolicy: "reject",
		},
		Security: config.SecurityConfig{
			RateLimit: config.RateLimitConfig{
				Enabled: false,
				Key:     "ip",
			},
		},
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	// маленький локальный index, чтобы не тянуть strings в каждый тест (можно и strings.Contains).
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
