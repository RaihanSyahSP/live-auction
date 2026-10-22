# Task 7: Auth Controller, Middleware, dan Routes

Status: panduan implementasi manual.

Task 7 menghubungkan `AuthService` ke HTTP. Cookie dipasang di controller. Middleware menangani access token, admin, Origin, dan rate limit.

## File yang dibuat atau diubah

```text
server/
  internal/
    helper/json.go
    exception/error_handler.go
    exception/errors.go
    middleware/auth.go
    middleware/admin.go
    middleware/origin.go
    middleware/rate_limit.go
    controller/auth_controller.go
    controller/auth_controller_test.go
    app/router.go
    app/router_test.go
  cmd/api/main.go
```

Dependency yang perlu ditambahkan:

```powershell
cd D:\live-auction\server
go get golang.org/x/time/rate@latest
```

`jwt`, validator, dan bcrypt sudah ada dari Task 5 dan 6.

---

## 1. Helper JSON

Buat `internal/helper/json.go`:

```go
package helper

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("body must contain a single JSON object")
	}

	return nil
}
```

---

## 2. Error tambahan

Di `internal/exception/errors.go`, tambahkan:

```go
ErrUnauthenticated = errors.New("unauthenticated")
ErrForbidden       = errors.New("forbidden")
ErrRateLimited     = errors.New("rate limited")
```

---

## 3. Error handler

Buat `internal/exception/error_handler.go`:

```go
package exception

import (
	"errors"
	"net/http"

	"live-auction/server/internal/helper"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

func WriteError(w http.ResponseWriter, err error) {
	status, code, message := mapError(err)
	helper.WriteJSON(w, status, errorResponse{
		Error: errorBody{
			Code:    code,
			Message: message,
		},
	})
}

func mapError(err error) (int, string, string) {
	switch {
	case errors.Is(err, ErrValidation):
		return http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Data tidak valid"
	case errors.Is(err, ErrEmailTaken):
		return http.StatusConflict, "EMAIL_TAKEN", "Email sudah terdaftar"
	case errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email atau password salah"
	case errors.Is(err, ErrUnauthenticated):
		return http.StatusUnauthorized, "UNAUTHENTICATED", "Silakan login"
	case errors.Is(err, ErrTokenExpired):
		return http.StatusUnauthorized, "TOKEN_EXPIRED", "Sesi sudah berakhir"
	case errors.Is(err, ErrTokenInvalid):
		return http.StatusUnauthorized, "TOKEN_INVALID", "Token tidak valid"
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN", "Akses ditolak"
	case errors.Is(err, ErrUserNotFound):
		return http.StatusNotFound, "USER_NOT_FOUND", "User tidak ditemukan"
	case errors.Is(err, ErrRefreshInvalid):
		return http.StatusUnauthorized, "REFRESH_INVALID", "Refresh token tidak valid"
	case errors.Is(err, ErrRefreshReused):
		return http.StatusUnauthorized, "REFRESH_REUSED", "Refresh token sudah dipakai"
	case errors.Is(err, ErrRefreshRace):
		return http.StatusConflict, "REFRESH_RACE", "Refresh token sedang diproses"
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests, "RATE_LIMITED", "Terlalu banyak percobaan"
	default:
		return http.StatusInternalServerError, "INTERNAL", "Terjadi kesalahan"
	}
}
```

---

## 4. Middleware auth

Buat `internal/middleware/auth.go`:

```go
package middleware

import (
	"context"
	"net/http"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/exception"
)

type principalContextKey struct{}

func Auth(tokens *auth.AccessTokens, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.AccessCookieName)
		if err != nil {
			exception.WriteError(w, exception.ErrUnauthenticated)
			return
		}

		principal, err := tokens.Parse(cookie.Value)
		if err != nil {
			exception.WriteError(w, err)
			return
		}

		ctx := WithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithPrincipal(ctx context.Context, principal auth.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFrom(ctx context.Context) (auth.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(auth.Principal)
	return principal, ok
}
```

---

## 5. Middleware admin

Buat `internal/middleware/admin.go`:

```go
package middleware

import (
	"net/http"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"
)

func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFrom(r.Context())
		if !ok || principal.Role != domain.RoleAdmin {
			exception.WriteError(w, exception.ErrForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
```

---

## 6. Middleware Origin

Buat `internal/middleware/origin.go`:

```go
package middleware

import (
	"net/http"
	"strings"

	"live-auction/server/internal/exception"
)

func Origin(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && origin != allowedOrigin {
			exception.WriteError(w, exception.ErrForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
```

Origin kosong diizinkan untuk curl dan API client. Browser modern mengirim Origin untuk POST.

---

## 7. Rate limit

Buat `internal/middleware/rate_limit.go`:

```go
package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"live-auction/server/internal/exception"
)

type RateLimiter struct {
	mu       sync.Mutex
	limit    rate.Limit
	burst    int
	visitors map[string]*rate.Limiter
}

func NewRateLimiter(limit rate.Limit, burst int) *RateLimiter {
	return &RateLimiter{
		limit:    limit,
		burst:    burst,
		visitors: make(map[string]*rate.Limiter),
	}
}

func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	limiter := l.visitors[key]
	if limiter == nil {
		limiter = rate.NewLimiter(l.limit, l.burst)
		l.visitors[key] = limiter
	}
	l.mu.Unlock()

	return limiter.Allow()
}

func RateLimit(
	limiter *RateLimiter,
	keyFunc func(*http.Request) string,
	next http.Handler,
) http.Handler {
	if limiter == nil {
		return next
	}
	if keyFunc == nil {
		keyFunc = ClientIP
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(keyFunc(r)) {
			exception.WriteError(w, exception.ErrRateLimited)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
```

Catatan: map disimpan di memori dan tidak dibersihkan. Untuk satu instance ini cukup. Kalau nanti multi-instance, pindahkan ke Redis.

---

## 8. Auth controller

Buat `internal/controller/auth_controller.go`:

```go
package controller

import (
	"net/http"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/model/web/auth_request"
	"live-auction/server/internal/service"
)

type AuthController struct {
	authService service.AuthService
	cookies     auth.Cookies
}

func NewAuthController(authService service.AuthService, cookies auth.Cookies) *AuthController {
	return &AuthController{
		authService: authService,
		cookies:     cookies,
	}
}

func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var req auth_request.RegisterRequest
	if err := helper.ReadJSON(w, r, &req); err != nil {
		exception.WriteError(w, exception.ErrValidation)
		return
	}

	result, err := c.authService.Register(r.Context(), req)
	if err != nil {
		exception.WriteError(w, err)
		return
	}

	c.cookies.Set(w, result.AccessToken, result.RefreshToken)
	helper.WriteJSON(w, http.StatusCreated, result.User)
}

func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req auth_request.LoginRequest
	if err := helper.ReadJSON(w, r, &req); err != nil {
		exception.WriteError(w, exception.ErrValidation)
		return
	}

	result, err := c.authService.Login(r.Context(), req)
	if err != nil {
		exception.WriteError(w, err)
		return
	}

	c.cookies.Set(w, result.AccessToken, result.RefreshToken)
	helper.WriteJSON(w, http.StatusOK, result.User)
}

func (c *AuthController) Refresh(w http.ResponseWriter, r *http.Request) {
	rawRefresh, err := refreshCookie(r)
	if err != nil {
		exception.WriteError(w, exception.ErrRefreshInvalid)
		return
	}

	result, err := c.authService.Refresh(r.Context(), rawRefresh)
	if err != nil {
		c.cookies.Clear(w)
		exception.WriteError(w, err)
		return
	}

	c.cookies.Set(w, result.AccessToken, result.RefreshToken)
	helper.WriteJSON(w, http.StatusOK, result.User)
}

func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	rawRefresh, _ := refreshCookie(r)

	if err := c.authService.Logout(r.Context(), rawRefresh); err != nil {
		exception.WriteError(w, err)
		return
	}

	c.cookies.Clear(w)
	w.WriteHeader(http.StatusNoContent)
}

func (c *AuthController) Me(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		exception.WriteError(w, exception.ErrUnauthenticated)
		return
	}

	user, err := c.authService.Me(r.Context(), principal.UserID)
	if err != nil {
		exception.WriteError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, user)
}

func refreshCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(auth.RefreshCookieName)
	if err != nil {
		return "", exception.ErrRefreshInvalid
	}
	return cookie.Value, nil
}
```

---

## 9. Router

Ganti `internal/app/router.go`:

```go
package app

import (
	"net/http"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/controller"
	"live-auction/server/internal/middleware"
)

type RouterDeps struct {
	Health         *controller.HealthController
	Auth           *controller.AuthController
	AccessTokens   *auth.AccessTokens
	AllowedOrigin  string
	LoginLimiter   *middleware.RateLimiter
	RefreshLimiter *middleware.RateLimiter
}

func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	if deps.Health != nil {
		mux.HandleFunc("GET /healthz", deps.Health.Health)
	}

	if deps.Auth == nil {
		return mux
	}

	register := http.HandlerFunc(deps.Auth.Register)
	login := http.HandlerFunc(deps.Auth.Login)
	refresh := http.HandlerFunc(deps.Auth.Refresh)
	logout := http.HandlerFunc(deps.Auth.Logout)
	me := middleware.Auth(
		deps.AccessTokens,
		http.HandlerFunc(deps.Auth.Me),
	)

	mux.Handle(
		"POST /api/auth/register",
		middleware.Origin(
			deps.AllowedOrigin,
			middleware.RateLimit(deps.LoginLimiter, middleware.ClientIP, register),
		),
	)
	mux.Handle(
		"POST /api/auth/login",
		middleware.Origin(
			deps.AllowedOrigin,
			middleware.RateLimit(deps.LoginLimiter, middleware.ClientIP, login),
		),
	)
	mux.Handle(
		"POST /api/auth/refresh",
		middleware.Origin(
			deps.AllowedOrigin,
			middleware.RateLimit(deps.RefreshLimiter, middleware.ClientIP, refresh),
		),
	)
	mux.Handle(
		"POST /api/auth/logout",
		middleware.Origin(deps.AllowedOrigin, logout),
	)
	mux.Handle("GET /api/auth/me", me)

	return mux
}
```

---

## 10. Update router test

Ubah pemanggilan di `internal/app/router_test.go`:

```go
router := app.NewRouter(app.RouterDeps{
	Health: controller.NewHealthController(),
})
```

Bukan lagi:

```go
router := app.NewRouter(controller.NewHealthController())
```

---

## 11. Wiring main

Sesuaikan `cmd/api/main.go` sampai bagian pembuatan handler:

```go
import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	"live-auction/server/internal/app"
	"live-auction/server/internal/auth"
	"live-auction/server/internal/config"
	"live-auction/server/internal/controller"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/repository"
	refreshrepo "live-auction/server/internal/repository/refresh_token"
	userrepo "live-auction/server/internal/repository/user"
	"live-auction/server/internal/service"
)
```

Lalu setelah `config.Load()`:

```go
databaseCtx, cancelDB := context.WithTimeout(context.Background(), 10*time.Second)
defer cancelDB()

pool, err := app.NewDB(databaseCtx, cfg)
if err != nil {
	slog.Error("connect database", "error", err)
	os.Exit(1)
}
defer pool.Close()

userRepository := userrepo.NewUserRepositoryImpl()
refreshRepository := refreshrepo.NewRefreshTokenRepositoryImpl()
txManager := repository.NewTxManager(pool)

accessTokens := auth.NewAccessTokens(
	[]byte(cfg.JWTSecret),
	"live-auction",
	"live-auction-web",
	cfg.AccessTokenTTL,
)
cookies := auth.NewCookies(
	cfg.CookieSecure,
	cfg.AccessTokenTTL,
	cfg.RefreshTokenTTL,
)

authService := service.NewAuthService(
	userRepository,
	refreshRepository,
	txManager,
	accessTokens,
	time.Now,
	cfg.RefreshTokenTTL,
	cfg.RefreshGracePeriod,
)

authController := controller.NewAuthController(authService, cookies)

loginLimiter := middleware.NewRateLimiter(
	rate.Every(6*time.Second), // 10 per menit
	10,
)
refreshLimiter := middleware.NewRateLimiter(
	rate.Every(2*time.Second), // 30 per menit
	30,
)

handler := app.NewRouter(app.RouterDeps{
	Health:         controller.NewHealthController(),
	Auth:           authController,
	AccessTokens:   accessTokens,
	AllowedOrigin:  cfg.AppOrigin,
	LoginLimiter:   loginLimiter,
	RefreshLimiter: refreshLimiter,
})
```

Sisanya tetap:

```go
server := app.NewServer(cfg, handler)
```

---

## 12. Test controller

Buat `internal/controller/auth_controller_test.go`.

Fake service:

```go
package controller_test

import (
	"context"
	"encoding/json"
	"errors"
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
```

Helper:

```go
func newTestAuthController(svc *fakeAuthService) *controller.AuthController {
	return controller.NewAuthController(
		svc,
		auth.NewCookies(false, 15*time.Minute, 168*time.Hour),
	)
}
```

Contoh test register:

```go
func TestAuthControllerRegister(t *testing.T) {
	userID := uuid.New()
	svc := &fakeAuthService{
		registerResult: auth_response.AuthTokens{
			User: auth_response.UserResponse{
				ID:          userID.String(),
				Email:       "raihan@example.com",
				DisplayName: "Raihan",
				Role:        "user",
			},
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
		},
	}

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

	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(cookies))
	}
}
```

Test yang perlu dibuat:

- register 201 dan dua cookie
- login 200 dan user
- refresh 200 dan dua cookie baru
- refresh invalid 401 `REFRESH_INVALID`
- me 200 dengan principal
- me 401 tanpa principal
- malformed JSON 422 `VALIDATION_ERROR`
- logout 204 dan dua cookie clear

Contoh test me dengan principal:

```go
func TestAuthControllerMe(t *testing.T) {
	svc := &fakeAuthService{
		meResult: auth_response.UserResponse{
			ID:          uuid.NewString(),
			Email:       "raihan@example.com",
			DisplayName: "Raihan",
			Role:        "user",
		},
	}
	handler := newTestAuthController(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req = req.WithContext(middleware.WithPrincipal(req.Context(), auth.Principal{
		UserID: uuid.New(),
		Role:   "user",
	}))
	rec := httptest.NewRecorder()

	handler.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
```

---

## 13. Test middleware

Test admin dapat memakai `Auth` lalu `Admin` dengan access token asli:

```go
func TestAdminMiddleware(t *testing.T) {
	tokens := auth.NewAccessTokens(
		[]byte("01234567890123456789012345678901"),
		"live-auction",
		"live-auction-web",
		time.Minute,
	)
	userID := uuid.New()

	adminToken, _ := tokens.Issue(userID, domain.RoleAdmin)
	userToken, _ := tokens.Issue(userID, domain.RoleUser)

	handler := middleware.Auth(tokens, middleware.Admin(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	))

	for _, tt := range []struct {
		name string
		raw  string
		want int
	}{
		{name: "admin", raw: adminToken, want: http.StatusOK},
		{name: "user", raw: userToken, want: http.StatusForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/test", nil)
			req.AddCookie(&http.Cookie{Name: auth.AccessCookieName, Value: tt.raw})
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
```

Import `domain` dan `auth` di test.

Test Origin:

```go
func TestOriginRejectsWrongOrigin(t *testing.T) {
	handler := middleware.Origin("http://localhost:5173",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
```

Test rate limit:

```go
func TestRateLimit(t *testing.T) {
	limiter := middleware.NewRateLimiter(rate.Every(time.Hour), 1)
	handler := middleware.RateLimit(limiter, func(r *http.Request) string {
		return "test"
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d", first.Code)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429", second.Code)
	}
}
```

---

## 14. Verifikasi

```powershell
cd D:\live-auction\server

go mod tidy
go test ./...
go build ./...
go vet ./...
```

Untuk menjalankan test repository dan service yang membutuhkan database:

```powershell
$env:TEST_DATABASE_URL="postgres://postgres:root@localhost:5432/live_auction_test?sslmode=disable"
go test ./...
```

## Checklist selesai

- [ ] `helper/json.go` dibuat
- [ ] `exception/error_handler.go` dibuat
- [ ] `ErrUnauthenticated`, `ErrForbidden`, `ErrRateLimited` ditambahkan
- [ ] `middleware/auth.go` dibuat
- [ ] `middleware/admin.go` dibuat
- [ ] `middleware/origin.go` dibuat
- [ ] `middleware/rate_limit.go` dibuat
- [ ] `controller/auth_controller.go` dibuat
- [ ] `app.NewRouter` menerima `RouterDeps`
- [ ] `cmd/api/main.go` memakai `AuthService`, `AuthController`, dan limiter
- [ ] test register, login, refresh, me, logout
- [ ] test admin, Origin, dan rate limit
- [ ] `go test ./...` lulus
- [ ] `go build ./...` lulus
- [ ] `go vet ./...` lulus

Setelah Task 7 selesai, cookie auth sudah berjalan lewat HTTP. Task 8 berikutnya adalah auction controller dan admin endpoints.
