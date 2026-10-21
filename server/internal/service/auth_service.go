package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/auth_request"
	"live-auction/server/internal/model/web/auth_response"
	"live-auction/server/internal/repository"
	refreshrepo "live-auction/server/internal/repository/refresh_token"
	userrepo "live-auction/server/internal/repository/user"
)

type AuthService interface {
	Register(ctx context.Context, req auth_request.RegisterRequest) (auth_response.AuthTokens, error)
	Login(ctx context.Context, req auth_request.LoginRequest) (auth_response.AuthTokens, error)
	Refresh(ctx context.Context, rawRefresh string) (auth_response.AuthTokens, error)
	Logout(ctx context.Context, rawRefresh string) error
	Me(ctx context.Context, userID uuid.UUID) (auth_response.UserResponse, error)
}

type authService struct {
	users      userrepo.UserRepository
	refresh    refreshrepo.RefreshTokenRepository
	tx         repository.TxManager
	tokens     *auth.AccessTokens
	now        func() time.Time
	refreshTTL time.Duration
	grace      time.Duration
	validate   *validator.Validate
}

func NewAuthService(
	users userrepo.UserRepository,
	refresh refreshrepo.RefreshTokenRepository,
	tx repository.TxManager,
	tokens *auth.AccessTokens,
	now func() time.Time,
	refreshTTL time.Duration,
	grace time.Duration,
) AuthService {
	if now == nil {
		now = time.Now
	}

	return &authService{
		users:      users,
		refresh:    refresh,
		tx:         tx,
		tokens:     tokens,
		now:        now,
		refreshTTL: refreshTTL,
		grace:      grace,
		validate:   validator.New(),
	}
}

func (s *authService) Register(
	ctx context.Context,
	req auth_request.RegisterRequest,
) (auth_response.AuthTokens, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if err := s.validate.Struct(req); err != nil {
		return auth_response.AuthTokens{}, exception.ErrValidation
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return auth_response.AuthTokens{}, err
	}

	var result auth_response.AuthTokens

	err = s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		user, err := s.users.Create(ctx, tx, domain.User{
			Email:        req.Email,
			PasswordHash: string(passwordHash),
			DisplayName:  req.DisplayName,
			Role:         domain.RoleUser,
		})
		if err != nil {
			return err
		}

		result, err = s.issueTokens(ctx, tx, user)
		return err
	})

	return result, err
}

func (s *authService) Login(
	ctx context.Context,
	req auth_request.LoginRequest,
) (auth_response.AuthTokens, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if err := s.validate.Struct(req); err != nil {
		return auth_response.AuthTokens{}, exception.ErrValidation
	}

	var result auth_response.AuthTokens

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		user, err := s.users.FindByEmail(ctx, tx, req.Email)
		if errors.Is(err, exception.ErrUserNotFound) {
			return exception.ErrInvalidCredentials
		}
		if err != nil {
			return err
		}

		if err := bcrypt.CompareHashAndPassword(
			[]byte(user.PasswordHash),
			[]byte(req.Password),
		); err != nil {
			return exception.ErrInvalidCredentials
		}

		result, err = s.issueTokens(ctx, tx, user)
		return err
	})

	return result, err
}

func (s *authService) Refresh(
	ctx context.Context,
	rawRefresh string,
) (auth_response.AuthTokens, error) {
	rawRefresh = strings.TrimSpace(rawRefresh)
	if rawRefresh == "" {
		return auth_response.AuthTokens{}, exception.ErrRefreshInvalid
	}

	hash := auth.HashRefreshToken(rawRefresh)

	var result auth_response.AuthTokens
	var reuseErr error

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		token, err := s.refresh.FindByHashForUpdate(ctx, tx, hash)
		if errors.Is(err, exception.ErrRefreshTokenNotFound) {
			return exception.ErrRefreshInvalid
		}
		if err != nil {
			return err
		}

		now := s.now().UTC()

		if token.RevokedAt != nil {
			return exception.ErrRefreshInvalid
		}
		if !now.Before(token.ExpiresAt) {
			return exception.ErrRefreshInvalid
		}

		if token.RotatedAt != nil {
			if token.ReplacedByID != nil &&
				now.Sub(*token.RotatedAt) <= s.grace {
				return exception.ErrRefreshRace
			}

			if err := s.refresh.RevokeFamily(ctx, tx, token.FamilyID, now); err != nil {
				return err
			}

			reuseErr = exception.ErrRefreshReused
			return nil
		}

		user, err := s.users.FindByID(ctx, tx, token.UserID)
		if errors.Is(err, exception.ErrUserNotFound) {
			return exception.ErrRefreshInvalid
		}
		if err != nil {
			return err
		}

		newRaw, newHash, err := auth.GenerateRefreshToken()
		if err != nil {
			return err
		}

		newToken, err := s.refresh.Create(ctx, tx, domain.RefreshToken{
			UserID:    token.UserID,
			FamilyID:  token.FamilyID,
			TokenHash: newHash,
			ExpiresAt: now.Add(s.refreshTTL),
		})
		if err != nil {
			return err
		}

		if _, err := s.refresh.Rotate(ctx, tx, token.ID, newToken.ID, now); err != nil {
			return err
		}

		accessToken, err := s.tokens.Issue(user.ID, user.Role)
		if err != nil {
			return err
		}

		result = auth_response.AuthTokens{
			User:         toUserResponse(user),
			AccessToken:  accessToken,
			RefreshToken: newRaw,
		}

		return nil
	})
	if err != nil {
		return auth_response.AuthTokens{}, err
	}
	if reuseErr != nil {
		return auth_response.AuthTokens{}, reuseErr
	}

	return result, nil
}

func (s *authService) Logout(ctx context.Context, rawRefresh string) error {
	rawRefresh = strings.TrimSpace(rawRefresh)
	if rawRefresh == "" {
		return nil
	}

	hash := auth.HashRefreshToken(rawRefresh)

	return s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		token, err := s.refresh.FindByHashForUpdate(ctx, tx, hash)
		if errors.Is(err, exception.ErrRefreshTokenNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		return s.refresh.RevokeFamily(ctx, tx, token.FamilyID, s.now().UTC())
	})
}

func (s *authService) Me(ctx context.Context, userID uuid.UUID) (auth_response.UserResponse, error) {
	var result auth_response.UserResponse

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		user, err := s.users.FindByID(ctx, tx, userID)
		if err != nil {
			return err
		}

		result = toUserResponse(user)
		return nil
	})

	return result, err
}

func (s *authService) issueTokens(
	ctx context.Context,
	tx pgx.Tx,
	user domain.User,
) (auth_response.AuthTokens, error) {
	accessToken, err := s.tokens.Issue(user.ID, user.Role)
	if err != nil {
		return auth_response.AuthTokens{}, err
	}

	rawRefresh, hash, err := auth.GenerateRefreshToken()
	if err != nil {
		return auth_response.AuthTokens{}, err
	}

	_, err = s.refresh.Create(ctx, tx, domain.RefreshToken{
		UserID:    user.ID,
		FamilyID:  uuid.New(),
		TokenHash: hash,
		ExpiresAt: s.now().Add(s.refreshTTL),
	})
	if err != nil {
		return auth_response.AuthTokens{}, err
	}

	return auth_response.AuthTokens{
		User:         toUserResponse(user),
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

func toUserResponse(user domain.User) auth_response.UserResponse {
	return auth_response.UserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        string(user.Role),
	}
}
