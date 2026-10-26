# Task 8: Auction Rules, Service, dan Endpoints

Status: panduan implementasi manual.

Task 8 menambahkan:

- aturan minimum bid dan anti-sniping
- read endpoint lelang
- admin create/start/end/delete draft
- result endpoint
- close due untuk scheduler

Task 8 belum menangani bid. Bid masuk Task 9.

## Prerequisite: AuctionRepository

Task 8 membutuhkan `AuctionRepository` dari Task 3. Kalau belum ada, buat dulu mengikuti panduan:

`docs/superpowers/plans/2026-10-22-auction-repository.md`

File yang dibuat:

```text
server/internal/repository/auction/
  auction_repository.go
  auction_repository_impl.go
  auction_repository_test.go
```

Interface minimal:

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

Query penting:

```sql
-- FindCurrentLive
select ... from auctions where status = 'live';

-- LockByID
select ... from auctions where id = $1 for update;

-- ListEnded
select ... from auctions
where status = 'ended'
order by ended_at desc
limit $1;

-- ListDue
select ... from auctions
where status = 'live' and ends_at <= $1;
```

`FindByID`, `FindCurrentLive`, dan `LockByID` harus memetakan `pgx.ErrNoRows` ke `exception.ErrAuctionNotFound`.

Kolom `specs` adalah JSONB. Saat insert, `json.Marshal` ke `[]byte`. Saat scan, `json.Unmarshal`. Normalisasi `nil` slice menjadi slice kosong karena kolom `includes` dan `image_urls` bertipe `NOT NULL`.

Interface di atas belum cukup untuk dikompilasi. Implementasi SQL `auction_repository_impl.go` harus dibuat lebih dulu. Task 8 hanya melengkapi rules, service, controller, dan route di atas repository tersebut.

---

## 1. Error tambahan

Di `internal/exception/errors.go`:

```go
ErrAuctionNotFound     = errors.New("auction not found")
ErrAuctionNotDraft     = errors.New("auction is not draft")
ErrAuctionNotLive      = errors.New("auction is not live")
ErrAuctionAlreadyLive  = errors.New("another auction is already live")
```

Di `internal/exception/error_handler.go`, tambahkan sebelum default:

```go
case errors.Is(err, ErrAuctionNotFound):
	return http.StatusNotFound, "AUCTION_NOT_FOUND", "Lelang tidak ditemukan"
case errors.Is(err, ErrAuctionNotDraft):
	return http.StatusConflict, "AUCTION_NOT_DRAFT", "Lelang bukan draft"
case errors.Is(err, ErrAuctionNotLive):
	return http.StatusConflict, "AUCTION_NOT_LIVE", "Lelang tidak sedang live"
case errors.Is(err, ErrAuctionAlreadyLive):
	return http.StatusConflict, "AUCTION_ALREADY_LIVE", "Masih ada lelang lain yang live"
```

---

## 2. Auction rules

Buat `internal/auction/rules.go`:

```go
package auction

import (
	"time"

	"live-auction/server/internal/model/domain"
)

func MinimumBid(a domain.Auction) int64 {
	if a.HighestBidderID == nil {
		return a.StartingPrice
	}
	return a.CurrentPrice + a.MinIncrement
}

func NextEndsAt(now, endsAt time.Time, extensions int) (time.Time, bool) {
	if extensions >= 5 {
		return endsAt, false
	}
	if !now.Before(endsAt) {
		return endsAt, false
	}
	if endsAt.Sub(now) <= 60*time.Second {
		return endsAt.Add(60 * time.Second), true
	}
	return endsAt, false
}
```

Arti:

- Bid pertama minimal sama dengan `starting_price`.
- Bid berikutnya minimal `current_price + min_increment`.
- Anti-sniping aktif saat sisa waktu 60 detik atau kurang.
- Maksimal 5 perpanjangan.

Test minimal:

```go
func TestMinimumBid(t *testing.T) {
	auction := domain.Auction{
		StartingPrice: 1000,
		MinIncrement:  100,
	}
	if got := MinimumBid(auction); got != 1000 {
		t.Fatalf("minimum = %d, want 1000", got)
	}

	bidder := uuid.New()
	auction.HighestBidderID = &bidder
	auction.CurrentPrice = 1500
	if got := MinimumBid(auction); got != 1600 {
		t.Fatalf("minimum = %d, want 1600", got)
	}
}
```

---

## 3. Request DTO

Buat `internal/model/web/auction_request/create_auction.go`:

```go
package auction_request

type SpecRequest struct {
	Label string `json:"label" validate:"required"`
	Value string `json:"value" validate:"required"`
}

type CreateAuctionRequest struct {
	Title           string        `json:"title" validate:"required,min=3,max=120"`
	Subtitle        string        `json:"subtitle" validate:"omitempty,max=120"`
	Description     string        `json:"description" validate:"required,min=10"`
	Condition       string        `json:"condition" validate:"required"`
	Includes        []string      `json:"includes" validate:"omitempty,dive,min=1"`
	Specs           []SpecRequest `json:"specs" validate:"omitempty,dive"`
	ImageURLs       []string      `json:"image_urls" validate:"omitempty,dive,url"`
	StartingPrice   int64         `json:"starting_price" validate:"required,gt=0"`
	MinIncrement    int64         `json:"min_increment" validate:"required,gt=0"`
	DurationMinutes int           `json:"duration_minutes" validate:"required,min=1,max=60"`
}
```

---

## 4. Response DTO

Buat `internal/model/web/auction_response/auction.go`:

```go
package auction_response

import (
	"time"

	"github.com/google/uuid"

	"live-auction/server/internal/model/domain"
)

type SpecResponse struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type AuctionResponse struct {
	ID              uuid.UUID          `json:"id"`
	CreatedBy       uuid.UUID          `json:"created_by"`
	Lot             string             `json:"lot"`
	Title           string             `json:"title"`
	Subtitle        string             `json:"subtitle"`
	Description     string             `json:"description"`
	Condition       string             `json:"condition"`
	Includes        []string           `json:"includes"`
	Specs           []SpecResponse     `json:"specs"`
	ImageURLs       []string           `json:"image_urls"`
	StartingPrice   int64              `json:"starting_price"`
	MinIncrement    int64              `json:"min_increment"`
	DurationSeconds int                `json:"duration_seconds"`
	CurrentPrice    int64              `json:"current_price"`
	HighestBidderID *uuid.UUID         `json:"highest_bidder_id,omitempty"`
	Status          domain.AuctionStatus `json:"status"`
	StartsAt        *time.Time         `json:"starts_at,omitempty"`
	EndsAt          *time.Time         `json:"ends_at,omitempty"`
	EndedAt         *time.Time         `json:"ended_at,omitempty"`
	ExtensionCount  int                `json:"extension_count"`
	Version         int64              `json:"version"`
	WinnerID        *uuid.UUID         `json:"winner_id,omitempty"`
	FinalPrice      *int64             `json:"final_price,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

type BidResponse struct {
	ID          uuid.UUID `json:"id"`
	AuctionID   uuid.UUID `json:"auction_id"`
	UserID      uuid.UUID `json:"user_id"`
	Amount      int64     `json:"amount"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type AuctionDetailResponse struct {
	Auction    AuctionResponse `json:"auction"`
	Bids       []BidResponse   `json:"bids"`
	ServerTime time.Time       `json:"server_time"`
}

type ResultResponse struct {
	Auction AuctionResponse `json:"auction"`
}
```

---

## 5. Auction service

Buat `internal/service/auction_service.go`.

Import yang dibutuhkan:

```go
import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/auction_request"
	"live-auction/server/internal/model/web/auction_response"
	"live-auction/server/internal/repository"
	auctionrepo "live-auction/server/internal/repository/auction"
	bidrepo "live-auction/server/internal/repository/bid"
)
```

### Interface

```go
type AuctionService interface {
	Current(ctx context.Context) (auction_response.AuctionDetailResponse, error)
	Detail(ctx context.Context, id uuid.UUID) (auction_response.AuctionDetailResponse, error)
	Bids(ctx context.Context, auctionID uuid.UUID, beforeID *uuid.UUID, limit int) ([]auction_response.BidResponse, error)
	Results(ctx context.Context, limit int) ([]auction_response.ResultResponse, error)
	List(ctx context.Context, limit int, offset int) ([]auction_response.AuctionResponse, error)
	CreateDraft(ctx context.Context, adminID uuid.UUID, req auction_request.CreateAuctionRequest) (auction_response.AuctionResponse, error)
	Start(ctx context.Context, adminID uuid.UUID, id uuid.UUID) (auction_response.AuctionResponse, error)
	End(ctx context.Context, adminID uuid.UUID, id uuid.UUID) (auction_response.AuctionResponse, error)
	DeleteDraft(ctx context.Context, adminID uuid.UUID, id uuid.UUID) error
	CloseDue(ctx context.Context) ([]domain.Auction, error)
}
```

### Struct

```go
type auctionService struct {
	auctions auctionrepo.AuctionRepository
	bids     bidrepo.BidRepository
	tx       repository.TxManager
	now      func() time.Time
	validate *validator.Validate
}

func NewAuctionService(
	auctions auctionrepo.AuctionRepository,
	bids bidrepo.BidRepository,
	tx repository.TxManager,
	now func() time.Time,
) AuctionService {
	if now == nil {
		now = time.Now
	}
	return &auctionService{
		auctions: auctions,
		bids:     bids,
		tx:       tx,
		now:      now,
		validate: validator.New(),
	}
}
```

### Current

```go
func (s *auctionService) Current(ctx context.Context) (auction_response.AuctionDetailResponse, error) {
	var result auction_response.AuctionDetailResponse

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		auction, err := s.auctions.FindCurrentLive(ctx, tx)
		if err != nil {
			return err
		}

		bids, err := s.bids.ListByAuction(ctx, tx, auction.ID, nil, 50)
		if err != nil {
			return err
		}

		result = auction_response.AuctionDetailResponse{
			Auction:    toAuctionResponse(auction),
			Bids:       toBidResponses(bids),
			ServerTime: s.now().UTC(),
		}
		return nil
	})

	return result, err
}
```

`AuctionController` akan mengubah `ErrAuctionNotFound` menjadi 204.

### Detail

```go
func (s *auctionService) Detail(
	ctx context.Context,
	id uuid.UUID,
) (auction_response.AuctionDetailResponse, error) {
	var result auction_response.AuctionDetailResponse

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		auction, err := s.auctions.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}

		bids, err := s.bids.ListByAuction(ctx, tx, id, nil, 50)
		if err != nil {
			return err
		}

		result = auction_response.AuctionDetailResponse{
			Auction:    toAuctionResponse(auction),
			Bids:       toBidResponses(bids),
			ServerTime: s.now().UTC(),
		}
		return nil
	})

	return result, err
}
```

### Bids

```go
func (s *auctionService) Bids(
	ctx context.Context,
	auctionID uuid.UUID,
	beforeID *uuid.UUID,
	limit int,
) ([]auction_response.BidResponse, error) {
	var result []auction_response.BidResponse

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		bids, err := s.bids.ListByAuction(ctx, tx, auctionID, beforeID, limit)
		if err != nil {
			return err
		}
		result = toBidResponses(bids)
		return nil
	})

	return result, err
}
```

### Results

```go
func (s *auctionService) Results(
	ctx context.Context,
	limit int,
) ([]auction_response.ResultResponse, error) {
	var result []auction_response.ResultResponse

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		auctions, err := s.auctions.ListEnded(ctx, tx, limit)
		if err != nil {
			return err
		}

		for _, auction := range auctions {
			result = append(result, auction_response.ResultResponse{
				Auction: toAuctionResponse(auction),
			})
		}
		return nil
	})

	return result, err
}
```

### List

```go
func (s *auctionService) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]auction_response.AuctionResponse, error) {
	var result []auction_response.AuctionResponse

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		auctions, err := s.auctions.List(ctx, tx, limit, offset)
		if err != nil {
			return err
		}
		for _, auction := range auctions {
			result = append(result, toAuctionResponse(auction))
		}
		return nil
	})

	return result, err
}
```

### CreateDraft

```go
func (s *auctionService) CreateDraft(
	ctx context.Context,
	adminID uuid.UUID,
	req auction_request.CreateAuctionRequest,
) (auction_response.AuctionResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return auction_response.AuctionResponse{}, exception.ErrValidation
	}

	includes := req.Includes
	if includes == nil {
		includes = []string{}
	}
	imageURLs := req.ImageURLs
	if imageURLs == nil {
		imageURLs = []string{}
	}
	specs := make([]domain.Spec, 0, len(req.Specs))
	for _, spec := range req.Specs {
		specs = append(specs, domain.Spec{Label: spec.Label, Value: spec.Value})
	}

	lot := "LOT-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:8])

	var result auction_response.AuctionResponse

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		auction, err := s.auctions.Create(ctx, tx, domain.Auction{
			CreatedBy:       adminID,
			Lot:             lot,
			Title:           req.Title,
			Subtitle:        req.Subtitle,
			Description:     req.Description,
			Condition:       req.Condition,
			Includes:        includes,
			Specs:           specs,
			ImageURLs:       imageURLs,
			StartingPrice:   req.StartingPrice,
			MinIncrement:    req.MinIncrement,
			DurationSeconds: req.DurationMinutes * 60,
			CurrentPrice:    req.StartingPrice,
			Status:          domain.AuctionStatusDraft,
		})
		if err != nil {
			return err
		}

		result = toAuctionResponse(auction)
		return nil
	})

	return result, err
}
```

### Start

```go
func (s *auctionService) Start(
	ctx context.Context,
	adminID uuid.UUID,
	id uuid.UUID,
) (auction_response.AuctionResponse, error) {
	var result auction_response.AuctionResponse

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		auction, err := s.auctions.LockByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if auction.Status != domain.AuctionStatusDraft {
			return exception.ErrAuctionNotDraft
		}

		if _, err := s.auctions.FindCurrentLive(ctx, tx); err == nil {
			return exception.ErrAuctionAlreadyLive
		} else if !errors.Is(err, exception.ErrAuctionNotFound) {
			return err
		}

		now := s.now().UTC()
		endsAt := now.Add(time.Duration(auction.DurationSeconds) * time.Second)

		auction.Status = domain.AuctionStatusLive
		auction.StartsAt = &now
		auction.EndsAt = &endsAt
		auction.Version++

		updated, err := s.auctions.Update(ctx, tx, auction)
		if err != nil {
			if isOneLiveViolation(err) {
				return exception.ErrAuctionAlreadyLive
			}
			return err
		}

		result = toAuctionResponse(updated)
		return nil
	})

	return result, err
}
```

`isOneLiveViolation`:

```go
func isOneLiveViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "one_live_auction"
}
```

### End

```go
func (s *auctionService) End(
	ctx context.Context,
	adminID uuid.UUID,
	id uuid.UUID,
) (auction_response.AuctionResponse, error) {
	var result auction_response.AuctionResponse

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		auction, err := s.auctions.LockByID(ctx, tx, id)
		if err != nil {
			return err
		}

		if auction.Status == domain.AuctionStatusEnded {
			result = toAuctionResponse(auction)
			return nil
		}
		if auction.Status != domain.AuctionStatusLive {
			return exception.ErrAuctionNotLive
		}

		now := s.now().UTC()
		auction.Status = domain.AuctionStatusEnded
		auction.EndedAt = &now
		auction.Version++

		if auction.HighestBidderID != nil {
			winnerID := *auction.HighestBidderID
			finalPrice := auction.CurrentPrice
			auction.WinnerID = &winnerID
			auction.FinalPrice = &finalPrice
		}

		updated, err := s.auctions.Update(ctx, tx, auction)
		if err != nil {
			return err
		}

		result = toAuctionResponse(updated)
		return nil
	})

	return result, err
}
```

End dengan status `ended` mengembalikan data yang sudah ada. Jadi idempoten.

### DeleteDraft

```go
func (s *auctionService) DeleteDraft(
	ctx context.Context,
	adminID uuid.UUID,
	id uuid.UUID,
) error {
	return s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		auction, err := s.auctions.LockByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if auction.Status != domain.AuctionStatusDraft {
			return exception.ErrAuctionNotDraft
		}
		return s.auctions.DeleteDraft(ctx, tx, id)
	})
}
```

### CloseDue

```go
func (s *auctionService) CloseDue(ctx context.Context) ([]domain.Auction, error) {
	var due []domain.Auction

	if err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		auctions, err := s.auctions.ListDue(ctx, tx, s.now().UTC())
		if err != nil {
			return err
		}
		due = auctions
		return nil
	}); err != nil {
		return nil, err
	}

	var ended []domain.Auction
	for _, auction := range due {
		updated, err := s.End(ctx, uuid.Nil, auction.ID)
		if err != nil {
			return nil, err
		}
		_ = updated
		ended = append(ended, auction)
	}

	return ended, nil
}
```

Versi sederhana ini memanggil `End` dalam transaksi terpisah. Untuk satu instance ini cukup.

### Mapper

```go
func toAuctionResponse(auction domain.Auction) auction_response.AuctionResponse {
	specs := make([]auction_response.SpecResponse, 0, len(auction.Specs))
	for _, spec := range auction.Specs {
		specs = append(specs, auction_response.SpecResponse{
			Label: spec.Label,
			Value: spec.Value,
		})
	}

	return auction_response.AuctionResponse{
		ID:              auction.ID,
		CreatedBy:       auction.CreatedBy,
		Lot:             auction.Lot,
		Title:           auction.Title,
		Subtitle:        auction.Subtitle,
		Description:     auction.Description,
		Condition:       auction.Condition,
		Includes:        auction.Includes,
		Specs:           specs,
		ImageURLs:       auction.ImageURLs,
		StartingPrice:   auction.StartingPrice,
		MinIncrement:    auction.MinIncrement,
		DurationSeconds: auction.DurationSeconds,
		CurrentPrice:    auction.CurrentPrice,
		HighestBidderID: auction.HighestBidderID,
		Status:          auction.Status,
		StartsAt:        auction.StartsAt,
		EndsAt:          auction.EndsAt,
		EndedAt:         auction.EndedAt,
		ExtensionCount:  auction.ExtensionCount,
		Version:         auction.Version,
		WinnerID:        auction.WinnerID,
		FinalPrice:      auction.FinalPrice,
		CreatedAt:       auction.CreatedAt,
	}
}

func toBidResponses(bids []domain.BidView) []auction_response.BidResponse {
	result := make([]auction_response.BidResponse, 0, len(bids))
	for _, bid := range bids {
		result = append(result, auction_response.BidResponse{
			ID:          bid.ID,
			AuctionID:   bid.AuctionID,
			UserID:      bid.UserID,
			Amount:      bid.Amount,
			DisplayName: bid.DisplayName,
			CreatedAt:   bid.CreatedAt,
		})
	}
	return result
}
```

---

## 6. Public auction controller

Buat `internal/controller/auction_controller.go`:

```go
package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/service"
)

type AuctionController struct {
	auctionService service.AuctionService
}

func NewAuctionController(auctionService service.AuctionService) *AuctionController {
	return &AuctionController{auctionService: auctionService}
}

func (c *AuctionController) Current(w http.ResponseWriter, r *http.Request) {
	result, err := c.auctionService.Current(r.Context())
	if errors.Is(err, exception.ErrAuctionNotFound) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		exception.WriteError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, result)
}

func (c *AuctionController) Detail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		exception.WriteError(w, exception.ErrValidation)
		return
	}

	result, err := c.auctionService.Detail(r.Context(), id)
	if err != nil {
		exception.WriteError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, result)
}

func (c *AuctionController) Bids(w http.ResponseWriter, r *http.Request) {
	auctionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		exception.WriteError(w, exception.ErrValidation)
		return
	}

	var beforeID *uuid.UUID
	if raw := r.URL.Query().Get("before_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			exception.WriteError(w, exception.ErrValidation)
			return
		}
		beforeID = &id
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			exception.WriteError(w, exception.ErrValidation)
			return
		}
		limit = parsed
	}

	result, err := c.auctionService.Bids(r.Context(), auctionID, beforeID, limit)
	if err != nil {
		exception.WriteError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, result)
}

func (c *AuctionController) Results(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			exception.WriteError(w, exception.ErrValidation)
			return
		}
		limit = parsed
	}

	result, err := c.auctionService.Results(r.Context(), limit)
	if err != nil {
		exception.WriteError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, result)
}
```

---

## 7. Admin auction controller

Buat `internal/controller/admin_controller.go`:

```go
package controller

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/model/web/auction_request"
	"live-auction/server/internal/service"
)

type AdminController struct {
	auctionService service.AuctionService
}

func NewAdminController(auctionService service.AuctionService) *AdminController {
	return &AdminController{auctionService: auctionService}
}

func (c *AdminController) List(w http.ResponseWriter, r *http.Request) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			exception.WriteError(w, exception.ErrValidation)
			return
		}
		page = parsed
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			exception.WriteError(w, exception.ErrValidation)
			return
		}
		limit = parsed
	}

	offset := (page - 1) * limit
	result, err := c.auctionService.List(r.Context(), limit, offset)
	if err != nil {
		exception.WriteError(w, err)
		return
	}
	helper.WriteJSON(w, http.StatusOK, result)
}

func (c *AdminController) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		exception.WriteError(w, exception.ErrUnauthenticated)
		return
	}

	var req auction_request.CreateAuctionRequest
	if err := helper.ReadJSON(w, r, &req); err != nil {
		exception.WriteError(w, exception.ErrValidation)
		return
	}

	result, err := c.auctionService.CreateDraft(r.Context(), principal.UserID, req)
	if err != nil {
		exception.WriteError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusCreated, result)
}

func (c *AdminController) Start(w http.ResponseWriter, r *http.Request) {
	c.changeStatus(w, r, "start")
}

func (c *AdminController) End(w http.ResponseWriter, r *http.Request) {
	c.changeStatus(w, r, "end")
}

func (c *AdminController) changeStatus(w http.ResponseWriter, r *http.Request, action string) {
	principal, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		exception.WriteError(w, exception.ErrUnauthenticated)
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		exception.WriteError(w, exception.ErrValidation)
		return
	}

	var result any
	if action == "start" {
		result, err = c.auctionService.Start(r.Context(), principal.UserID, id)
	} else {
		result, err = c.auctionService.End(r.Context(), principal.UserID, id)
	}
	if err != nil {
		exception.WriteError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, result)
}

func (c *AdminController) Delete(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		exception.WriteError(w, exception.ErrUnauthenticated)
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		exception.WriteError(w, exception.ErrValidation)
		return
	}

	if err := c.auctionService.DeleteDraft(r.Context(), principal.UserID, id); err != nil {
		exception.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

---

## 8. Router

Tambahkan field baru di `app.RouterDeps`:

```go
Auction *controller.AuctionController
Admin   *controller.AdminController
```

Daftarkan route:

```go
if deps.Auction != nil {
	mux.HandleFunc("GET /api/auctions/current", deps.Auction.Current)
	mux.HandleFunc("GET /api/auctions/{id}", deps.Auction.Detail)
	mux.HandleFunc("GET /api/auctions/{id}/bids", deps.Auction.Bids)
	mux.HandleFunc("GET /api/results", deps.Auction.Results)
}

if deps.Auth != nil && deps.Admin != nil {
	adminOnly := func(h http.Handler) http.Handler {
		return middleware.Auth(
			deps.AccessTokens,
			middleware.Admin(h),
		)
	}
	adminStateChange := func(h http.Handler) http.Handler {
		return middleware.Origin(
			deps.AllowedOrigin,
			adminOnly(h),
		)
	}

	mux.Handle("GET /api/admin/auctions", adminOnly(http.HandlerFunc(deps.Admin.List)))
	mux.Handle("POST /api/admin/auctions", adminStateChange(http.HandlerFunc(deps.Admin.Create)))
	mux.Handle("POST /api/admin/auctions/{id}/start", adminStateChange(http.HandlerFunc(deps.Admin.Start)))
	mux.Handle("POST /api/admin/auctions/{id}/end", adminStateChange(http.HandlerFunc(deps.Admin.End)))
	mux.Handle("DELETE /api/admin/auctions/{id}", adminStateChange(http.HandlerFunc(deps.Admin.Delete)))
}
```

Jangan memakai early return `if deps.Auth == nil { return mux }` sebelum route auction. Kalau early return, public auction tidak terdaftar.

---

## 9. Wiring main

Tambahkan:

```go
bidRepository := bidrepo.NewBidRepositoryImpl()
auctionRepository := auctionrepo.NewAuctionRepositoryImpl()

auctionService := service.NewAuctionService(
	auctionRepository,
	bidRepository,
	txManager,
	time.Now,
)

auctionController := controller.NewAuctionController(auctionService)
adminController := controller.NewAdminController(auctionService)
```

Lalu isi `RouterDeps`:

```go
Auction: auctionController,
Admin:   adminController,
```

Import:

```go
auctionrepo "live-auction/server/internal/repository/auction"
bidrepo "live-auction/server/internal/repository/bid"
```

---

## 10. Test yang perlu dibuat

### Rules

- bid pertama minimal `starting_price`
- bid berikutnya `current_price + min_increment`
- anti-sniping aktif pada sisa 60 detik
- tidak aktif di luar 60 detik
- maksimal 5 perpanjangan
- boundary tepat 60 detik

### Service

- create draft menghasilkan status draft, current price = starting price, dan lot unik
- start draft mengisi `starts_at`, `ends_at`, status live, dan version naik
- start kedua saat ada live menghasilkan `ErrAuctionAlreadyLive`
- start non-draft menghasilkan `ErrAuctionNotDraft`
- end live mengisi winner, final price, ended_at, dan version naik
- end kedua idempoten
- end tanpa bid menghasilkan winner nil dan final price nil
- delete hanya draft
- current mengembalikan live atau `ErrAuctionNotFound`
- detail mengembalikan auction dan bid terbaru
- detail ID tidak ada menghasilkan `ErrAuctionNotFound`
- results hanya ended dan urut `ended_at desc`
- close due hanya menutup live dengan `ends_at <= now`

### Controller

- current 200 saat live
- current 204 saat tidak ada live
- detail 200
- detail UUID invalid 422
- bids 200 dan pagination
- results 200
- admin list 200 dengan `page` dan `limit`
- admin list page atau limit invalid menghasilkan 422
- admin create 201
- admin start 200
- admin end 200
- admin delete 204
- admin tanpa role admin 403
- Origin salah 403

---

## 11. Verifikasi

```powershell
cd D:\live-auction\server

go mod tidy
go test ./...
go build ./...
go vet ./...
```

Dengan test database:

```powershell
$env:TEST_DATABASE_URL="postgres://postgres:root@localhost:5432/live_auction_test?sslmode=disable"
go test ./...
```

## Checklist selesai

- [ ] error auction ditambahkan
- [ ] error handler memetakan error auction
- [ ] `auction/rules.go` dibuat
- [ ] `auction_request` dan `auction_response` dibuat
- [ ] `AuctionService` dibuat
- [ ] `AuctionController` dibuat
- [ ] `AdminController` dibuat
- [ ] `RouterDeps` ditambah `Auction` dan `Admin`
- [ ] route public dan admin terdaftar
- [ ] main wiring auction service/controller
- [ ] test rules, service, controller
- [ ] `go test ./...` lulus
- [ ] `go build ./...` lulus
- [ ] `go vet ./...` lulus
