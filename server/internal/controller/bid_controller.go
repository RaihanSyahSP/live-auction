package controller

import (
	"net/http"

	"github.com/google/uuid"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/model/web/bid_request"
	"live-auction/server/internal/service"
)

type BidController struct {
	bidService service.BidService
}

func NewBidController(bidService service.BidService) *BidController {
	return &BidController{bidService: bidService}
}

func (c *BidController) PlaceBid(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		exception.WriteError(w, exception.ErrUnauthenticated)
		return
	}

	auctionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		exception.WriteError(w, exception.ErrValidation)
		return
	}

	var req bid_request.PlaceBidRequest
	if err := helper.ReadJSON(w, r, &req); err != nil {
		exception.WriteError(w, exception.ErrValidation)
		return
	}

	result, err := c.bidService.PlaceBid(
		r.Context(),
		principal.UserID,
		auctionID,
		req,
	)
	if err != nil {
		exception.WriteError(w, err)
		return
	}

	status := http.StatusCreated
	if result.Replay {
		status = http.StatusOK
	}

	helper.WriteJSON(w, status, result)
}
