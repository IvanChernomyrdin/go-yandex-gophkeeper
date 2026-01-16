// Package config отвечает за:
// - чтение server.yaml
// - подстановку переменных окружения вида ${JWT_SIGNING_KEY}
// - проставление дефолтов
// - валидацию (чтобы сервер не стартовал с дырявыми настройками)
package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/stretchr/testify/assert/yaml"
)

// Config — корневая структура всего конфига сервера.
type Config struct {
	Env           string              `yaml:"env"` // dev|stage|prod
	Server        ServerConfig        `yaml:"server"`
	TLS           TLSConfig           `yaml:"tls"`
	DB            DBConfig            `yaml:"db"`
	Migrations    MigrationsConfig    `yaml:"migrations"`
	Auth          AuthConfig          `yaml:"auth"`
	Password      PasswordConfig      `yaml:"password"`
	Secrets       SecretsConfig       `yaml:"secrets"`
	Concurrency   ConcurrencyConfig   `yaml:"concurrency"`
	Security      SecurityConfig      `yaml:"security"`
	Log           LogConfig           `yaml:"log"`
	Observability ObservabilityConfig `yaml:"observability"`
}

// ServerConfig — настройки HTTP-сервера.
type ServerConfig struct {
	Host              string        `yaml:"host"`
	Port              int           `yaml:"port"`
	TrustProxy        bool          `yaml:"trust_proxy"` // доверять ли заголовкам X-Forwarded-*
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"` // время на graceful shutdown
	MaxHeaderBytes    int           `yaml:"max_header_bytes"` // лимит размера заголовков
	MaxBodyBytes      int64         `yaml:"max_body_bytes"`   // лимит размера тела запроса
}

// TLSConfig — настройки HTTPS.
type TLSConfig struct {
	Enabled    bool   `yaml:"enabled"`
	CertFile   string `yaml:"cert_file"`
	KeyFile    string `yaml:"key_file"`
	MinVersion string `yaml:"min_version"` // "1.2"|"1.3" (1.0/1.1 запрещаем т.к. устарели)
	H2         bool   `yaml:"h2"`
}

// DBConfig — настройки подключения к базе данных.
type DBConfig struct {
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	QueryTimeout    time.Duration `yaml:"query_timeout"` // таймаут на запросы к БД
}

// MigrationsConfig — настройки миграций БД.
type MigrationsConfig struct {
	Enabled     bool          `yaml:"enabled"`
	Path        string        `yaml:"path"`
	LockTimeout time.Duration `yaml:"lock_timeout"` // сколько ждать advisory lock на миграции
}

// AuthConfig — настройки аутентификации/авторизации.
type AuthConfig struct {
	Issuer     string         `yaml:"issuer"`
	Audience   string         `yaml:"audience"`
	AccessTTL  time.Duration  `yaml:"access_ttl"`
	RefreshTTL time.Duration  `yaml:"refresh_ttl"`
	JWT        JWTConfig      `yaml:"jwt"`
	Sessions   SessionsConfig `yaml:"sessions"`
}

// JWTConfig — как подписываем JWT.
type JWTConfig struct {
	Algorithm  string `yaml:"algorithm"`   // сейчас поддерживаем только HS256
	SigningKey string `yaml:"signing_key"` // может содержать ${JWT_SIGNING_KEY}
}

// SessionsConfig — настройки хранения refresh-сессий (на сервере).
type SessionsConfig struct {
	Store              string `yaml:"store"` // db (позже можно redis)
	RotateRefresh      bool   `yaml:"rotate_refresh"`
	ReuseDetection     bool   `yaml:"reuse_detection"`
	MaxSessionsPerUser int    `yaml:"max_sessions_per_user"`
}

// PasswordConfig — настройки хэширования паролей пользователей.
type PasswordConfig struct {
	Hasher string       `yaml:"hasher"` // argon2id|bcrypt
	Argon2 Argon2Config `yaml:"argon2"`
	Bcrypt BcryptConfig `yaml:"bcrypt"`
}

// Argon2Config — параметры argon2id.
type Argon2Config struct {
	Time      uint32 `yaml:"time"`
	MemoryKiB uint32 `yaml:"memory_kib"`
	Threads   uint8  `yaml:"threads"`
	KeyLen    uint32 `yaml:"key_len"`
	SaltLen   uint32 `yaml:"salt_len"`
}

// BcryptConfig — параметры bcrypt.
type BcryptConfig struct {
	Cost int `yaml:"cost"`
}

// SecretsConfig — ограничения и политика хранения секретов.
type SecretsConfig struct {
	StoreCiphertext bool     `yaml:"store_ciphertext"` // сервер хранит ciphertext (E2E на клиенте)
	MaxPayloadBytes int64    `yaml:"max_payload_bytes"`
	MaxMetaBytes    int64    `yaml:"max_meta_bytes"`
	AllowedTypes    []string `yaml:"allowed_types"`
}

// ConcurrencyConfig — политика конфликтов при обновлении данных.
type ConcurrencyConfig struct {
	Strategy       string `yaml:"strategy"`        // optimistic_lock|last_write_wins
	ConflictPolicy string `yaml:"conflict_policy"` // reject|server_wins|client_wins
}

// SecurityConfig — ограничения/защита.
type SecurityConfig struct {
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// RateLimitConfig — простой rate limit (например по IP).
type RateLimitConfig struct {
	Enabled bool    `yaml:"enabled"`
	RPS     float64 `yaml:"rps"`
	Burst   int     `yaml:"burst"`
	Key     string  `yaml:"key"` // ip|user
}

// LogConfig — настройки логирования (zap).
type LogConfig struct {
	Logger      string            `yaml:"logger"` // zap
	Level       string            `yaml:"level"`  // debug|info|warn|error
	Format      string            `yaml:"format"` // json|console
	Development bool              `yaml:"development"`
	Sampling    LogSamplingConfig `yaml:"sampling"`
	Redact      LogRedactConfig   `yaml:"redact"`
}

type LogSamplingConfig struct {
	Enabled    bool `yaml:"enabled"`
	Initial    int  `yaml:"initial"`
	Thereafter int  `yaml:"thereafter"`
}

type LogRedactConfig struct {
	Enabled bool     `yaml:"enabled"`
	Fields  []string `yaml:"fields"` // какие поля затираем в логах
}

// ObservabilityConfig — метрики/pprof.
type ObservabilityConfig struct {
	Metrics MetricsConfig `yaml:"metrics"`
	Pprof   PprofConfig   `yaml:"pprof"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type PprofConfig struct {
	Enabled    bool   `yaml:"enabled"`
	PathPrefix string `yaml:"path_prefix"`
}

// Load читает YAML, подставляет переменные окружения вида ${VAR},
// затем парсит в структуру, проставляет дефолты и валидирует.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать конфиг: %w", err)
	}

	// Подставляем переменные окружения в текст YAML:
	// signing_key: "${JWT_SIGNING_KEY}" -> signing_key: "реальное_значение"
	expanded := ExpandEnvStrict(string(raw))
	raw = []byte(expanded)

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("не удалось распарсить yaml: %w", err)
	}

	ApplyDefaults(&cfg)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ExpandEnvStrict заменяет ${VAR} на значение из окружения.
// Если переменная не задана — оставляем ${VAR} как есть,
// а потом Validate() упадёт с понятной ошибкой.
func ExpandEnvStrict(s string) string {
	re := regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`)
	return re.ReplaceAllStringFunc(s, func(m string) string {
		sub := re.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		if val, ok := os.LookupEnv(sub[1]); ok {
			return val
		}
		return m
	})
}

// ApplyDefaults — дефолтные значения, если в yaml поле не задано.
func ApplyDefaults(cfg *Config) {
	if cfg.Env == "" {
		cfg.Env = "dev"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Auth.JWT.Algorithm == "" {
		cfg.Auth.JWT.Algorithm = "HS256"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "json"
	}
	if cfg.Security.RateLimit.Key == "" {
		cfg.Security.RateLimit.Key = "ip"
	}
}

// Validate проверяет, что конфиг заполнен корректно и безопасно.
// Если что-то не так — возвращаем ошибку и сервер НЕ стартует.
func (c *Config) Validate() error {
	// Базовая проверка сервера
	if c.Server.Host == "" {
		return errors.New("server.host обязателен")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port некорректен: %d", c.Server.Port)
	}

	// TLS/HTTPS
	if c.TLS.Enabled {
		if c.TLS.CertFile == "" || c.TLS.KeyFile == "" {
			return errors.New("tls.cert_file и tls.key_file обязательны при tls.enabled=true")
		}
		if c.TLS.MinVersion == "" {
			c.TLS.MinVersion = "1.2"
		}
		// TLS 1.0/1.1 считаются небезопасными — запрещаем
		if c.TLS.MinVersion == "1.0" || c.TLS.MinVersion == "1.1" {
			return fmt.Errorf("tls.min_version=%s небезопасен; используй 1.2 или 1.3", c.TLS.MinVersion)
		}
	}

	// База данных
	if c.DB.DSN == "" {
		return errors.New("db.dsn обязателен")
	}

	// JWT
	alg := strings.ToUpper(strings.TrimSpace(c.Auth.JWT.Algorithm))
	if alg != "HS256" {
		return fmt.Errorf("auth.jwt.algorithm должен быть HS256 (сейчас %q)", c.Auth.JWT.Algorithm)
	}

	key := strings.TrimSpace(c.Auth.JWT.SigningKey)
	if key == "" {
		return errors.New("auth.jwt.signing_key обязателен (через ${JWT_SIGNING_KEY} или прямо строкой)")
	}
	// Если ${JWT_SIGNING_KEY} не подставился — значит переменная окружения не задана
	if strings.Contains(key, "${") && strings.Contains(key, "}") {
		return fmt.Errorf("auth.jwt.signing_key содержит неподставленную переменную: %q (нужно задать JWT_SIGNING_KEY)", key)
	}
	// Для HS256 ключ должен быть длинным и случайным
	if len(key) < 32 {
		return fmt.Errorf("auth.jwt.signing_key слишком короткий (%d символов); нужно >= 32", len(key))
	}

	// Rate limit
	if c.Security.RateLimit.Enabled {
		if c.Security.RateLimit.RPS <= 0 {
			return errors.New("security.rate_limit.rps должен быть > 0 при включённом rate_limit")
		}
		if c.Security.RateLimit.Burst <= 0 {
			return errors.New("security.rate_limit.burst должен быть > 0 при включённом rate_limit")
		}
		if c.Security.RateLimit.Key != "ip" && c.Security.RateLimit.Key != "user" {
			return fmt.Errorf("security.rate_limit.key должен быть ip|user (сейчас %q)", c.Security.RateLimit.Key)
		}
	}

	// Хэширование паролей
	switch strings.ToLower(c.Password.Hasher) {
	case "argon2id":
		if c.Password.Argon2.Time == 0 || c.Password.Argon2.MemoryKiB == 0 || c.Password.Argon2.Threads == 0 {
			return errors.New("password.argon2 должен быть настроен для argon2id")
		}
	case "bcrypt":
		if c.Password.Bcrypt.Cost == 0 {
			return errors.New("password.bcrypt.cost должен быть задан для bcrypt")
		}
	default:
		return fmt.Errorf("password.hasher должен быть argon2id|bcrypt (сейчас %q)", c.Password.Hasher)
	}

	// Политика конфликтов
	if c.Concurrency.Strategy == "" {
		return errors.New("concurrency.strategy обязателен")
	}
	if c.Concurrency.ConflictPolicy == "" {
		return errors.New("concurrency.conflict_policy обязателен")
	}

	// Сессии refresh
	if c.Auth.Sessions.Store == "" {
		return errors.New("auth.sessions.store обязателен")
	}
	if c.Auth.Sessions.MaxSessionsPerUser <= 0 {
		return errors.New("auth.sessions.max_sessions_per_user должен быть > 0")
	}

	return nil
}

// ApplyEnvOverrides — опциональная штука: даёт возможность переопределять
// некоторые настройки через переменные окружения без ${...} в yaml.
// Например SERVER_PORT=9090 переопределит server.port.
func (c *Config) ApplyEnvOverrides() {
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			c.Server.Port = p
		}
	}
}
