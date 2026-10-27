package config_test

import (
	"strings"
	"testing"
	"time"

	"live-auction/server/internal/config"
)

func setValidEnv(t *testing.T) {
	t.Helper()

	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "postgres://postgres:root@localhost:5432/live_auction?sslmode=disable")
	t.Setenv("APP_ORIGIN", "http://localhost:5173")
	t.Setenv("JWT_SECRET", strings.Repeat("a", 32))
	t.Setenv("ACCESS_TOKEN_TTL", "")
	t.Setenv("REFRESH_TOKEN_TTL", "")
	t.Setenv("REFRESH_GRACE_PERIOD", "")
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("DB_MAX_CONNS", "")
	t.Setenv("SEED_ADMIN_EMAIL", "")
	t.Setenv("SEED_ADMIN_PASSWORD", "")
	t.Setenv("SEED_BIDDER_PASSWORD", "")
}

func TestLoadConfigDefaults(t *testing.T) {
	setValidEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, 15*time.Minute)
	}
	if cfg.RefreshTokenTTL != 168*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want %v", cfg.RefreshTokenTTL, 168*time.Hour)
	}
	if cfg.RefreshGracePeriod != 10*time.Second {
		t.Errorf("RefreshGracePeriod = %v, want %v", cfg.RefreshGracePeriod, 10*time.Second)
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure = true, want false")
	}
	if cfg.DBMaxConns != 20 {
		t.Errorf("DBMaxConns = %d, want 20", cfg.DBMaxConns)
	}
}

func TestLoadConfigParsesDBMaxConns(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DB_MAX_CONNS", "7")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DBMaxConns != 7 {
		t.Errorf("DBMaxConns = %d, want 7", cfg.DBMaxConns)
	}
}

func TestLoadConfigRejectsInvalidDBMaxConns(t *testing.T) {
	for _, value := range []string{"0", "-1", "abc"} {
		t.Run(value, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("DB_MAX_CONNS", value)

			if _, err := config.Load(); err == nil {
				t.Fatalf("Load() error = nil, want error for DB_MAX_CONNS=%s", value)
			}
		})
	}
}

func TestLoadConfigParsesDurations(t *testing.T) {
	setValidEnv(t)
	t.Setenv("ACCESS_TOKEN_TTL", "1m")
	t.Setenv("REFRESH_TOKEN_TTL", "24h")
	t.Setenv("REFRESH_GRACE_PERIOD", "5s")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AccessTokenTTL != time.Minute {
		t.Errorf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, time.Minute)
	}
	if cfg.RefreshTokenTTL != 24*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want %v", cfg.RefreshTokenTTL, 24*time.Hour)
	}
	if cfg.RefreshGracePeriod != 5*time.Second {
		t.Errorf("RefreshGracePeriod = %v, want %v", cfg.RefreshGracePeriod, 5*time.Second)
	}
}

func TestLoadConfigNormalizesOrigin(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_ORIGIN", "http://localhost:5173/")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppOrigin != "http://localhost:5173" {
		t.Errorf("AppOrigin = %q, want %q", cfg.AppOrigin, "http://localhost:5173")
	}
}

func TestLoadConfigParsesSeedValues(t *testing.T) {
	setValidEnv(t)
	t.Setenv("SEED_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("SEED_ADMIN_PASSWORD", "admin-password")
	t.Setenv("SEED_BIDDER_PASSWORD", "bidder-password")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SeedAdminEmail != "admin@example.com" {
		t.Errorf("SeedAdminEmail = %q", cfg.SeedAdminEmail)
	}
	if cfg.SeedAdminPassword != "admin-password" {
		t.Errorf("SeedAdminPassword = %q", cfg.SeedAdminPassword)
	}
	if cfg.SeedBidderPassword != "bidder-password" {
		t.Errorf("SeedBidderPassword = %q", cfg.SeedBidderPassword)
	}
}

func TestLoadConfigRejectsMissingRequiredValues(t *testing.T) {
	for _, key := range []string{"DATABASE_URL", "APP_ORIGIN", "JWT_SECRET"} {
		t.Run(key, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv(key, "")

			if _, err := config.Load(); err == nil {
				t.Fatalf("Load() error = nil, want error for missing %s", key)
			}
		})
	}
}

func TestLoadConfigRejectsShortJWTSecret(t *testing.T) {
	setValidEnv(t)
	t.Setenv("JWT_SECRET", strings.Repeat("a", 31))

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want error for short JWT_SECRET")
	}
}

func TestLoadConfigRejectsInvalidDuration(t *testing.T) {
	for _, key := range []string{"ACCESS_TOKEN_TTL", "REFRESH_TOKEN_TTL", "REFRESH_GRACE_PERIOD"} {
		t.Run(key, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv(key, "not-a-duration")

			if _, err := config.Load(); err == nil {
				t.Fatalf("Load() error = nil, want error for invalid %s", key)
			}
		})
	}
}

func TestLoadConfigRejectsInvalidBool(t *testing.T) {
	setValidEnv(t)
	t.Setenv("COOKIE_SECURE", "not-a-bool")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want error for invalid COOKIE_SECURE")
	}
}

func TestLoadConfigRejectsInvalidPort(t *testing.T) {
	setValidEnv(t)
	t.Setenv("PORT", "abc")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want error for invalid PORT")
	}
}

func TestLoadConfigRejectsOriginWithPath(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_ORIGIN", "http://localhost:5173/app")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want error for APP_ORIGIN with path")
	}
}

func TestLoadConfigRejectsInsecureCookieForHTTPS(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_ORIGIN", "https://app.example.com")
	t.Setenv("COOKIE_SECURE", "false")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() error = nil, want error for insecure cookie on HTTPS")
	}
}

func TestLoadConfigAcceptsSecureCookieForHTTPS(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_ORIGIN", "https://app.example.com")
	t.Setenv("COOKIE_SECURE", "true")

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
}
