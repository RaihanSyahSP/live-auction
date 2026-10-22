package exception

import (
	"errors"
	"live-auction/server/internal/helper"
	"net/http"
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
