package refresh_token

import (
	"context"
	"errors"
	"fmt"
	"time"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RefreshTokenRepositoryImpl struct{}

func NewRefreshTokenRepositoryImpl() *RefreshTokenRepositoryImpl {
	return &RefreshTokenRepositoryImpl{}
}

var _ RefreshTokenRepository = (*RefreshTokenRepositoryImpl)(nil)

func (r *RefreshTokenRepositoryImpl) Create(
	ctx context.Context,
	tx pgx.Tx,
	token domain.RefreshToken,
) (domain.RefreshToken, error) {
	const query = `
		insert into refresh_tokens (user_id, family_id, token_hash, expires_at)
		values ($1, $2, $3, $4)
		returning
			id,
			user_id,
			family_id,
			token_hash,
			expires_at,
			created_at,
			rotated_at,
			replaced_by_id,
			revoked_at
	`

	created, err := scanRefreshToken(tx.QueryRow(
		ctx,
		query,
		token.UserID,
		token.FamilyID,
		token.TokenHash,
		token.ExpiresAt,
	))
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("create refresh token: %w", err)
	}

	return created, nil
}

func (r *RefreshTokenRepositoryImpl) FindByHashForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	hash []byte,
) (domain.RefreshToken, error) {
	const query = `
		select
			id,
			user_id,
			family_id,
			token_hash,
			expires_at,
			created_at,
			rotated_at,
			replaced_by_id,
			revoked_at
		from refresh_tokens
		where token_hash = $1
		for update
	`

	token, err := scanRefreshToken(tx.QueryRow(ctx, query, hash))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RefreshToken{}, exception.ErrRefreshTokenNotFound
	}
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("find refresh token by hash for update: %w", err)
	}

	return token, nil
}

func (r *RefreshTokenRepositoryImpl) Rotate(
	ctx context.Context,
	tx pgx.Tx,
	oldID uuid.UUID,
	newID uuid.UUID,
	rotatedAt time.Time,
) (domain.RefreshToken, error) {
	const query = `
		update refresh_tokens
		set rotated_at = $1, replaced_by_id = $2
		where id = $3
		returning
			id,
			user_id,
			family_id,
			token_hash,
			expires_at,
			created_at,
			rotated_at,
			replaced_by_id,
			revoked_at
	`

	token, err := scanRefreshToken(tx.QueryRow(ctx, query, rotatedAt, newID, oldID))
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	return token, nil
}

func (r *RefreshTokenRepositoryImpl) RevokeFamily(
	ctx context.Context,
	tx pgx.Tx,
	familyID uuid.UUID,
	revokedAt time.Time,
) error {
	const query = `
		update refresh_tokens
		set revoked_at = $1
		where family_id = $2
		  and revoked_at is null
	`

	if _, err := tx.Exec(ctx, query, revokedAt, familyID); err != nil {
		return fmt.Errorf("revoke refresh token family: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepositoryImpl) DeleteExpired(
	ctx context.Context,
	tx pgx.Tx,
	now time.Time,
) error {
	const query = `
		delete from refresh_tokens
		where expires_at < $1
	`

	if _, err := tx.Exec(ctx, query, now); err != nil {
		return fmt.Errorf("delete expired refresh tokens: %w", err)
	}

	return nil
}

func scanRefreshToken(row pgx.Row) (domain.RefreshToken, error) {
	var token domain.RefreshToken

	err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.FamilyID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RotatedAt,
		&token.ReplacedByID,
		&token.RevokedAt,
	)
	if err != nil {
		return domain.RefreshToken{}, err
	}

	return token, nil
}
