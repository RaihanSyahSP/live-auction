package bid

import (
	"context"

	"live-auction/server/internal/model/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type BidRepository interface {
	Insert(ctx context.Context, tx pgx.Tx, bid domain.Bid) (domain.Bid, error)
	FindByIdempotencyKey(ctx context.Context, tx pgx.Tx, auctionID uuid.UUID, userID uuid.UUID, key uuid.UUID) (domain.Bid, error)
	ListByAuction(ctx context.Context, tx pgx.Tx, auctionID uuid.UUID, beforeID *uuid.UUID, limit int) ([]domain.BidView, error)
	ListByUserLatest(ctx context.Context, tx pgx.Tx, userID uuid.UUID, limit int, offset int) ([]domain.MyBid, int, error)
	ListByUserAll(ctx context.Context, tx pgx.Tx, userID uuid.UUID, limit int, offset int) ([]domain.MyBid, int, error)
}
