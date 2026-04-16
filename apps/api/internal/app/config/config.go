package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Env string

const (
	EnvLocal       Env = "local"
	EnvDevelopment Env = "development"
	EnvTest        Env = "test"
	EnvProduction  Env = "production"
)

type Config struct {
	AppName  string
	Env      Env
	HTTP     HTTPConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Storage  StorageConfig
	Search   SearchConfig
	Mail     MailConfig
}

type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	URL           string
	MinConns      int32
	MaxConns      int32
	MigrationsDir string
}

type AuthConfig struct {
	SessionCookieName      string
	CSRFCookieName         string
	CSRFHeaderName         string
	Secret                 string
	SessionTTL             time.Duration
	EmailCodeTTL           time.Duration
	EmailCodeLength        int
	DevLoginEnabled        bool
	DevLoginEmail          string
	DevEmailCodeInResponse bool
	OAuth                  OAuthConfig
}

type OAuthConfig struct {
	StateCookieName string
	StateTTL        time.Duration
	Providers       map[string]OAuthProviderConfig
}

type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	RedirectURL  string
	Scopes       []string
}

type StorageConfig struct {
	Provider         string
	LocalPath        string
	PublicBaseURL    string
	MaxUploadBytes   int64
	S3Endpoint       string
	S3Region         string
	S3Bucket         string
	S3AccessKey      string
	S3SecretKey      string
	S3ForcePathStyle bool
}

type SearchConfig struct {
	Provider       string
	MeiliURL       string
	MeiliAPIKey    string
	MeiliIndex     string
	RequestTimeout time.Duration
}

type MailConfig struct {
	Provider     string
	From         string
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
}

func Load() (Config, error) {
	env := Env(getenv("APP_ENV", string(EnvLocal)))
	if !env.Valid() {
		return Config{}, fmt.Errorf("unknown APP_ENV %q", env)
	}

	cfg := Config{
		AppName: getenv("APP_NAME", "catch-api"),
		Env:     env,
		HTTP: HTTPConfig{
			Addr:            getenv("HTTP_ADDR", ":8080"),
			ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Database: DatabaseConfig{
			URL:           getenv("DATABASE_URL", "postgres://catch:catch@localhost:5432/catch?sslmode=disable"),
			MinConns:      getInt32("DATABASE_MIN_CONNS", 1),
			MaxConns:      getInt32("DATABASE_MAX_CONNS", 10),
			MigrationsDir: getenv("DATABASE_MIGRATIONS_DIR", "apps/api/migrations"),
		},
		Auth: AuthConfig{
			SessionCookieName:      getenv("SESSION_COOKIE_NAME", "catch_session"),
			CSRFCookieName:         getenv("CSRF_COOKIE_NAME", "catch_csrf"),
			CSRFHeaderName:         getenv("CSRF_HEADER_NAME", "X-CSRF-Token"),
			Secret:                 getenv("AUTH_SECRET", localAuthSecret(env)),
			SessionTTL:             getDuration("AUTH_SESSION_TTL", 30*24*time.Hour),
			EmailCodeTTL:           getDuration("AUTH_EMAIL_CODE_TTL", 10*time.Minute),
			EmailCodeLength:        getInt("AUTH_EMAIL_CODE_LENGTH", 6),
			DevLoginEnabled:        getBool("AUTH_DEV_LOGIN_ENABLED", env == EnvLocal),
			DevLoginEmail:          getenv("AUTH_DEV_LOGIN_EMAIL", "dev@catch.local"),
			DevEmailCodeInResponse: getBool("AUTH_DEV_EMAIL_CODE_IN_RESPONSE", env.AllowsDevTools()),
			OAuth: OAuthConfig{
				StateCookieName: getenv("OAUTH_STATE_COOKIE_NAME", "catch_oauth_state"),
				StateTTL:        getDuration("OAUTH_STATE_TTL", 10*time.Minute),
				Providers: map[string]OAuthProviderConfig{
					"google": {
						ClientID:     getenv("OAUTH_GOOGLE_CLIENT_ID", ""),
						ClientSecret: getenv("OAUTH_GOOGLE_CLIENT_SECRET", ""),
						AuthURL:      getenv("OAUTH_GOOGLE_AUTH_URL", "https://accounts.google.com/o/oauth2/v2/auth"),
						TokenURL:     getenv("OAUTH_GOOGLE_TOKEN_URL", "https://oauth2.googleapis.com/token"),
						UserInfoURL:  getenv("OAUTH_GOOGLE_USERINFO_URL", "https://openidconnect.googleapis.com/v1/userinfo"),
						RedirectURL:  getenv("OAUTH_GOOGLE_REDIRECT_URL", ""),
						Scopes:       splitCSV(getenv("OAUTH_GOOGLE_SCOPES", "openid,email,profile")),
					},
					"vk": {
						ClientID:     getenv("OAUTH_VK_CLIENT_ID", ""),
						ClientSecret: getenv("OAUTH_VK_CLIENT_SECRET", ""),
						AuthURL:      getenv("OAUTH_VK_AUTH_URL", ""),
						TokenURL:     getenv("OAUTH_VK_TOKEN_URL", ""),
						UserInfoURL:  getenv("OAUTH_VK_USERINFO_URL", ""),
						RedirectURL:  getenv("OAUTH_VK_REDIRECT_URL", ""),
						Scopes:       splitCSV(getenv("OAUTH_VK_SCOPES", "email")),
					},
					"yandex": {
						ClientID:     getenv("OAUTH_YANDEX_CLIENT_ID", ""),
						ClientSecret: getenv("OAUTH_YANDEX_CLIENT_SECRET", ""),
						AuthURL:      getenv("OAUTH_YANDEX_AUTH_URL", "https://oauth.yandex.ru/authorize"),
						TokenURL:     getenv("OAUTH_YANDEX_TOKEN_URL", "https://oauth.yandex.ru/token"),
						UserInfoURL:  getenv("OAUTH_YANDEX_USERINFO_URL", "https://login.yandex.ru/info"),
						RedirectURL:  getenv("OAUTH_YANDEX_REDIRECT_URL", ""),
						Scopes:       splitCSV(getenv("OAUTH_YANDEX_SCOPES", "login:email,login:info")),
					},
				},
			},
		},
		Storage: StorageConfig{
			Provider:         getenv("STORAGE_PROVIDER", "local"),
			LocalPath:        getenv("STORAGE_LOCAL_PATH", "var/storage"),
			PublicBaseURL:    getenv("STORAGE_PUBLIC_BASE_URL", "/api/v1/media/files"),
			MaxUploadBytes:   getInt64("STORAGE_MAX_UPLOAD_BYTES", 10*1024*1024),
			S3Endpoint:       getenv("STORAGE_S3_ENDPOINT", ""),
			S3Region:         getenv("STORAGE_S3_REGION", "ru-central1"),
			S3Bucket:         getenv("STORAGE_S3_BUCKET", ""),
			S3AccessKey:      getenv("STORAGE_S3_ACCESS_KEY", ""),
			S3SecretKey:      getenv("STORAGE_S3_SECRET_KEY", ""),
			S3ForcePathStyle: getBool("STORAGE_S3_FORCE_PATH_STYLE", true),
		},
		Search: SearchConfig{
			Provider:       getenv("SEARCH_PROVIDER", "disabled"),
			MeiliURL:       getenv("MEILI_URL", "http://localhost:7700"),
			MeiliAPIKey:    getenv("MEILI_API_KEY", ""),
			MeiliIndex:     getenv("MEILI_ARTICLES_INDEX", "catch_articles"),
			RequestTimeout: getDuration("SEARCH_REQUEST_TIMEOUT", 3*time.Second),
		},
		Mail: MailConfig{
			Provider:     getenv("MAIL_PROVIDER", "log"),
			From:         getenv("MAIL_FROM", "Catch <noreply@catch.local>"),
			SMTPHost:     getenv("SMTP_HOST", ""),
			SMTPPort:     getInt("SMTP_PORT", 587),
			SMTPUsername: getenv("SMTP_USERNAME", ""),
			SMTPPassword: getenv("SMTP_PASSWORD", ""),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.AppName == "" {
		return errors.New("APP_NAME is required")
	}
	if c.HTTP.Addr == "" {
		return errors.New("HTTP_ADDR is required")
	}
	if c.Database.URL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if c.Database.MinConns < 0 {
		return errors.New("DATABASE_MIN_CONNS must be greater than or equal to 0")
	}
	if c.Database.MaxConns <= 0 {
		return errors.New("DATABASE_MAX_CONNS must be greater than 0")
	}
	if c.Database.MaxConns < c.Database.MinConns {
		return errors.New("DATABASE_MAX_CONNS must be greater than or equal to DATABASE_MIN_CONNS")
	}
	if c.Auth.SessionCookieName == "" {
		return errors.New("SESSION_COOKIE_NAME is required")
	}
	if c.Auth.CSRFCookieName == "" {
		return errors.New("CSRF_COOKIE_NAME is required")
	}
	if c.Auth.CSRFHeaderName == "" {
		return errors.New("CSRF_HEADER_NAME is required")
	}
	if c.Auth.Secret == "" {
		return errors.New("AUTH_SECRET is required")
	}
	if c.Env == EnvProduction && c.Auth.Secret == "local-dev-secret-change-me" {
		return errors.New("AUTH_SECRET must be changed in production")
	}
	if c.Auth.SessionTTL <= 0 {
		return errors.New("AUTH_SESSION_TTL must be positive")
	}
	if c.Auth.EmailCodeTTL <= 0 {
		return errors.New("AUTH_EMAIL_CODE_TTL must be positive")
	}
	if c.Auth.EmailCodeLength < 6 || c.Auth.EmailCodeLength > 12 {
		return errors.New("AUTH_EMAIL_CODE_LENGTH must be between 6 and 12")
	}
	if c.Auth.DevLoginEmail == "" {
		return errors.New("AUTH_DEV_LOGIN_EMAIL is required")
	}
	if c.Env == EnvProduction && (c.Auth.DevLoginEnabled || c.Auth.DevEmailCodeInResponse) {
		return errors.New("dev auth features cannot be enabled in production")
	}
	if c.Auth.OAuth.StateCookieName == "" {
		return errors.New("OAUTH_STATE_COOKIE_NAME is required")
	}
	if c.Auth.OAuth.StateTTL <= 0 {
		return errors.New("OAUTH_STATE_TTL must be positive")
	}
	for provider, providerCfg := range c.Auth.OAuth.Providers {
		if !oauthProviderKnown(provider) {
			return fmt.Errorf("unknown OAuth provider %q", provider)
		}
		if !providerCfg.Enabled() {
			continue
		}
		if providerCfg.ClientSecret == "" {
			return fmt.Errorf("OAUTH_%s_CLIENT_SECRET is required when provider is enabled", strings.ToUpper(provider))
		}
		if providerCfg.AuthURL == "" {
			return fmt.Errorf("OAUTH_%s_AUTH_URL is required when provider is enabled", strings.ToUpper(provider))
		}
		if providerCfg.TokenURL == "" {
			return fmt.Errorf("OAUTH_%s_TOKEN_URL is required when provider is enabled", strings.ToUpper(provider))
		}
		if providerCfg.UserInfoURL == "" {
			return fmt.Errorf("OAUTH_%s_USERINFO_URL is required when provider is enabled", strings.ToUpper(provider))
		}
		if providerCfg.RedirectURL == "" {
			return fmt.Errorf("OAUTH_%s_REDIRECT_URL is required when provider is enabled", strings.ToUpper(provider))
		}
	}
	switch c.Storage.Provider {
	case "local":
		if c.Storage.LocalPath == "" {
			return errors.New("STORAGE_LOCAL_PATH is required when STORAGE_PROVIDER=local")
		}
	case "s3":
		if c.Storage.S3Region == "" {
			return errors.New("STORAGE_S3_REGION is required when STORAGE_PROVIDER=s3")
		}
		if c.Storage.S3Bucket == "" {
			return errors.New("STORAGE_S3_BUCKET is required when STORAGE_PROVIDER=s3")
		}
		if c.Storage.S3AccessKey == "" {
			return errors.New("STORAGE_S3_ACCESS_KEY is required when STORAGE_PROVIDER=s3")
		}
		if c.Storage.S3SecretKey == "" {
			return errors.New("STORAGE_S3_SECRET_KEY is required when STORAGE_PROVIDER=s3")
		}
	default:
		return errors.New("STORAGE_PROVIDER must be local or s3")
	}
	if c.Storage.PublicBaseURL == "" {
		return errors.New("STORAGE_PUBLIC_BASE_URL is required")
	}
	if c.Storage.MaxUploadBytes <= 0 {
		return errors.New("STORAGE_MAX_UPLOAD_BYTES must be positive")
	}
	switch c.Search.Provider {
	case "disabled", "meilisearch":
	default:
		return errors.New("SEARCH_PROVIDER must be disabled or meilisearch")
	}
	if c.Search.Provider == "meilisearch" {
		if c.Search.MeiliURL == "" {
			return errors.New("MEILI_URL is required when SEARCH_PROVIDER=meilisearch")
		}
		if c.Search.MeiliIndex == "" {
			return errors.New("MEILI_ARTICLES_INDEX is required when SEARCH_PROVIDER=meilisearch")
		}
		if c.Search.RequestTimeout <= 0 {
			return errors.New("SEARCH_REQUEST_TIMEOUT must be positive")
		}
	}
	switch c.Mail.Provider {
	case "", "disabled", "log", "smtp":
	default:
		return errors.New("MAIL_PROVIDER must be disabled, log or smtp")
	}
	if c.Mail.Provider == "smtp" {
		if c.Mail.From == "" {
			return errors.New("MAIL_FROM is required when MAIL_PROVIDER=smtp")
		}
		if c.Mail.SMTPHost == "" {
			return errors.New("SMTP_HOST is required when MAIL_PROVIDER=smtp")
		}
		if c.Mail.SMTPPort <= 0 {
			return errors.New("SMTP_PORT must be positive")
		}
	}
	return nil
}

func (e Env) Valid() bool {
	switch e {
	case EnvLocal, EnvDevelopment, EnvTest, EnvProduction:
		return true
	default:
		return false
	}
}

func (e Env) IsProduction() bool {
	return e == EnvProduction
}

func (e Env) AllowsDevTools() bool {
	return e == EnvLocal || e == EnvDevelopment || e == EnvTest
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getInt32(key string, fallback int32) int32 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(parsed)
}

func getInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getInt64(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func (c OAuthProviderConfig) Enabled() bool {
	return c.ClientID != ""
}

func oauthProviderKnown(provider string) bool {
	switch provider {
	case "google", "vk", "yandex":
		return true
	default:
		return false
	}
}

func localAuthSecret(env Env) string {
	if env.AllowsDevTools() {
		return "local-dev-secret-change-me"
	}
	return ""
}
