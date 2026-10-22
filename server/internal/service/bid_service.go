package service

import (
	"context"
	"errors"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/auction_response"
	"time"

	auctionrules "live-auction/server/internal/auction"

	"live-auction/server/internal/model/web/bid_request"
	"live-auction/server/internal/model/web/bid_response"
	"live-auction/server/internal/repository"
	auctionrepo "live-auction/server/internal/repository/auction"
	bidrepo "live-auction/server/internal/repository/bid"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type BidService interface {
	PlaceBid(
		ctx context.Context,
		userID uuid.UUID,
		auctionID uuid.UUID,
		req bid_request.PlaceBidRequest,
	) (bid_response.PlaceBidResponse, error)
}

type bidService struct {
	auctions auctionrepo.AuctionRepository
	bids     bidrepo.BidRepository
	tx       repository.TxManager
	now      func() time.Time
}

// PlaceBid implements [BidService].
func (s *bidService) PlaceBid(ctx context.Context, userID uuid.UUID, auctionID uuid.UUID, req bid_request.PlaceBidRequest) (bid_response.PlaceBidResponse, error) {
	const maxAmount int64 = 1_000_000_000_000

	if req.Amount <= 0 || req.Amount > maxAmount {
		return bid_response.PlaceBidResponse{}, exception.ErrInvalidAmount
	}

	var result bid_response.PlaceBidResponse

	err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		auction, err := s.auctions.LockByID(ctx, tx, auctionID)
		if err != nil {
			return err
		}

		now := s.now().UTC()

		existing, err := s.bids.FindByIdempotencyKey(
			ctx,
			tx,
			auctionID,
			userID,
			req.IdempotencyKey,
		)

		if err == nil {
			if existing.Amount != req.Amount {
				return exception.ErrIdempotencyConflict
			}

			result = bid_response.PlaceBidResponse{
				Auction: toAuctionResponse(auction),
				Bid:     toBidResponse(existing),
				Replay:  true,
			}
			return nil
		}

		if !errors.Is(err, exception.ErrBidNotFound) {
			return err
		}

		if auction.Status != domain.AuctionStatusLive {
			return exception.ErrAuctionNotLive
		}
		if auction.EndsAt == nil || !now.Before(*auction.EndsAt) {
			return exception.ErrAuctionNotLive
		}
		if auction.CreatedBy == userID {
			return exception.ErrCreatorCannotBid
		}
		if auction.HighestBidderID != nil && *auction.HighestBidderID == userID {
			return exception.ErrAlreadyHighest
		}

		minimum := auctionrules.MinimumBid(auction)
		if req.Amount < minimum {
			return exception.ErrBidTooLow
		}

		previousHighest := auction.HighestBidderID
		extended := false

		newEndsAt, shouldExtend := auctionrules.NextEndsAt(
			now,
			*auction.EndsAt,
			auction.ExtensionCount,
		)
		if shouldExtend {
			auction.EndsAt = &newEndsAt
			auction.ExtensionCount++
			extended = true
		}

		auction.CurrentPrice = req.Amount
		auction.HighestBidderID = &userID
		auction.Version++

		inserted, err := s.bids.Insert(ctx, tx, domain.Bid{
			AuctionID:      auctionID,
			UserID:         userID,
			Amount:         req.Amount,
			IdempotencyKey: req.IdempotencyKey,
		})
		if errors.Is(err, exception.ErrBidAlreadyExists) {
			// Row lock biasanya sudah mencegah ini. Ini hanya safety net.
			return exception.ErrIdempotencyConflict
		}
		if err != nil {
			return err
		}

		updated, err := s.auctions.Update(ctx, tx, auction)
		if err != nil {
			return err
		}

		result = bid_response.PlaceBidResponse{
			Auction:               toAuctionResponse(updated),
			Bid:                   toBidResponse(inserted),
			Replay:                false,
			Extended:              extended,
			PreviousHighestUserID: previousHighest,
		}
		return nil
	})

	return result, err
}

func NewBidService(
	auctions auctionrepo.AuctionRepository,
	bids bidrepo.BidRepository,
	tx repository.TxManager,
	now func() time.Time,
) BidService {
	if now == nil {
		now = time.Now
	}
	return &bidService{
		auctions: auctions,
		bids:     bids,
		tx:       tx,
		now:      now,
	}
}

func toBidResponse(bid domain.Bid) auction_response.BidResponse {
	return auction_response.BidResponse{
		ID:        bid.ID,
		AuctionID: bid.AuctionID,
		UserID:    bid.UserID,
		Amount:    bid.Amount,
		CreatedAt: bid.CreatedAt,
	}
}
