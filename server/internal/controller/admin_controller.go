package controller

import (
	"live-auction/server/internal/exception"
	"live-auction/server/internal/helper"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/model/web/auction_request"
	"live-auction/server/internal/model/web/auction_response"
	"live-auction/server/internal/realtime"
	"live-auction/server/internal/service"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

type AdminController struct {
	auctionService service.AuctionService
	rooms          *realtime.RoomManager
}

func NewAdminController(auctionService service.AuctionService, rooms *realtime.RoomManager) *AdminController {
	return &AdminController{
		auctionService: auctionService,
		rooms:          rooms,
	}
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
	result, ok := c.changeStatus(w, r, "end")
	if !ok {
		return
	}
	c.rooms.PublishEnded(result)
}

func (c *AdminController) changeStatus(w http.ResponseWriter, r *http.Request, action string) (auction_response.AuctionResponse, bool) {
	principal, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		exception.WriteError(w, exception.ErrUnauthenticated)
		return auction_response.AuctionResponse{}, false
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		exception.WriteError(w, exception.ErrValidation)
		return auction_response.AuctionResponse{}, false
	}

	var result auction_response.AuctionResponse
	if action == "start" {
		result, err = c.auctionService.Start(r.Context(), principal.UserID, id)
	} else {
		result, err = c.auctionService.End(r.Context(), principal.UserID, id)
	}
	if err != nil {
		exception.WriteError(w, err)
		return auction_response.AuctionResponse{}, false
	}

	helper.WriteJSON(w, http.StatusOK, result)
	return result, true
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
