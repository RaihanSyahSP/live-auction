package controller

import (
	"context"
	"encoding/json"
	"live-auction/server/internal/auth"
	"live-auction/server/internal/realtime"
	"live-auction/server/internal/service"
	"net/http"
	"net/url"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type WebSocketController struct {
	rooms         *realtime.RoomManager
	auctions      service.AuctionService
	tokens        *auth.AccessTokens
	allowedOrigin string
}

func NewWebSocketController(
	rooms *realtime.RoomManager,
	auctions service.AuctionService,
	tokens *auth.AccessTokens,
	allowedOrigin string,
) *WebSocketController {
	return &WebSocketController{
		rooms:         rooms,
		auctions:      auctions,
		tokens:        tokens,
		allowedOrigin: allowedOrigin,
	}
}

func (c *WebSocketController) ServeWS(w http.ResponseWriter, r *http.Request) {
	auctionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid auction id", http.StatusUnprocessableEntity)
		return
	}

	userID := c.optionalUserID(r)

	detail, err := c.auctions.Detail(r.Context(), auctionID)
	if err != nil {
		http.Error(w, "auction not found", http.StatusNotFound)
		return
	}

	originPatterns := []string{}
	if u, err := url.Parse(c.allowedOrigin); err == nil && u.Host != "" {
		originPatterns = append(originPatterns, u.Host)
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: originPatterns,
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "closed")

	fallback := realtime.Snapshot{
		Type:       "snapshot",
		Version:    detail.Auction.Version,
		ServerTime: detail.ServerTime,
		Auction:    detail.Auction,
		Bids:       detail.Bids,
	}

	client, snapshot := c.rooms.Join(auctionID, userID, fallback)
	defer c.rooms.Leave(auctionID, client)

	if err := writeJSON(r.Context(), conn, snapshot); err != nil {
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go c.writeLoop(ctx, conn, client)
	go c.pingLoop(ctx, conn)

	for {
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
	}
}

func (c *WebSocketController) optionalUserID(r *http.Request) *uuid.UUID {
	cookie, err := r.Cookie(auth.AccessCookieName)
	if err != nil {
		return nil
	}

	principal, err := c.tokens.Parse(cookie.Value)
	if err != nil {
		return nil
	}

	return &principal.UserID
}

func (c *WebSocketController) writeLoop(ctx context.Context, conn *websocket.Conn, client *realtime.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-client.Send:
			if !ok {
				return
			}

			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, data)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *WebSocketController) pingLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func writeJSON(ctx context.Context, conn *websocket.Conn, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return conn.Write(writeCtx, websocket.MessageText, data)
}
