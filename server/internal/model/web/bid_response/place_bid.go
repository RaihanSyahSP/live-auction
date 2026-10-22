package bid_response

import (
	"github.com/google/uuid"

	"live-auction/server/internal/model/web/auction_response"
)

type PlaceBidResponse struct {
	Auction               auction_response.AuctionResponse `json:"auction"`
	Bid                   auction_response.BidResponse     `json:"bid"`
	Replay                bool                             `json:"replay"`
	Extended              bool                             `json:"extended"`
	PreviousHighestUserID *uuid.UUID                       `json:"previous_highest_user_id,omitempty"`
}
