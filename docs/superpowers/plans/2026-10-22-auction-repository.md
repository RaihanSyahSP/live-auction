# Task 3 Lanjutan: Auction Repository

Status: panduan implementasi manual.

File ini melengkapi bagian Task 3 yang terlewat:

```text
server/internal/repository/auction/
  auction_repository.go
  auction_repository_impl.go
  auction_repository_test.go
```

Task 8 membutuhkan repository ini.

## 1. Error tambahan

Di `internal/exception/errors.go`:

```go
ErrAuctionNotFound = errors.New("auction not found")
```

Kalau belum ada error auction lain, tambahkan juga:

```go
ErrAuctionNotDraft    = errors.New("auction is not draft")
ErrAuctionNotLive     = errors.New("auction is not live")
ErrAuctionAlreadyLive = errors.New("another auction is already live")
```

## 2. Interface

Buat `internal/repository/auction/auction_repository.go`:

```go
package auction

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"live-auction/server/internal/model/domain"
)

type AuctionRepository interface {
	Create(ctx context.Context, tx pgx.Tx, auction domain.Auction) (domain.Auction, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (domain.Auction, error)
	FindCurrentLive(ctx context.Context, tx pgx.Tx) (domain.Auction, error)
	List(ctx context.Context, tx pgx.Tx, limit int, offset int) ([]domain.Auction, error)
	ListEnded(ctx context.Context, tx pgx.Tx, limit int) ([]domain.Auction, error)
	LockByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (domain.Auction, error)
	Update(ctx context.Context, tx pgx.Tx, auction domain.Auction) (domain.Auction, error)
	DeleteDraft(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	ListDue(ctx context.Context, tx pgx.Tx, now time.Time) ([]domain.Auction, error)
}
```

## 3. Implementasi

Buat `internal/repository/auction/auction_repository_impl.go`:

```go
package auction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"
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

func NewAuctionRepositoryImpl() *AuctionRepositoryImpl {
	return &AuctionRepositoryImpl{}
}

var _ AuctionRepository = (*AuctionRepositoryImpl)(nil)

func (r *AuctionRepositoryImpl) Create(
	ctx context.Context,
	tx pgx.Tx,
	auction domain.Auction,
) (domain.Auction, error) {
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
		auction.FinalPrice,
	))
	if err != nil {
		return domain.Auction{}, fmt.Errorf("create auction: %w", err)
	}

	return created, nil
}

func (r *AuctionRepositoryImpl) FindByID(
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
) (domain.Auction, error) {
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

func (r *AuctionRepositoryImpl) FindCurrentLive(
	ctx context.Context,
	tx pgx.Tx,
) (domain.Auction, error) {
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

func (r *AuctionRepositoryImpl) List(
	ctx context.Context,
	tx pgx.Tx,
	limit int,
	offset int,
) ([]domain.Auction, error) {
	const query = `
		select ` + auctionColumns + `
		from auctions
		order by created_at desc, id desc
		limit $1 offset $2
	`

	return scanAuctions(tx.Query(ctx, query, limit, offset))
}

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

func (r *AuctionRepositoryImpl) LockByID(
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
) (domain.Auction, error) {
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

func (r *AuctionRepositoryImpl) Update(
	ctx context.Context,
	tx pgx.Tx,
	auction domain.Auction,
) (domain.Auction, error) {
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

func (r *AuctionRepositoryImpl) DeleteDraft(
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
) error {
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

func (r *AuctionRepositoryImpl) ListDue(
	ctx context.Context,
	tx pgx.Tx,
	now time.Time,
) ([]domain.Auction, error) {
	const query = `
		select ` + auctionColumns + `
		from auctions
		where status = 'live' and ends_at <= $1
		order by ends_at
	`

	return scanAuctions(tx.Query(ctx, query, now))
}

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
```

## 4. Test repository

Buat `internal/repository/auction/auction_repository_test.go`.

Gunakan helper test yang sudah ada:

```go
package auction_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/model/domain"
	auctionrepo "live-auction/server/internal/repository/auction"
)

func newAuctionRepo(t *testing.T) (*auctionrepo.AuctionRepositoryImpl, pgx.Tx) {
	t.Helper()

	pool := helper.OpenTestDB(t)
	tx := helper.BeginTestTx(t, pool)
	return auctionrepo.NewAuctionRepositoryImpl(), tx
}

func createAuctionUser(t *testing.T, tx pgx.Tx) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := tx.QueryRow(
		context.Background(),
		`insert into users (email, password_hash, display_name, role)
		 values ($1, 'hash', 'Auction User', 'user')
		 returning id`,
		uuid.NewString()+"@example.com",
	).Scan(&id)
	if err != nil {
		t.Fatalf("create user fixture: %v", err)
	}
	return id
}
```

Test create dan find:

```go
func TestAuctionRepositoryCreateAndFind(t *testing.T) {
	repo, tx := newAuctionRepo(t)
	ctx := context.Background()
	userID := createAuctionUser(t, tx)

	created, err := repo.Create(ctx, tx, domain.Auction{
		CreatedBy:       userID,
		Lot:             "LOT-" + uuid.NewString()[:8],
		Title:           "Fujifilm X-T2",
		Description:     "Mirrorless APS-C",
		Condition:       "Used",
		Includes:        []string{"Body", "Charger"},
		Specs:           []domain.Spec{{Label: "Sensor", Value: "24 MP"}},
		ImageURLs:       []string{},
		StartingPrice:   5_000_000,
		MinIncrement:    100_000,
		DurationSeconds: 600,
		CurrentPrice:    5_000_000,
		Status:          domain.AuctionStatusDraft,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(ctx, tx, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %v, want %v", found.ID, created.ID)
	}
	if len(found.Specs) != 1 || found.Specs[0].Label != "Sensor" {
		t.Errorf("Specs = %+v", found.Specs)
	}
}
```

Test not found:

```go
func TestAuctionRepositoryFindByIDNotFound(t *testing.T) {
	repo, tx := newAuctionRepo(t)

	_, err := repo.FindByID(context.Background(), tx, uuid.New())
	if !errors.Is(err, exception.ErrAuctionNotFound) {
		t.Fatalf("FindByID() error = %v, want %v", err, exception.ErrAuctionNotFound)
	}
}
```

Test due:

```go
func TestAuctionRepositoryListDue(t *testing.T) {
	repo, tx := newAuctionRepo(t)
	ctx := context.Background()
	userID := createAuctionUser(t, tx)
	now := time.Now().UTC()
	endsAt := now.Add(-time.Minute)

	_, err := repo.Create(ctx, tx, domain.Auction{
		CreatedBy:       userID,
		Lot:             "LOT-" + uuid.NewString()[:8],
		Title:           "Due auction",
		Description:     "Test",
		Condition:       "Used",
		StartingPrice:   1000,
		MinIncrement:    100,
		DurationSeconds: 60,
		CurrentPrice:    1000,
		Status:          domain.AuctionStatusLive,
		StartsAt:        ptrTime(now.Add(-time.Hour)),
		EndsAt:          &endsAt,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	due, err := repo.ListDue(ctx, tx, now)
	if err != nil {
		t.Fatalf("ListDue() error = %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("ListDue() length = %d, want 1", len(due))
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
```

Test tambahan yang perlu dibuat:

- `FindCurrentLive` saat ada live
- `FindCurrentLive` saat tidak ada live
- `List` urut `created_at desc`
- `ListEnded` hanya ended
- `LockByID` mengembalikan row
- `Update` menyimpan status, harga, winner, final price
- `DeleteDraft` hanya menghapus draft
- `DeleteDraft` pada live mengembalikan error

## 5. Verifikasi

```powershell
cd D:\live-auction\server
go test ./internal/repository/auction -v
go test ./...
go build ./...
go vet ./...
```

Dengan test database:

```powershell
$env:TEST_DATABASE_URL="postgres://postgres:root@localhost:5432/live_auction_test?sslmode=disable"
go test ./internal/repository/auction -v
```

## Checklist

- [ ] `ErrAuctionNotFound` ditambahkan
- [ ] interface `AuctionRepository` dibuat
- [ ] implementasi SQL dibuat
- [ ] `specs` JSONB memakai marshal dan unmarshal
- [ ] `includes` dan `image_urls` dinormalisasi menjadi slice kosong
- [ ] `pgx.ErrNoRows` dipetakan ke `ErrAuctionNotFound`
- [ ] test create/find
- [ ] test not found
- [ ] test current live
- [ ] test list dengan limit dan offset
- [ ] test list ended
- [ ] test lock
- [ ] test update
- [ ] test delete draft
- [ ] test list due
- [ ] `go test ./...` lulus
- [ ] `go build ./...` lulus
- [ ] `go vet ./...` lulus
