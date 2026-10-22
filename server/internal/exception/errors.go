package exception

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrEmailTaken           = errors.New("email already taken")
	ErrBidNotFound          = errors.New("bid not found")
	ErrBidAlreadyExists     = errors.New("bid already exists")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrTokenExpired         = errors.New("token expired")
	ErrTokenInvalid         = errors.New("token invalid")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrValidation           = errors.New("validation error")
	ErrRefreshInvalid       = errors.New("refresh token invalid")
	ErrRefreshReused        = errors.New("refresh token reused")
	ErrRefreshRace          = errors.New("refresh token race")
	ErrUnauthenticated      = errors.New("unauthenticated")
	ErrForbidden            = errors.New("forbidden")
	ErrRateLimited          = errors.New("rate limited")
	ErrAuctionNotFound      = errors.New("auction not found")
	ErrAuctionNotDraft      = errors.New("auction is not draft")
	ErrAuctionNotLive       = errors.New("auction is not live")
	ErrAuctionAlreadyLive   = errors.New("another auction is already live")
)
