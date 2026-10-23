package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"live-auction/server/internal/auth"
	"live-auction/server/internal/middleware"
	"live-auction/server/internal/model/domain"
)

func TestLoggingAddsRequestIDAndSafeFields(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	userID := uuid.New()

	handler := middleware.Logging(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := middleware.WithPrincipal(r.Context(), auth.Principal{
			UserID: userID,
			Role:   domain.RoleUser,
		})
		_ = r.WithContext(ctx)
		w.WriteHeader(http.StatusCreated)
	}))

	auctionID := uuid.New()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/auctions/"+auctionID.String()+"/bids?email=secret@example.com",
		nil,
	)
	req.SetPathValue("id", auctionID.String())
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "super-secret"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("X-Request-ID is empty")
	}

	logLine := output.String()
	if strings.Contains(logLine, "secret@example.com") {
		t.Error("log contains email from query")
	}
	if strings.Contains(logLine, "super-secret") {
		t.Error("log contains cookie value")
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(logLine)), &entry); err != nil {
		t.Fatalf("decode log: %v", err)
	}

	if entry["request_id"] != requestID {
		t.Errorf("request_id = %v, want %s", entry["request_id"], requestID)
	}
	if entry["method"] != http.MethodPost {
		t.Errorf("method = %v, want POST", entry["method"])
	}
	if entry["status"] != float64(http.StatusCreated) {
		t.Errorf("status = %v, want 201", entry["status"])
	}
	if entry["user_id"] != userID.String() {
		t.Errorf("user_id = %v, want %s", entry["user_id"], userID)
	}
	if _, ok := entry["auction_id"]; !ok {
		t.Error("auction_id is missing")
	}
	if _, ok := entry["duration_ms"]; !ok {
		t.Error("duration_ms is missing")
	}
}

func TestLoggingPreservesWebSocketUpgrade(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			t.Errorf("accept websocket: %v", err)
			return
		}
		_ = conn.Close(websocket.StatusNormalClosure, "done")
	})

	server := httptest.NewServer(middleware.Logging(logger, mux))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	_ = conn.Close(websocket.StatusNormalClosure, "test done")
}

func TestLoggingRecordsDefaultStatus(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))

	handler := middleware.Logging(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.String())), &entry); err != nil {
		t.Fatalf("decode log: %v", err)
	}
	if entry["status"] != float64(http.StatusOK) {
		t.Errorf("status = %v, want 200", entry["status"])
	}
}
