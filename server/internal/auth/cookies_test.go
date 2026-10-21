package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"live-auction/server/internal/auth"
)

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

func TestCookiesSet(t *testing.T) {
	cookies := auth.NewCookies(false, 15*time.Minute, 168*time.Hour)
	rec := httptest.NewRecorder()

	cookies.Set(rec, "access-token", "refresh-token")
	byName := cookiesByName(t, rec)

	if len(byName) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(byName))
	}

	access := byName[auth.AccessCookieName]
	if access == nil {
		t.Fatal("access cookie is missing")
	}
	if access.Value != "access-token" {
		t.Errorf("access value = %q", access.Value)
	}
	if access.Path != "/" {
		t.Errorf("access path = %q, want /", access.Path)
	}
	if !access.HttpOnly {
		t.Error("access HttpOnly = false, want true")
	}
	if access.Secure {
		t.Error("access Secure = true, want false")
	}
	if access.SameSite != http.SameSiteLaxMode {
		t.Errorf("access SameSite = %v, want Lax", access.SameSite)
	}
	if access.MaxAge != 900 {
		t.Errorf("access MaxAge = %d, want 900", access.MaxAge)
	}
	if access.Expires.IsZero() {
		t.Error("access Expires is zero")
	}

	refresh := byName[auth.RefreshCookieName]
	if refresh == nil {
		t.Fatal("refresh cookie is missing")
	}
	if refresh.Value != "refresh-token" {
		t.Errorf("refresh value = %q", refresh.Value)
	}
	if refresh.Path != "/api/auth" {
		t.Errorf("refresh path = %q, want /api/auth", refresh.Path)
	}
	if !refresh.HttpOnly {
		t.Error("refresh HttpOnly = false, want true")
	}
	if refresh.Secure {
		t.Error("refresh Secure = true, want false")
	}
	if refresh.SameSite != http.SameSiteLaxMode {
		t.Errorf("refresh SameSite = %v, want Lax", refresh.SameSite)
	}
	if refresh.MaxAge != 604800 {
		t.Errorf("refresh MaxAge = %d, want 604800", refresh.MaxAge)
	}
	if refresh.Expires.IsZero() {
		t.Error("refresh Expires is zero")
	}
}

func TestCookiesSetSecure(t *testing.T) {
	cookies := auth.NewCookies(true, time.Minute, time.Hour)
	rec := httptest.NewRecorder()

	cookies.Set(rec, "access-token", "refresh-token")
	byName := cookiesByName(t, rec)

	for _, name := range []string{auth.AccessCookieName, auth.RefreshCookieName} {
		cookie := byName[name]
		if cookie == nil {
			t.Fatalf("%s cookie is missing", name)
		}
		if !cookie.Secure {
			t.Errorf("%s Secure = false, want true", name)
		}
	}
}

func TestCookiesClear(t *testing.T) {
	cookies := auth.NewCookies(true, 15*time.Minute, 168*time.Hour)
	rec := httptest.NewRecorder()

	cookies.Clear(rec)
	byName := cookiesByName(t, rec)

	if len(byName) != 2 {
		t.Fatalf("cookie count = %d, want 2", len(byName))
	}

	expectedPaths := map[string]string{
		auth.AccessCookieName:  "/",
		auth.RefreshCookieName: "/api/auth",
	}
	for name, path := range expectedPaths {
		cookie := byName[name]
		if cookie == nil {
			t.Fatalf("%s cookie is missing", name)
		}
		if cookie.Value != "" {
			t.Errorf("%s value = %q, want empty", name, cookie.Value)
		}
		if cookie.Path != path {
			t.Errorf("%s path = %q, want %q", name, cookie.Path, path)
		}
		if cookie.MaxAge != -1 {
			t.Errorf("%s MaxAge = %d, want -1", name, cookie.MaxAge)
		}
		if !cookie.HttpOnly {
			t.Errorf("%s HttpOnly = false, want true", name)
		}
		if !cookie.Secure {
			t.Errorf("%s Secure = false, want true", name)
		}
		if cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("%s SameSite = %v, want Lax", name, cookie.SameSite)
		}
	}
}
