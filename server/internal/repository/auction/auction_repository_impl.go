package auction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const auctionColumns = `
	id,
	created_by,
	lot,
	title,
	subtitle,
	description,
	condition,
	includes,
	specs,
	image_urls,
	starting_price,
	min_increment,
	duration_seconds,
	current_price,
	highest_bidder_id,
	status,
	starts_at,
	ends_at,
	ended_at,
	extension_count,
	version,
	winner_id,
	final_price,
	created_at
`

type AuctionRepositoryImpl struct{}

// Create implements [AuctionRepository].
func (a *AuctionRepositoryImpl) Create(ctx context.Context, tx pgx.Tx, auction domain.Auction) (domain.Auction, error) {
	specs, err := marshalSpecs(auction.Specs)

	if err != nil {
		return domain.Auction{}, err
	}

	const query = `
		insert into auctions (
			created_by,
			lot,
			title,
			subtitle,
			description,
			condition,
			includes,
			specs,
			image_urls,
			starting_price,
			min_increment,
			duration_seconds,
			current_price,
			highest_bidder_id,
			status,
			starts_at,
			ends_at,
			ended_at,
			extension_count,
			version,
			winner_id,
			final_price
		)
		values (
			$1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9,
			$10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22
		)
		returning ` + auctionColumns

	created, err := scanAuction(tx.QueryRow(
		ctx,
		query,
		auction.CreatedBy,
		auction.Lot,
		auction.Title,
		auction.Subtitle,
		auction.Description,
		auction.Condition,
		emptyStrings(auction.Includes),
		specs,
		emptyStrings(auction.ImageURLs),
		auction.StartingPrice,
		auction.MinIncrement,
		auction.DurationSeconds,
		auction.CurrentPrice,
		auction.HighestBidderID,
		string(auction.Status),
		auction.StartsAt,
		auction.EndsAt,
		auction.EndedAt,
		auction.ExtensionCount,
		auction.Version,
		auction.WinnerID,
		auction.FinalPrice))

	if err != nil {
		return domain.Auction{}, fmt.Errorf("create auction: %w", err)
	}

	return created, err
}

// DeleteDraft implements [AuctionRepository].
func (a *AuctionRepositoryImpl) DeleteDraft(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	const query = `
		delete from auctions
		where id = $1 and status = 'draft'
	`

	tag, err := tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete draft auction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return exception.ErrAuctionNotFound
	}

	return nil
}

// FindByID implements [AuctionRepository].
func (a *AuctionRepositoryImpl) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (domain.Auction, error) {
	const query = `select ` + auctionColumns + ` from auctions where id = $1`

	auction, err := scanAuction(tx.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Auction{}, exception.ErrAuctionNotFound
	}
	if err != nil {
		return domain.Auction{}, fmt.Errorf("find auction by id: %w", err)
	}

	return auction, nil
}

// FindCurrentLive implements [AuctionRepository].
func (a *AuctionRepositoryImpl) FindCurrentLive(ctx context.Context, tx pgx.Tx) (domain.Auction, error) {
	const query = `select ` + auctionColumns + ` from auctions where status = 'live'`

	auction, err := scanAuction(tx.QueryRow(ctx, query))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Auction{}, exception.ErrAuctionNotFound
	}
	if err != nil {
		return domain.Auction{}, fmt.Errorf("find current live auction: %w", err)
	}

	return auction, nil
}

// List implements [AuctionRepository].
func (a *AuctionRepositoryImpl) List(ctx context.Context, tx pgx.Tx, limit int, offset int) ([]domain.Auction, error) {
	const query = `
			select ` + auctionColumns + `
			from auctions
			order by created_at desc, id desc
			limit $1 offset $2
		`

	return scanAuctions(tx.Query(ctx, query, limit, offset))
}

// ListDue implements [AuctionRepository].
func (a *AuctionRepositoryImpl) ListDue(ctx context.Context, tx pgx.Tx, now time.Time) ([]domain.Auction, error) {
	const query = `
		select ` + auctionColumns + `
		from auctions
		where status = 'live' and ends_at <= $1
		order by ends_at
	`

	return scanAuctions(tx.Query(ctx, query, now))
}

// ListEnded implements [AuctionRepository].
func (r *AuctionRepositoryImpl) ListEnded(
	ctx context.Context,
	tx pgx.Tx,
	limit int,
) ([]domain.Auction, error) {
	const query = `
		select ` + auctionColumns + `
		from auctions
		where status = 'ended'
		order by ended_at desc
		limit $1
	`

	return scanAuctions(tx.Query(ctx, query, limit))
}

// LockByID implements [AuctionRepository].
func (a *AuctionRepositoryImpl) LockByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (domain.Auction, error) {
	const query = `
		select ` + auctionColumns + `
		from auctions
		where id = $1
		for update
	`

	auction, err := scanAuction(tx.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Auction{}, exception.ErrAuctionNotFound
	}
	if err != nil {
		return domain.Auction{}, fmt.Errorf("lock auction by id: %w", err)
	}

	return auction, nil
}

// Update implements [AuctionRepository].
func (a *AuctionRepositoryImpl) Update(ctx context.Context, tx pgx.Tx, auction domain.Auction) (domain.Auction, error) {
	specs, err := marshalSpecs(auction.Specs)
	if err != nil {
		return domain.Auction{}, err
	}

	const query = `
		update auctions set
			lot = $2,
			title = $3,
			subtitle = $4,
			description = $5,
			condition = $6,
			includes = $7,
			specs = $8::jsonb,
			image_urls = $9,
			starting_price = $10,
			min_increment = $11,
			duration_seconds = $12,
			current_price = $13,
			highest_bidder_id = $14,
			status = $15,
			starts_at = $16,
			ends_at = $17,
			ended_at = $18,
			extension_count = $19,
			version = $20,
			winner_id = $21,
			final_price = $22
		where id = $1
		returning ` + auctionColumns

	updated, err := scanAuction(tx.QueryRow(
		ctx,
		query,
		auction.ID,
		auction.Lot,
		auction.Title,
		auction.Subtitle,
		auction.Description,
		auction.Condition,
		emptyStrings(auction.Includes),
		specs,
		emptyStrings(auction.ImageURLs),
		auction.StartingPrice,
		auction.MinIncrement,
		auction.DurationSeconds,
		auction.CurrentPrice,
		auction.HighestBidderID,
		string(auction.Status),
		auction.StartsAt,
		auction.EndsAt,
		auction.EndedAt,
		auction.ExtensionCount,
		auction.Version,
		auction.WinnerID,
		auction.FinalPrice,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Auction{}, exception.ErrAuctionNotFound
	}
	if err != nil {
		return domain.Auction{}, fmt.Errorf("update auction: %w", err)
	}

	return updated, nil
}

func NewAuctionRepositoryImpl() *AuctionRepositoryImpl {
	return &AuctionRepositoryImpl{}
}

var _ AuctionRepository = (*AuctionRepositoryImpl)(nil)

func scanAuction(row pgx.Row) (domain.Auction, error) {
	var auction domain.Auction
	var rawSpecs []byte
	var status string

	err := row.Scan(
		&auction.ID,
		&auction.CreatedBy,
		&auction.Lot,
		&auction.Title,
		&auction.Subtitle,
		&auction.Description,
		&auction.Condition,
		&auction.Includes,
		&rawSpecs,
		&auction.ImageURLs,
		&auction.StartingPrice,
		&auction.MinIncrement,
		&auction.DurationSeconds,
		&auction.CurrentPrice,
		&auction.HighestBidderID,
		&status,
		&auction.StartsAt,
		&auction.EndsAt,
		&auction.EndedAt,
		&auction.ExtensionCount,
		&auction.Version,
		&auction.WinnerID,
		&auction.FinalPrice,
		&auction.CreatedAt,
	)
	if err != nil {
		return domain.Auction{}, err
	}

	if len(rawSpecs) > 0 {
		if err := json.Unmarshal(rawSpecs, &auction.Specs); err != nil {
			return domain.Auction{}, fmt.Errorf("unmarshal auction specs: %w", err)
		}
	}

	auction.Status = domain.AuctionStatus(status)
	return auction, nil
}

func scanAuctions(rows pgx.Rows, err error) ([]domain.Auction, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var auctions []domain.Auction
	for rows.Next() {
		auction, err := scanAuction(rows)
		if err != nil {
			return nil, err
		}
		auctions = append(auctions, auction)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return auctions, nil
}

func marshalSpecs(specs []domain.Spec) (string, error) {
	if specs == nil {
		specs = []domain.Spec{}
	}
	raw, err := json.Marshal(specs)
	if err != nil {
		return "", fmt.Errorf("marshal auction specs: %w", err)
	}
	return string(raw), nil
}

func emptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
