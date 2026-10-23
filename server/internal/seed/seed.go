package seed

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	BidderEmail = "bidder@example.com"
	LiveLot     = "LOT-DEMO-LIVE"
	DraftLot    = "LOT-DEMO-DRAFT"
)

type Config struct {
	AdminEmail     string
	AdminPassword  string
	BidderPassword string
}

func Run(ctx context.Context, pool *pgxpool.Pool, cfg Config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := runTx(ctx, tx, cfg); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}

	return nil
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.AdminEmail) == "" ||
		cfg.AdminPassword == "" ||
		cfg.BidderPassword == "" {
		return errors.New("seed credentials are required")
	}
	return nil
}

func runTx(ctx context.Context, tx pgx.Tx, cfg Config) error {
	adminID, err := ensureUser(ctx, tx, cfg.AdminEmail, cfg.AdminPassword, "Admin Demo", "admin")
	if err != nil {
		return err
	}

	if _, err := ensureUser(ctx, tx, BidderEmail, cfg.BidderPassword, "Bidder Demo", "user"); err != nil {
		return err
	}

	now := time.Now().UTC()
	liveStartsAt := now
	liveEndsAt := now.Add(10 * time.Minute)

	var liveID uuid.UUID
	err = tx.QueryRow(ctx, `select id from auctions where status = 'live' limit 1`).Scan(&liveID)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := ensureAuction(
			ctx, tx, adminID, LiveLot, "Vintage Camera",
			"Kamera vintage untuk demo lelang live.",
			1_000_000, 100_000, "live", &liveStartsAt, &liveEndsAt,
		); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("find live auction: %w", err)
	}

	if _, err := ensureAuction(
		ctx, tx, adminID, DraftLot, "Mechanical Keyboard",
		"Keyboard mekanik untuk demo draft.",
		500_000, 50_000, "draft", nil, nil,
	); err != nil {
		return err
	}

	return nil
}

func ensureUser(
	ctx context.Context,
	tx pgx.Tx,
	email string,
	password string,
	displayName string,
	role string,
) (uuid.UUID, error) {
	var id uuid.UUID

	err := tx.QueryRow(
		ctx,
		`select id from users where lower(email) = lower($1)`,
		email,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("find seed user: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return uuid.Nil, fmt.Errorf("hash seed password: %w", err)
	}

	err = tx.QueryRow(
		ctx,
		`insert into users (email, password_hash, display_name, role)
		 values ($1, $2, $3, $4)
		 returning id`,
		strings.ToLower(strings.TrimSpace(email)),
		string(hash),
		displayName,
		role,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create seed user: %w", err)
	}

	return id, nil
}

func ensureAuction(
	ctx context.Context,
	tx pgx.Tx,
	createdBy uuid.UUID,
	lot string,
	title string,
	description string,
	startingPrice int64,
	minIncrement int64,
	status string,
	startsAt *time.Time,
	endsAt *time.Time,
) (uuid.UUID, error) {
	var id uuid.UUID

	err := tx.QueryRow(ctx, `select id from auctions where lot = $1`, lot).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("find seed auction: %w", err)
	}

	err = tx.QueryRow(
		ctx,
		`insert into auctions (
			created_by, lot, title, description, condition,
			starting_price, min_increment, duration_seconds,
			current_price, status, starts_at, ends_at,
			includes, image_urls, specs
		) values (
			$1, $2, $3, $4, 'Used',
			$5, $6, 600,
			$5, $7, $8, $9,
			'{}', '{}', '[]'::jsonb
		) returning id`,
		createdBy,
		lot,
		title,
		description,
		startingPrice,
		minIncrement,
		status,
		startsAt,
		endsAt,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create seed auction: %w", err)
	}

	return id, nil
}
