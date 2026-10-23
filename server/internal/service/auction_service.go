package service

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
	CloseDue(ctx context.Context) ([]auction_response.AuctionResponse, error)
}

type auctionService struct {
	auctions auctionrepo.AuctionRepository
	bids     bidrepo.BidRepository
	tx       repository.TxManager
	now      func() time.Time
	validate *validator.Validate
}

// Bids implements [AuctionService].
func (s *auctionService) Bids(ctx context.Context, auctionID uuid.UUID, beforeID *uuid.UUID, limit int) ([]auction_response.BidResponse, error) {
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

// CloseDue implements [AuctionService].
func (s *auctionService) CloseDue(ctx context.Context) ([]auction_response.AuctionResponse, error) {
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

	var ended []auction_response.AuctionResponse
	for _, auction := range due {
		updated, err := s.End(ctx, uuid.Nil, auction.ID)
		if err != nil {
			return nil, err
		}
		ended = append(ended, updated)
	}

	return ended, nil
}

// CreateDraft implements [AuctionService].
func (s *auctionService) CreateDraft(ctx context.Context, adminID uuid.UUID, req auction_request.CreateAuctionRequest) (auction_response.AuctionResponse, error) {
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

// Current implements [AuctionService].
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

// DeleteDraft implements [AuctionService].
func (s *auctionService) DeleteDraft(ctx context.Context, adminID uuid.UUID, id uuid.UUID) error {
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

// Detail implements [AuctionService].
func (s *auctionService) Detail(ctx context.Context, id uuid.UUID) (auction_response.AuctionDetailResponse, error) {
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

// End implements [AuctionService].
func (s *auctionService) End(ctx context.Context, adminID uuid.UUID, id uuid.UUID) (auction_response.AuctionResponse, error) {
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

// List implements [AuctionService].
func (s *auctionService) List(ctx context.Context, limit int, offset int) ([]auction_response.AuctionResponse, error) {
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

// Results implements [AuctionService].
func (s *auctionService) Results(ctx context.Context, limit int) ([]auction_response.ResultResponse, error) {
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

// Start implements [AuctionService].
func (s *auctionService) Start(ctx context.Context, adminID uuid.UUID, id uuid.UUID) (auction_response.AuctionResponse, error) {
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

func isOneLiveViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "one_live_auction"
}

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
