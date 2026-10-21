package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"
)

func TestAccessTokensIssueAndParse(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	tokens := auth.NewAccessTokens(secret, "live-auction", "live-auction-web", 15*time.Minute)

	for _, role := range []domain.Role{domain.RoleUser, domain.RoleAdmin} {
		t.Run(string(role), func(t *testing.T) {
			userID := uuid.New()

			raw, err := tokens.Issue(userID, role)
			if err != nil {
				t.Fatalf("Issue() error = %v", err)
			}

			principal, err := tokens.Parse(raw)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if principal.UserID != userID {
				t.Errorf("UserID = %v, want %v", principal.UserID, userID)
			}
			if principal.Role != role {
				t.Errorf("Role = %q, want %q", principal.Role, role)
			}
		})
	}
}

func TestAccessTokensRejectsWrongSecret(t *testing.T) {
	issuer := "live-auction"
	audience := "live-auction-web"
	issuerTokens := auth.NewAccessTokens([]byte("01234567890123456789012345678901"), issuer, audience, time.Minute)
	parser := auth.NewAccessTokens([]byte("abcdefghijklmnopqrstuvwxyz123456"), issuer, audience, time.Minute)

	raw, err := issuerTokens.Issue(uuid.New(), domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	_, err = parser.Parse(raw)
	if !errors.Is(err, exception.ErrTokenInvalid) {
		t.Fatalf("Parse() error = %v, want %v", err, exception.ErrTokenInvalid)
	}
}

func TestAccessTokensRejectsExpiredToken(t *testing.T) {
	tokens := auth.NewAccessTokens(
		[]byte("01234567890123456789012345678901"),
		"live-auction",
		"live-auction-web",
		-time.Minute,
	)

	raw, err := tokens.Issue(uuid.New(), domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	_, err = tokens.Parse(raw)
	if !errors.Is(err, exception.ErrTokenExpired) {
		t.Fatalf("Parse() error = %v, want %v", err, exception.ErrTokenExpired)
	}
}

func TestAccessTokensRejectsMalformedToken(t *testing.T) {
	tokens := auth.NewAccessTokens(
		[]byte("01234567890123456789012345678901"),
		"live-auction",
		"live-auction-web",
		time.Minute,
	)

	_, err := tokens.Parse("not-a-jwt")
	if !errors.Is(err, exception.ErrTokenInvalid) {
		t.Fatalf("Parse() error = %v, want %v", err, exception.ErrTokenInvalid)
	}
}

func TestAccessTokensRejectsWrongIssuer(t *testing.T) {
	issuerTokens := auth.NewAccessTokens(
		[]byte("01234567890123456789012345678901"),
		"live-auction",
		"live-auction-web",
		time.Minute,
	)
	parser := auth.NewAccessTokens(
		[]byte("01234567890123456789012345678901"),
		"other-issuer",
		"live-auction-web",
		time.Minute,
	)

	raw, err := issuerTokens.Issue(uuid.New(), domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	_, err = parser.Parse(raw)
	if !errors.Is(err, exception.ErrTokenInvalid) {
		t.Fatalf("Parse() error = %v, want %v", err, exception.ErrTokenInvalid)
	}
}

func TestAccessTokensRejectsWrongAudience(t *testing.T) {
	issuerTokens := auth.NewAccessTokens(
		[]byte("01234567890123456789012345678901"),
		"live-auction",
		"live-auction-web",
		time.Minute,
	)
	parser := auth.NewAccessTokens(
		[]byte("01234567890123456789012345678901"),
		"live-auction",
		"other-audience",
		time.Minute,
	)

	raw, err := issuerTokens.Issue(uuid.New(), domain.RoleUser)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	_, err = parser.Parse(raw)
	if !errors.Is(err, exception.ErrTokenInvalid) {
		t.Fatalf("Parse() error = %v, want %v", err, exception.ErrTokenInvalid)
	}
}

func TestAccessTokensRejectsInvalidRole(t *testing.T) {
	tokens := auth.NewAccessTokens(
		[]byte("01234567890123456789012345678901"),
		"live-auction",
		"live-auction-web",
		time.Minute,
	)

	raw, err := tokens.Issue(uuid.New(), domain.Role("bidder"))
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	_, err = tokens.Parse(raw)
	if !errors.Is(err, exception.ErrTokenInvalid) {
		t.Fatalf("Parse() error = %v, want %v", err, exception.ErrTokenInvalid)
	}
}
