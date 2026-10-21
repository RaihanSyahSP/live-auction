package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               string
	DatabaseURL        string
	AppOrigin          string
	JWTSecret          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	RefreshGracePeriod time.Duration
	CookieSecure       bool
	SeedAdminEmail     string
	SeedAdminPassword  string
	SeedBidderPassword string
}

func Load() (Config, error) {
	cfg := Config{
		Port:               envOrDefault("PORT", "8080"),
		DatabaseURL:        strings.TrimSpace(os.Getenv("DATABASE_URL")),
		AppOrigin:          strings.TrimRight(strings.TrimSpace(os.Getenv("APP_ORIGIN")), "/"),
		JWTSecret:          strings.TrimSpace(os.Getenv("JWT_SECRET")),
		SeedAdminEmail:     strings.TrimSpace(os.Getenv("SEED_ADMIN_EMAIL")),
		SeedAdminPassword:  strings.TrimSpace(os.Getenv("SEED_ADMIN_PASSWORD")),
		SeedBidderPassword: strings.TrimSpace(os.Getenv("SEED_BIDDER_PASSWORD")),
	}

	var err error
	if cfg.AccessTokenTTL, err = durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.RefreshTokenTTL, err = durationEnv("REFRESH_TOKEN_TTL", 168*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.RefreshGracePeriod, err = durationEnv("REFRESH_GRACE_PERIOD", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.CookieSecure, err = boolEnv("COOKIE_SECURE", false); err != nil {
		return Config{}, err
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.AppOrigin == "" {
		return fmt.Errorf("APP_ORIGIN is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 bytes")
	}

	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("PORT must be a number between 1 and 65535")
	}

	origin, err := url.Parse(c.AppOrigin)
	if err != nil || origin.Scheme == "" || origin.Host == "" {
		return fmt.Errorf("APP_ORIGIN must be an absolute URL")
	}
	if origin.Scheme != "http" && origin.Scheme != "https" {
		return fmt.Errorf("APP_ORIGIN must use http or https")
	}
	if origin.Path != "" && origin.Path != "/" {
		return fmt.Errorf("APP_ORIGIN must not contain a path")
	}
	if origin.RawQuery != "" || origin.Fragment != "" {
		return fmt.Errorf("APP_ORIGIN must not contain query or fragment")
	}

	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be greater than zero")
	}
	if c.RefreshTokenTTL <= 0 {
		return fmt.Errorf("REFRESH_TOKEN_TTL must be greater than zero")
	}
	if c.AccessTokenTTL >= c.RefreshTokenTTL {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be less than REFRESH_TOKEN_TTL")
	}
	if c.RefreshGracePeriod < 0 {
		return fmt.Errorf("REFRESH_GRACE_PERIOD must not be negative")
	}
	if c.RefreshGracePeriod >= c.RefreshTokenTTL {
		return fmt.Errorf("REFRESH_GRACE_PERIOD must be less than REFRESH_TOKEN_TTL")
	}

	if origin.Scheme == "https" && !c.CookieSecure {
		return fmt.Errorf("COOKIE_SECURE must be true when APP_ORIGIN uses https")
	}

	return nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return duration, nil
}

func boolEnv(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}
