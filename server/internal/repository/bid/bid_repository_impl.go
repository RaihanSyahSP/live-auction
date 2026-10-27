package bid

import (
	"context"
	"errors"
	"fmt"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type BidRepositoryImpl struct{}

func NewBidRepositoryImpl() *BidRepositoryImpl {
	return &BidRepositoryImpl{}
}

var _ BidRepository = (*BidRepositoryImpl)(nil)

func (b *BidRepositoryImpl) Insert(ctx context.Context, tx pgx.Tx, bid domain.Bid) (domain.Bid, error) {
	const query = `
		with inserted as (
			insert into bids (auction_id, user_id, amount, idempotency_key)
			values ($1, $2, $3, $4)
			returning id, auction_id, user_id, amount, idempotency_key, created_at
		)
		select
			i.id,
			i.auction_id,
			i.user_id,
			i.amount,
			i.idempotency_key,
			i.created_at,
			u.display_name
		from inserted i
		join users u on u.id = i.user_id
	`

	inserted, err := scanBid(tx.QueryRow(
		ctx,
		query,
		bid.AuctionID,
		bid.UserID,
		bid.Amount,
		bid.IdempotencyKey,
	))
	if err != nil {
		if isUniqueViolation(err, "bids_idempotency_idx") {
			return domain.Bid{}, exception.ErrBidAlreadyExists
		}
		return domain.Bid{}, fmt.Errorf("insert bid: %w", err)
	}

	return inserted, nil
}

func (b *BidRepositoryImpl) FindByIdempotencyKey(
	ctx context.Context,
	tx pgx.Tx,
	auctionID uuid.UUID,
	userID uuid.UUID,
	key uuid.UUID,
) (domain.Bid, error) {
	const query = `
		select
			b.id,
			b.auction_id,
			b.user_id,
			b.amount,
			b.idempotency_key,
			b.created_at,
			u.display_name
		from bids b
		join users u on u.id = b.user_id
		where b.auction_id = $1 and b.user_id = $2 and b.idempotency_key = $3
	`

	bid, err := scanBid(tx.QueryRow(ctx, query, auctionID, userID, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Bid{}, exception.ErrBidNotFound
	}
	if err != nil {
		return domain.Bid{}, fmt.Errorf("find bid by idempotency key: %w", err)
	}

	return bid, nil
}

func (b *BidRepositoryImpl) ListByAuction(
	ctx context.Context,
	tx pgx.Tx,
	auctionID uuid.UUID,
	beforeID *uuid.UUID,
	limit int,
) ([]domain.BidView, error) {
	const query = `
		select
			b.id,
			b.auction_id,
			b.user_id,
			b.amount,
			b.idempotency_key,
			b.created_at,
			u.display_name
		from bids b
		join users u on u.id = b.user_id
		where b.auction_id = $1
		  and ($2::uuid is null or b.id < $2)
		order by b.id desc
		limit $3
	`

	var before any
	if beforeID != nil {
		before = *beforeID
	}

	rows, err := tx.Query(ctx, query, auctionID, before, limit)
	if err != nil {
		return nil, fmt.Errorf("list bids by auction: %w", err)
	}
	defer rows.Close()

	var bids []domain.BidView
	for rows.Next() {
		bid, err := scanBidView(rows)
		if err != nil {
			return nil, fmt.Errorf("scan bid view: %w", err)
		}
		bids = append(bids, bid)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bid rows: %w", err)
	}

	return bids, nil
}

func scanBid(row pgx.Row) (domain.Bid, error) {
	var bid domain.Bid
	err := row.Scan(
		&bid.ID,
		&bid.AuctionID,
		&bid.UserID,
		&bid.Amount,
		&bid.IdempotencyKey,
		&bid.CreatedAt,
		&bid.DisplayName,
	)
	if err != nil {
		return domain.Bid{}, err
	}
	return bid, nil
}

func scanBidView(row pgx.Row) (domain.BidView, error) {
	var bid domain.BidView
	err := row.Scan(
		&bid.ID,
		&bid.AuctionID,
		&bid.UserID,
		&bid.Amount,
		&bid.IdempotencyKey,
		&bid.CreatedAt,
		&bid.DisplayName,
	)
	if err != nil {
		return domain.BidView{}, err
	}
	return bid, nil
}

func isUniqueViolation(err error, constraintName string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == constraintName
}
