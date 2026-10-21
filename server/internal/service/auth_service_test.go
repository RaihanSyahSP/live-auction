package service_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/auth_request"
	"live-auction/server/internal/model/web/auth_response"
	"live-auction/server/internal/repository"
	refreshrepo "live-auction/server/internal/repository/refresh_token"
	userrepo "live-auction/server/internal/repository/user"
	"live-auction/server/internal/service"
)

const (
	testJWTSecret  = "01234567890123456789012345678901"
	testRefreshTTL = 7 * 24 * time.Hour
	testGrace      = 10 * time.Second
)

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time {
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

type singleTxManager struct {
	tx pgx.Tx
}

func (m singleTxManager) WithinTx(ctx context.Context, fn func(pgx.Tx) error) error {
	return fn(m.tx)
}

type authFixture struct {
	svc     service.AuthService
	tokens  *auth.AccessTokens
	users   userrepo.UserRepository
	refresh refreshrepo.RefreshTokenRepository
	tx      pgx.Tx
	clock   *testClock
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()

	pool := helper.OpenTestDB(t)
	tx := helper.BeginTestTx(t, pool)
	clock := &testClock{now: time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)}
	tokens := auth.NewAccessTokens([]byte(testJWTSecret), "live-auction", "live-auction-web", 15*time.Minute)
	users := userrepo.NewUserRepositoryImpl()
	refresh := refreshrepo.NewRefreshTokenRepositoryImpl()

	return &authFixture{
		svc: service.NewAuthService(
			users,
			refresh,
			singleTxManager{tx: tx},
			tokens,
			clock.Now,
			testRefreshTTL,
			testGrace,
		),
		tokens:  tokens,
		users:   users,
		refresh: refresh,
		tx:      tx,
		clock:   clock,
	}
}

func (f *authFixture) register(t *testing.T, password string) auth_response.AuthTokens {
	t.Helper()

	result, err := f.svc.Register(context.Background(), auth_request.RegisterRequest{
		Email:       uuid.NewString() + "@example.com",
		Password:    password,
		DisplayName: "Raihan",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return result
}

func (f *authFixture) findRefresh(t *testing.T, raw string) domain.RefreshToken {
	t.Helper()

	token, err := f.refresh.FindByHashForUpdate(
		context.Background(),
		f.tx,
		auth.HashRefreshToken(raw),
	)
	if err != nil {
		t.Fatalf("FindByHashForUpdate() error = %v", err)
	}
	return token
}

func TestAuthServiceRegister(t *testing.T) {
	f := newAuthFixture(t)
	email := strings.ToUpper(uuid.NewString() + "@example.com")
	password := "password123"

	result, err := f.svc.Register(context.Background(), auth_request.RegisterRequest{
		Email:       email,
		Password:    password,
		DisplayName: "Raihan",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if result.User.ID == "" {
		t.Error("User.ID is empty")
	}
	if result.User.Email != strings.ToLower(email) {
		t.Errorf("User.Email = %q, want %q", result.User.Email, strings.ToLower(email))
	}
	if result.User.Role != string(domain.RoleUser) {
		t.Errorf("User.Role = %q, want %q", result.User.Role, domain.RoleUser)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatal("Register() returned empty tokens")
	}

	userID, err := uuid.Parse(result.User.ID)
	if err != nil {
		t.Fatalf("parse user ID: %v", err)
	}

	principal, err := f.tokens.Parse(result.AccessToken)
	if err != nil {
		t.Fatalf("Parse(access token) error = %v", err)
	}
	if principal.UserID != userID {
		t.Errorf("access token UserID = %v, want %v", principal.UserID, userID)
	}

	storedUser, err := f.users.FindByID(context.Background(), f.tx, userID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedUser.PasswordHash), []byte(password)); err != nil {
		t.Error("stored password hash does not match password")
	}
	if storedUser.PasswordHash == password {
		t.Error("password was stored as plaintext")
	}

	refreshToken := f.findRefresh(t, result.RefreshToken)
	if refreshToken.UserID != userID {
		t.Errorf("refresh token UserID = %v, want %v", refreshToken.UserID, userID)
	}
}

func TestAuthServiceRegisterValidation(t *testing.T) {
	f := newAuthFixture(t)

	tests := []struct {
		name string
		req  auth_request.RegisterRequest
	}{
		{
			name: "invalid email",
			req:  auth_request.RegisterRequest{Email: "not-an-email", Password: "password123", DisplayName: "Raihan"},
		},
		{
			name: "short password",
			req:  auth_request.RegisterRequest{Email: uuid.NewString() + "@example.com", Password: "short", DisplayName: "Raihan"},
		},
		{
			name: "empty display name",
			req:  auth_request.RegisterRequest{Email: uuid.NewString() + "@example.com", Password: "password123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := f.svc.Register(context.Background(), tt.req); !errors.Is(err, exception.ErrValidation) {
				t.Fatalf("Register() error = %v, want %v", err, exception.ErrValidation)
			}
		})
	}
}

func TestAuthServiceRegisterRejectsDuplicateEmail(t *testing.T) {
	f := newAuthFixture(t)
	email := uuid.NewString() + "@example.com"

	if _, err := f.svc.Register(context.Background(), auth_request.RegisterRequest{
		Email:       email,
		Password:    "password123",
		DisplayName: "First",
	}); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	_, err := f.svc.Register(context.Background(), auth_request.RegisterRequest{
		Email:       strings.ToUpper(email),
		Password:    "password123",
		DisplayName: "Second",
	})
	if !errors.Is(err, exception.ErrEmailTaken) {
		t.Fatalf("second Register() error = %v, want %v", err, exception.ErrEmailTaken)
	}
}

func TestAuthServiceLogin(t *testing.T) {
	f := newAuthFixture(t)
	email := uuid.NewString() + "@example.com"
	password := "password123"

	registered, err := f.svc.Register(context.Background(), auth_request.RegisterRequest{
		Email:       email,
		Password:    password,
		DisplayName: "Raihan",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	loggedIn, err := f.svc.Login(context.Background(), auth_request.LoginRequest{
		Email:    strings.ToUpper(email),
		Password: password,
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	registerRefresh := f.findRefresh(t, registered.RefreshToken)
	loginRefresh := f.findRefresh(t, loggedIn.RefreshToken)
	if registerRefresh.FamilyID == loginRefresh.FamilyID {
		t.Error("login reused the register refresh family")
	}
}

func TestAuthServiceLoginRejectsInvalidCredentials(t *testing.T) {
	f := newAuthFixture(t)
	email := uuid.NewString() + "@example.com"

	if _, err := f.svc.Register(context.Background(), auth_request.RegisterRequest{
		Email:       email,
		Password:    "password123",
		DisplayName: "Raihan",
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if _, err := f.svc.Login(context.Background(), auth_request.LoginRequest{
		Email:    email,
		Password: "wrong-password",
	}); !errors.Is(err, exception.ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v, want %v", err, exception.ErrInvalidCredentials)
	}

	if _, err := f.svc.Login(context.Background(), auth_request.LoginRequest{
		Email:    uuid.NewString() + "@example.com",
		Password: "password123",
	}); !errors.Is(err, exception.ErrInvalidCredentials) {
		t.Fatalf("unknown email error = %v, want %v", err, exception.ErrInvalidCredentials)
	}
}

func TestAuthServiceRefreshRotatesToken(t *testing.T) {
	f := newAuthFixture(t)
	first := f.register(t, "password123")
	f.clock.Advance(time.Minute)

	second, err := f.svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Error("refresh token was not rotated")
	}

	oldToken := f.findRefresh(t, first.RefreshToken)
	newToken := f.findRefresh(t, second.RefreshToken)
	if oldToken.RotatedAt == nil {
		t.Error("old token RotatedAt = nil")
	}
	if oldToken.ReplacedByID == nil || *oldToken.ReplacedByID != newToken.ID {
		t.Errorf("old token ReplacedByID = %v, want %v", oldToken.ReplacedByID, newToken.ID)
	}
	if oldToken.FamilyID != newToken.FamilyID {
		t.Error("new token used a different family")
	}

	if _, err := f.tokens.Parse(first.AccessToken); err != nil {
		t.Errorf("old access token should still parse: %v", err)
	}
	if _, err := f.tokens.Parse(second.AccessToken); err != nil {
		t.Errorf("new access token should parse: %v", err)
	}
}

func TestAuthServiceRefreshExpiredToken(t *testing.T) {
	f := newAuthFixture(t)
	registered := f.register(t, "password123")
	f.clock.Advance(testRefreshTTL + time.Second)

	if _, err := f.svc.Refresh(context.Background(), registered.RefreshToken); !errors.Is(err, exception.ErrRefreshInvalid) {
		t.Fatalf("Refresh() error = %v, want %v", err, exception.ErrRefreshInvalid)
	}
}

func TestAuthServiceRefreshRevokedToken(t *testing.T) {
	f := newAuthFixture(t)
	registered := f.register(t, "password123")

	if err := f.svc.Logout(context.Background(), registered.RefreshToken); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if _, err := f.svc.Refresh(context.Background(), registered.RefreshToken); !errors.Is(err, exception.ErrRefreshInvalid) {
		t.Fatalf("Refresh() error = %v, want %v", err, exception.ErrRefreshInvalid)
	}
}

func TestAuthServiceRefreshReuseWithinGrace(t *testing.T) {
	f := newAuthFixture(t)
	registered := f.register(t, "password123")

	rotated, err := f.svc.Refresh(context.Background(), registered.RefreshToken)
	if err != nil {
		t.Fatalf("first Refresh() error = %v", err)
	}

	f.clock.Advance(testGrace - time.Second)
	if _, err := f.svc.Refresh(context.Background(), registered.RefreshToken); !errors.Is(err, exception.ErrRefreshRace) {
		t.Fatalf("second Refresh() error = %v, want %v", err, exception.ErrRefreshRace)
	}

	active := f.findRefresh(t, rotated.RefreshToken)
	if active.RevokedAt != nil {
		t.Error("active token was revoked during grace race")
	}
}

func TestAuthServiceRefreshReuseAfterGrace(t *testing.T) {
	f := newAuthFixture(t)
	registered := f.register(t, "password123")

	rotated, err := f.svc.Refresh(context.Background(), registered.RefreshToken)
	if err != nil {
		t.Fatalf("first Refresh() error = %v", err)
	}

	f.clock.Advance(testGrace + time.Second)
	if _, err := f.svc.Refresh(context.Background(), registered.RefreshToken); !errors.Is(err, exception.ErrRefreshReused) {
		t.Fatalf("second Refresh() error = %v, want %v", err, exception.ErrRefreshReused)
	}

	oldToken := f.findRefresh(t, registered.RefreshToken)
	activeToken := f.findRefresh(t, rotated.RefreshToken)
	if oldToken.RevokedAt == nil || activeToken.RevokedAt == nil {
		t.Error("family was not revoked after reuse")
	}
}

func TestAuthServiceLogoutInvalidToken(t *testing.T) {
	f := newAuthFixture(t)

	if err := f.svc.Logout(context.Background(), "not-a-refresh-token"); err != nil {
		t.Fatalf("Logout() error = %v, want nil", err)
	}
}

func TestAuthServiceMe(t *testing.T) {
	f := newAuthFixture(t)
	registered := f.register(t, "password123")

	userID, err := uuid.Parse(registered.User.ID)
	if err != nil {
		t.Fatalf("parse user ID: %v", err)
	}

	user, err := f.svc.Me(context.Background(), userID)
	if err != nil {
		t.Fatalf("Me() error = %v", err)
	}
	if user.ID != registered.User.ID {
		t.Errorf("Me() ID = %q, want %q", user.ID, registered.User.ID)
	}

	if _, err := f.svc.Me(context.Background(), uuid.New()); !errors.Is(err, exception.ErrUserNotFound) {
		t.Fatalf("Me() error = %v, want %v", err, exception.ErrUserNotFound)
	}
}

func TestAuthServiceRefreshConcurrent(t *testing.T) {
	pool := helper.OpenTestDB(t)
	ctx := context.Background()
	tokens := auth.NewAccessTokens([]byte(testJWTSecret), "live-auction", "live-auction-web", 15*time.Minute)
	svc := service.NewAuthService(
		userrepo.NewUserRepositoryImpl(),
		refreshrepo.NewRefreshTokenRepositoryImpl(),
		repository.NewTxManager(pool),
		tokens,
		time.Now,
		testRefreshTTL,
		testGrace,
	)

	registered, err := svc.Register(ctx, auth_request.RegisterRequest{
		Email:       uuid.NewString() + "@example.com",
		Password:    "password123",
		DisplayName: "Concurrent",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	userID, err := uuid.Parse(registered.User.ID)
	if err != nil {
		t.Fatalf("parse user ID: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "delete from users where id = $1", userID)
	}()

	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Refresh(ctx, registered.RefreshToken)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	var success, race int
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, exception.ErrRefreshRace):
			race++
		default:
			t.Errorf("unexpected Refresh() error = %v", err)
		}
	}

	if success != 1 || race != 1 {
		t.Fatalf("success = %d, race = %d, want 1 and 1", success, race)
	}
}
