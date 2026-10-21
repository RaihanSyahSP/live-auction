package auth_test

import (
	"bytes"
	"testing"

	"live-auction/server/internal/auth"
)

func TestGenerateRefreshToken(t *testing.T) {
	raw, hash, err := auth.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	if len(raw) != 43 {
		t.Errorf("token length = %d, want 43", len(raw))
	}
	if len(hash) != 32 {
		t.Errorf("hash length = %d, want 32", len(hash))
	}
	if !bytes.Equal(hash, auth.HashRefreshToken(raw)) {
		t.Error("returned hash does not match HashRefreshToken")
	}

	raw2, _, err := auth.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("second GenerateRefreshToken() error = %v", err)
	}
	if raw == raw2 {
		t.Error("two generated tokens are equal")
	}
}

func TestHashRefreshTokenDeterministic(t *testing.T) {
	raw := "same-token"

	first := auth.HashRefreshToken(raw)
	second := auth.HashRefreshToken(raw)

	if !bytes.Equal(first, second) {
		t.Error("HashRefreshToken is not deterministic")
	}
	if len(first) != 32 {
		t.Errorf("hash length = %d, want 32", len(first))
	}
}
