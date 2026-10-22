package controller

import (
	"errors"
	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/service"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

type AuctionController struct {
	auctionService service.AuctionService
}

func NewAuctionController(auctionService service.AuctionService) *AuctionController {
	return &AuctionController{
		auctionService: auctionService,
	}
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
