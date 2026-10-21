package refresh_token

import (
	"context"
	"live-auction/server/internal/model/domain"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, tx pgx.Tx, token string) (domain.RefreshToken, error)
	FindByHashForUpdate(ctx context.Context, tx pgx.Tx, hash []byte) (domain.RefreshToken, error)
	Rotate(ctx context.Context, tx pgx.Tx, oldID uuid.UUID, newID uuid.UUID, rotatedAt time.Time) (domain.RefreshToken, error)
	RevokeFamily(ctx context.Context, tx pgx.Tx, familyID uuid.UUID, revokedAt time.Time) error
	DeleteExpired(ctx context.Context, tx pgx.Tx, now time.Time) error
}
