package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/metrics"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/model/web/bid_request"
	"live-auction/server/internal/realtime"
	"live-auction/server/internal/service"
)

type BidController struct {
	bidService service.BidService
	rooms      *realtime.RoomManager
	instrument *metrics.Metrics
}

func NewBidController(
	bidService service.BidService,
	rooms *realtime.RoomManager,
	instrument ...*metrics.Metrics,
) *BidController {
	c := &BidController{bidService: bidService, rooms: rooms}
	if len(instrument) > 0 {
		c.instrument = instrument[0]
	}
	return c
}

func (c *BidController) MyBids(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		exception.WriteError(w, exception.ErrUnauthenticated)
		return
	}

	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			exception.WriteError(w, exception.ErrValidation)
			return
		}
		page = parsed
	}

	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			exception.WriteError(w, exception.ErrValidation)
			return
		}
		limit = parsed
	}

	all := false
	if raw := r.URL.Query().Get("all"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			exception.WriteError(w, exception.ErrValidation)
			return
		}
		all = parsed
	}

	result, err := c.bidService.MyBids(
		r.Context(),
		principal.UserID,
		page,
		limit,
		all,
	)
	if err != nil {
		exception.WriteError(w, err)
		return
	}

	helper.WriteJSON(w, http.StatusOK, result)
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

	started := time.Now()

	result, err := c.bidService.PlaceBid(
		r.Context(),
		principal.UserID,
		auctionID,
		req,
	)
	if err != nil {
		c.instrument.ObserveBid("error", time.Since(started))
		exception.WriteError(w, err)
		return
	}

	resultLabel := "success"
	if result.Replay {
		resultLabel = "replay"
	}
	c.instrument.ObserveBid(resultLabel, time.Since(started))

	status := http.StatusCreated
	if result.Replay {
		status = http.StatusOK
	}

	helper.WriteJSON(w, status, result)

	c.rooms.PublishBid(
		result.Auction,
		result.Bid,
		result.PreviousHighestUserID,
		result.Extended,
	)

}
