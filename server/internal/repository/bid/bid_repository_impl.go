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

func (b *BidRepositoryImpl) ListByUserLatest(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	limit int,
	offset int,
) ([]domain.MyBid, int, error) {
	const query = `
		with latest as (
			select distinct on (b.auction_id)
				b.id,
				b.auction_id,
				b.user_id,
				b.amount,
				b.idempotency_key,
				b.created_at,
				a.lot,
				a.title,
				a.subtitle,
				a.status as auction_status,
				a.current_price,
				a.final_price,
				a.ends_at,
				a.ended_at,
				case
					when a.status = 'live' and a.highest_bidder_id = b.user_id then 'leading'
					when a.status = 'live' then 'outbid'
					when a.status = 'ended' and a.winner_id = b.user_id then 'won'
					else 'lost'
				end as bid_status
			from bids b
			join auctions a on a.id = b.auction_id
			where b.user_id = $1
			order by b.auction_id, b.created_at desc, b.id desc
		)
		select
			id, auction_id, user_id, amount, idempotency_key, created_at,
			lot, title, subtitle, auction_status, current_price, final_price,
			ends_at, ended_at, bid_status
		from latest
		order by created_at desc, id desc
		limit $2 offset $3
	`

	rows, err := tx.Query(ctx, query, userID, limit, offset)
	items, err := scanMyBids(rows, err)
	if err != nil {
		return nil, 0, fmt.Errorf("list latest bids by user: %w", err)
	}

	var total int
	if err := tx.QueryRow(
		ctx,
		`select count(distinct auction_id) from bids where user_id = $1`,
		userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count latest auctions by user: %w", err)
	}

	return items, total, nil
}

func (b *BidRepositoryImpl) ListByUserAll(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	limit int,
	offset int,
) ([]domain.MyBid, int, error) {
	const query = `
		select
			b.id,
			b.auction_id,
			b.user_id,
			b.amount,
			b.idempotency_key,
			b.created_at,
			a.lot,
			a.title,
			a.subtitle,
			a.status as auction_status,
			a.current_price,
			a.final_price,
			a.ends_at,
			a.ended_at,
			case
				when a.status = 'live' and a.highest_bidder_id = b.user_id then 'leading'
				when a.status = 'live' then 'outbid'
				when a.status = 'ended' and a.winner_id = b.user_id then 'won'
				else 'lost'
			end as bid_status
		from bids b
		join auctions a on a.id = b.auction_id
		where b.user_id = $1
		order by b.created_at desc, b.id desc
		limit $2 offset $3
	`

	rows, err := tx.Query(ctx, query, userID, limit, offset)
	items, err := scanMyBids(rows, err)
	if err != nil {
		return nil, 0, fmt.Errorf("list bids by user: %w", err)
	}

	var total int
	if err := tx.QueryRow(
		ctx,
		`select count(*) from bids where user_id = $1`,
		userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count bids by user: %w", err)
	}

	return items, total, nil
}

func scanMyBids(rows pgx.Rows, err error) ([]domain.MyBid, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.MyBid
	for rows.Next() {
		item, err := scanMyBid(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func scanMyBid(row pgx.Row) (domain.MyBid, error) {
	var item domain.MyBid
	var auctionStatus string
	var bidStatus string

	err := row.Scan(
		&item.ID,
		&item.AuctionID,
		&item.UserID,
		&item.Amount,
		&item.IdempotencyKey,
		&item.CreatedAt,
		&item.Auction.Lot,
		&item.Auction.Title,
		&item.Auction.Subtitle,
		&auctionStatus,
		&item.Auction.CurrentPrice,
		&item.Auction.FinalPrice,
		&item.Auction.EndsAt,
		&item.Auction.EndedAt,
		&bidStatus,
	)
	if err != nil {
		return domain.MyBid{}, err
	}

	item.Auction.ID = item.AuctionID
	item.Auction.Status = domain.AuctionStatus(auctionStatus)
	item.Status = domain.BidStatus(bidStatus)
	return item, nil
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
