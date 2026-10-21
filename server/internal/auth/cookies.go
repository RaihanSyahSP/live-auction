package auth

import (
	"net/http"
	"time"
)

const (
	AccessCookieName  = "access_token"
	RefreshCookieName = "refresh_token"
)

type Cookies struct {
	secure     bool
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewCookies(secure bool, accessTTL, refreshTTL time.Duration) Cookies {
	return Cookies{
		secure:     secure,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (c Cookies) Set(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     AccessCookieName,
		Value:    accessToken,
		Path:     "/",
		MaxAge:   int(c.accessTTL.Seconds()),
		Expires:  time.Now().Add(c.accessTTL).UTC(),
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    refreshToken,
		Path:     "/api/auth",
		MaxAge:   int(c.refreshTTL.Seconds()),
		Expires:  time.Now().Add(c.refreshTTL).UTC(),
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (c Cookies) Clear(w http.ResponseWriter) {
	c.clear(w, AccessCookieName, "/")
	c.clear(w, RefreshCookieName, "/api/auth")
}

func (c Cookies) clear(w http.ResponseWriter, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	})
}
