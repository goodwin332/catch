package config

import "testing"

func TestConfigRejectsDevLoginInProduction(t *testing.T) {
	cfg := Config{
		AppName: "catch-api",
		Env:     EnvProduction,
		HTTP: HTTPConfig{
			Addr: ":8080",
		},
		Database: DatabaseConfig{
			URL:      "postgres://catch:catch@localhost:5432/catch?sslmode=disable",
			MaxConns: 1,
		},
		Auth: AuthConfig{
			SessionCookieName: "catch_session",
			CSRFCookieName:    "catch_csrf",
			CSRFHeaderName:    "X-CSRF-Token",
			Secret:            "production-secret",
			SessionTTL:        1,
			EmailCodeTTL:      1,
			EmailCodeLength:   6,
			DevLoginEmail:     "dev@catch.local",
			DevLoginEnabled:   true,
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production config with dev login enabled to be rejected")
	}
}
