package controller_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/controller"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/auth_request"
	"live-auction/server/internal/model/web/auth_response"
)

type fakeAuthService struct {
	registerResult auth_response.AuthTokens
	registerErr    error
	loginResult    auth_response.AuthTokens
	loginErr       error
	refreshResult  auth_response.AuthTokens
	refreshErr     error
	logoutErr      error
	meResult       auth_response.UserResponse
	meErr          error
}

func (f *fakeAuthService) Register(ctx context.Context, req auth_request.RegisterRequest) (auth_response.AuthTokens, error) {
	return f.registerResult, f.registerErr
}

func (f *fakeAuthService) Login(ctx context.Context, req auth_request.LoginRequest) (auth_response.AuthTokens, error) {
	return f.loginResult, f.loginErr
}

func (f *fakeAuthService) Refresh(ctx context.Context, rawRefresh string) (auth_response.AuthTokens, error) {
	return f.refreshResult, f.refreshErr
}

func (f *fakeAuthService) Logout(ctx context.Context, rawRefresh string) error {
	return f.logoutErr
}

func (f *fakeAuthService) Me(ctx context.Context, userID uuid.UUID) (auth_response.UserResponse, error) {
	return f.meResult, f.meErr
}

func newTestAuthController(svc *fakeAuthService) *controller.AuthController {
	return controller.NewAuthController(
		svc,
		auth.NewCookies(false, 15*time.Minute, 168*time.Hour),
	)
}

func cookiesByName(t *testing.T, rec *httptest.ResponseRecorder) map[string]*http.Cookie {
	t.Helper()

	result := rec.Result()
	defer result.Body.Close()

	cookies := make(map[string]*http.Cookie)
	for _, cookie := range result.Cookies() {
		cookies[cookie.Name] = cookie
	}
	return cookies
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body.Error.Code
}

func testUserResponse() auth_response.UserResponse {
	return auth_response.UserResponse{
		ID:          uuid.NewString(),
		Email:       "raihan@example.com",
		DisplayName: "Raihan",
		Role:        "user",
	}
}

func testAuthTokens() auth_response.AuthTokens {
	return auth_response.AuthTokens{
		User:         testUserResponse(),
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}
}

func TestAuthControllerRegister(t *testing.T) {
	svc := &fakeAuthService{registerResult: testAuthTokens()}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/register",
		strings.NewReader(`{"email":"raihan@example.com","password":"password123","display_name":"Raihan"}`),
	)
	rec := httptest.NewRecorder()
	handler.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var got auth_response.UserResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Email != "raihan@example.com" {
		t.Errorf("email = %q, want raihan@example.com", got.Email)
	}

	cookies := cookiesByName(t, rec)
	if len(cookies) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(cookies))
	}
	if cookies[auth.AccessCookieName].Value != "access-token" {
		t.Error("access cookie value is wrong")
	}
	if cookies[auth.RefreshCookieName].Value != "refresh-token" {
		t.Error("refresh cookie value is wrong")
	}
}

func TestAuthControllerRegisterError(t *testing.T) {
	svc := &fakeAuthService{registerErr: exception.ErrEmailTaken}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/register",
		strings.NewReader(`{"email":"raihan@example.com","password":"password123","display_name":"Raihan"}`),
	)
	rec := httptest.NewRecorder()
	handler.Register(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if code := errorCode(t, rec); code != "EMAIL_TAKEN" {
		t.Errorf("error code = %q, want EMAIL_TAKEN", code)
	}
	if cookies := cookiesByName(t, rec); len(cookies) != 0 {
		t.Errorf("cookie count = %d, want 0", len(cookies))
	}
}

func TestAuthControllerRegisterRejectsMalformedJSON(t *testing.T) {
	svc := &fakeAuthService{}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{`))
	rec := httptest.NewRecorder()
	handler.Register(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if code := errorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Errorf("error code = %q, want VALIDATION_ERROR", code)
	}
}

func TestAuthControllerLogin(t *testing.T) {
	svc := &fakeAuthService{loginResult: testAuthTokens()}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/login",
		strings.NewReader(`{"email":"raihan@example.com","password":"password123"}`),
	)
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if cookies := cookiesByName(t, rec); len(cookies) != 2 {
		t.Errorf("cookie count = %d, want 2", len(cookies))
	}
}

func TestAuthControllerLoginError(t *testing.T) {
	svc := &fakeAuthService{loginErr: exception.ErrInvalidCredentials}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auth/login",
		strings.NewReader(`{"email":"raihan@example.com","password":"wrong-password"}`),
	)
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if code := errorCode(t, rec); code != "INVALID_CREDENTIALS" {
		t.Errorf("error code = %q, want INVALID_CREDENTIALS", code)
	}
	if cookies := cookiesByName(t, rec); len(cookies) != 0 {
		t.Errorf("cookie count = %d, want 0", len(cookies))
	}
}

func TestAuthControllerRefresh(t *testing.T) {
	svc := &fakeAuthService{refreshResult: testAuthTokens()}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: auth.RefreshCookieName, Value: "old-refresh-token"})
	rec := httptest.NewRecorder()
	handler.Refresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	cookies := cookiesByName(t, rec)
	if len(cookies) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(cookies))
	}
	if cookies[auth.AccessCookieName].Value != "access-token" {
		t.Error("access cookie value is wrong")
	}
	if cookies[auth.RefreshCookieName].Value != "refresh-token" {
		t.Error("refresh cookie value is wrong")
	}
}

func TestAuthControllerRefreshMissingCookie(t *testing.T) {
	svc := &fakeAuthService{}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	rec := httptest.NewRecorder()
	handler.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if code := errorCode(t, rec); code != "REFRESH_INVALID" {
		t.Errorf("error code = %q, want REFRESH_INVALID", code)
	}
}

func TestAuthControllerRefreshInvalidClearsCookies(t *testing.T) {
	svc := &fakeAuthService{refreshErr: exception.ErrRefreshInvalid}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: auth.RefreshCookieName, Value: "invalid-refresh-token"})
	rec := httptest.NewRecorder()
	handler.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if code := errorCode(t, rec); code != "REFRESH_INVALID" {
		t.Errorf("error code = %q, want REFRESH_INVALID", code)
	}

	cookies := cookiesByName(t, rec)
	if len(cookies) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(cookies))
	}
	for _, name := range []string{auth.AccessCookieName, auth.RefreshCookieName} {
		if cookies[name].MaxAge != -1 {
			t.Errorf("%s MaxAge = %d, want -1", name, cookies[name].MaxAge)
		}
	}
}

func TestAuthControllerMe(t *testing.T) {
	svc := &fakeAuthService{meResult: testUserResponse()}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req = req.WithContext(middleware.WithPrincipal(req.Context(), auth.Principal{
		UserID: uuid.New(),
		Role:   domain.RoleUser,
	}))
	rec := httptest.NewRecorder()
	handler.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got auth_response.UserResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Email != "raihan@example.com" {
		t.Errorf("email = %q, want raihan@example.com", got.Email)
	}
}

func TestAuthControllerMeUnauthenticated(t *testing.T) {
	svc := &fakeAuthService{}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	handler.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if code := errorCode(t, rec); code != "UNAUTHENTICATED" {
		t.Errorf("error code = %q, want UNAUTHENTICATED", code)
	}
}

func TestAuthControllerLogout(t *testing.T) {
	svc := &fakeAuthService{}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: auth.RefreshCookieName, Value: "refresh-token"})
	rec := httptest.NewRecorder()
	handler.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	cookies := cookiesByName(t, rec)
	if len(cookies) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(cookies))
	}
	for _, name := range []string{auth.AccessCookieName, auth.RefreshCookieName} {
		if cookies[name].Value != "" {
			t.Errorf("%s value = %q, want empty", name, cookies[name].Value)
		}
		if cookies[name].MaxAge != -1 {
			t.Errorf("%s MaxAge = %d, want -1", name, cookies[name].MaxAge)
		}
	}
}
