package exception

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrEmailTaken           = errors.New("email already taken")
	ErrBidNotFound          = errors.New("bid not found")
	ErrBidAlreadyExists     = errors.New("bid already exists")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)
