package app_test

import (
	"context"
	"testing"

	"live-auction/server/internal/app"
	"live-auction/server/internal/config"
)

func TestNewDBRejectsInvalidURL(t *testing.T) {
	_, err := app.NewDB(context.Background(), config.Config{DatabaseURL: "://not-a-url"})
	if err == nil {
		t.Fatal("NewDB() error = nil, want error for invalid database URL")
	}
}
