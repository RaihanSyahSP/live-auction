package auth

import (
	"errors"
	"fmt"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Principal struct {
	UserID uuid.UUID
	Role   domain.Role
}

type AccessTokens struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
}

type accessClaims struct {
	Role domain.Role `json:"role"`
	jwt.RegisteredClaims
}

func NewAccessTokens(secret []byte, issuer string, audience string, ttl time.Duration) *AccessTokens {
	return &AccessTokens{
		secret:   secret,
		issuer:   issuer,
		audience: audience,
		ttl:      ttl,
	}
}

func (a *AccessTokens) Issue(userID uuid.UUID, role domain.Role) (string, error) {
	now := time.Now().UTC()

	claims := accessClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    a.issuer,
			Audience:  jwt.ClaimStrings{a.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(a.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	raw, err := token.SignedString(a.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return raw, nil
}

func (a *AccessTokens) Parse(raw string) (Principal, error) {
	claims := &accessClaims{}

	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, exception.ErrTokenInvalid
			}
			return a.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(a.issuer),
		jwt.WithAudience(a.audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return Principal{}, exception.ErrTokenExpired
		}
		return Principal{}, exception.ErrTokenInvalid
	}

	if token == nil || !token.Valid {
		return Principal{}, exception.ErrTokenInvalid
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return Principal{}, exception.ErrTokenInvalid
	}

	if claims.Role != domain.RoleUser && claims.Role != domain.RoleAdmin {
		return Principal{}, exception.ErrTokenInvalid
	}

	return Principal{
		UserID: userID,
		Role:   claims.Role,
	}, nil
}
