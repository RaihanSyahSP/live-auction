package auction

import (
	"context"
	"live-auction/server/internal/model/domain"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AuctionRepository interface {
	Create(ctx context.Context, tx pgx.Tx, auction domain.Auction) (domain.Auction, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (domain.Auction, error)
	FindCurrentLive(ctx context.Context, tx pgx.Tx) (domain.Auction, error)
	List(ctx context.Context, tx pgx.Tx, limit int, offset int) ([]domain.Auction, error)
	ListEnded(ctx context.Context, tx pgx.Tx, limit int) ([]domain.ResultView, error)
	LockByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (domain.Auction, error)
	Update(ctx context.Context, tx pgx.Tx, auction domain.Auction) (domain.Auction, error)
	DeleteDraft(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	ListDue(ctx context.Context, tx pgx.Tx, now time.Time) ([]domain.Auction, error)
}
