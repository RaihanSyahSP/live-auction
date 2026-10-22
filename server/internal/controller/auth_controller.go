package controller

import (
	"live-auction/server/internal/auth"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/model/web/auth_request"
	"live-auction/server/internal/service"
	"net/http"
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
