package user

import (
	"context"
	"live-auction/server/internal/model/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	Create(ctx context.Context, tx pgx.Tx, user domain.User) (domain.User, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (domain.User, error)
	FindByEmail(ctx context.Context, tx pgx.Tx, email string) (domain.User, error)
}
