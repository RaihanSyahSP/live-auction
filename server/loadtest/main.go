package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type auctionDetail struct {
	Auction struct {
		CurrentPrice int64  `json:"current_price"`
		MinIncrement int64  `json:"min_increment"`
		Status       string `json:"status"`
	} `json:"auction"`
}

type wsEvent struct {
	Type    string `json:"type"`
	Payload struct {
		Bid struct {
			Amount int64 `json:"amount"`
		} `json:"bid"`
	} `json:"payload"`
}

type bidder struct {
	client *http.Client
	email  string
}

func main() {
	baseURL := flag.String("base-url", "http://localhost:8080", "API base URL")
	auctionID := flag.String("auction-id", "", "auction UUID")
	viewers := flag.Int("viewers", 50, "number of WebSocket viewers")
	bidders := flag.Int("bidders", 5, "number of bidder accounts")
	ratePerSecond := flag.Int("rate", 5, "total bids per second")
	duration := flag.Duration("duration", 30*time.Second, "test duration")
	origin := flag.String("origin", "http://localhost:4173", "allowed frontend origin")
	password := flag.String("password", os.Getenv("SEED_BIDDER_PASSWORD"), "bidder password")
	allowRemote := flag.Bool("allow-remote", false, "allow non-local base URL")
	flag.Parse()

	if *auctionID == "" {
		log.Fatal("-auction-id is required")
	}
	if *bidders < 1 {
		log.Fatal("-bidders must be at least 1")
	}
	if *ratePerSecond < 1 {
		log.Fatal("-rate must be at least 1")
	}
	if *password == "" {
		log.Fatal("-password or SEED_BIDDER_PASSWORD is required")
	}
	if err := validateTarget(*baseURL, *allowRemote); err != nil {
		log.Fatal(err)
	}
	if _, err := uuid.Parse(*auctionID); err != nil {
		log.Fatalf("invalid auction id: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	detail, err := fetchAuction(ctx, *baseURL, *auctionID)
	if err != nil {
		log.Fatalf("fetch auction: %v", err)
	}
	if detail.Auction.Status != "live" {
		log.Fatalf("auction status = %q, want live", detail.Auction.Status)
	}

	biddersList, err := makeBidders(ctx, *baseURL, *origin, *password, *bidders)
	if err != nil {
		log.Fatalf("prepare bidders: %v", err)
	}

	viewerConns, err := connectViewers(ctx, *baseURL, *auctionID, *origin, *viewers)
	if err != nil {
		log.Fatalf("connect viewers: %v", err)
	}
	defer closeViewers(viewerConns)

	var mu sync.Mutex
	sentAt := make(map[int64]time.Time)
	var latencies []time.Duration

	for i, conn := range viewerConns {
		if i == 0 {
			go readEvents(ctx, conn, &mu, sentAt, &latencies)
			continue
		}
		go func(conn *websocket.Conn) {
			_, _, _ = conn.Read(ctx)
		}(conn)
	}

	amount := detail.Auction.CurrentPrice + detail.Auction.MinIncrement
	interval := time.Second / time.Duration(*ratePerSecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	deadline := time.Now().Add(*duration)
	accepted := 0
	rejected := 0
	errorsCount := 0
	bidIndex := 0

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			bidder := biddersList[bidIndex%len(biddersList)]
			bidIndex++
			key := uuid.NewString()
			currentAmount := amount
			amount += detail.Auction.MinIncrement

			mu.Lock()
			sentAt[currentAmount] = time.Now()
			mu.Unlock()

			status, err := postJSON(bidder.client, *baseURL+"/api/auctions/"+*auctionID+"/bids", *origin, map[string]any{
				"amount":          currentAmount,
				"idempotency_key": key,
			})
			if err != nil {
				mu.Lock()
				delete(sentAt, currentAmount)
				mu.Unlock()
				errorsCount++
				continue
			}

			switch {
			case status == http.StatusCreated || status == http.StatusOK:
				accepted++
			case status >= 400 && status < 500:
				rejected++
				mu.Lock()
				delete(sentAt, currentAmount)
				mu.Unlock()
			default:
				errorsCount++
				mu.Lock()
				delete(sentAt, currentAmount)
				mu.Unlock()
			}
		}
	}

done:
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	result := append([]time.Duration(nil), latencies...)
	mu.Unlock()
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })

	fmt.Printf("accepted=%d rejected=%d errors=%d\n", accepted, rejected, errorsCount)
	if len(result) == 0 {
		fmt.Println("broadcast latency: no samples")
		return
	}

	fmt.Printf(
		"broadcast latency: p50=%s p95=%s p99=%s samples=%d\n",
		percentile(result, 0.50),
		percentile(result, 0.95),
		percentile(result, 0.99),
		len(result),
	)
}

func validateTarget(rawURL string, allowRemote bool) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid -base-url")
	}
	if strings.Contains(strings.ToLower(parsed.Host), "prod") {
		return fmt.Errorf("refusing to run against a production-looking host")
	}
	if allowRemote {
		return nil
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return fmt.Errorf("remote target requires -allow-remote")
	}
	return nil
}

func fetchAuction(ctx context.Context, baseURL, auctionID string) (auctionDetail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/auctions/"+auctionID, nil)
	if err != nil {
		return auctionDetail{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return auctionDetail{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return auctionDetail{}, fmt.Errorf("status %d", resp.StatusCode)
	}

	var detail auctionDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return auctionDetail{}, err
	}
	return detail, nil
}

func makeBidders(ctx context.Context, baseURL, origin, password string, count int) ([]bidder, error) {
	result := make([]bidder, 0, count)
	for i := 0; i < count; i++ {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		client := &http.Client{Jar: jar, Timeout: 10 * time.Second}
		email := fmt.Sprintf("loadtest%d@example.com", i+1)

		status, err := postJSON(client, baseURL+"/api/auth/login", origin, map[string]any{
			"email":    email,
			"password": password,
		})
		if err != nil {
			return nil, err
		}
		if status == http.StatusUnauthorized {
			status, err = postJSON(client, baseURL+"/api/auth/register", origin, map[string]any{
				"email":        email,
				"password":     password,
				"display_name": fmt.Sprintf("Loadtest %d", i+1),
			})
			if err != nil {
				return nil, err
			}
		}
		if status != http.StatusOK && status != http.StatusCreated {
			return nil, fmt.Errorf("login/register %s returned %d", email, status)
		}

		result = append(result, bidder{client: client, email: email})
	}
	return result, nil
}

func connectViewers(
	ctx context.Context,
	baseURL string,
	auctionID string,
	origin string,
	count int,
) ([]*websocket.Conn, error) {
	wsBase := strings.Replace(baseURL, "https://", "wss://", 1)
	wsBase = strings.Replace(wsBase, "http://", "ws://", 1)
	wsURL := wsBase + "/ws/auctions/" + auctionID

	conns := make([]*websocket.Conn, 0, count)
	for i := 0; i < count; i++ {
		dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		conn, _, err := websocket.Dial(dialCtx, wsURL, &websocket.DialOptions{
			HTTPHeader: http.Header{"Origin": []string{origin}},
		})
		cancel()
		if err != nil {
			closeViewers(conns)
			return nil, err
		}

		readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
		_, _, err = conn.Read(readCtx)
		readCancel()
		if err != nil {
			_ = conn.Close(websocket.StatusNormalClosure, "snapshot failed")
			closeViewers(conns)
			return nil, err
		}
		conns = append(conns, conn)
	}

	return conns, nil
}

func closeViewers(conns []*websocket.Conn) {
	for _, conn := range conns {
		_ = conn.Close(websocket.StatusNormalClosure, "load test done")
	}
}

func readEvents(
	ctx context.Context,
	conn *websocket.Conn,
	mu *sync.Mutex,
	sentAt map[int64]time.Time,
	latencies *[]time.Duration,
) {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}

		var event wsEvent
		if err := json.Unmarshal(data, &event); err != nil || event.Type != "bid" {
			continue
		}

		mu.Lock()
		if started, ok := sentAt[event.Payload.Bid.Amount]; ok {
			*latencies = append(*latencies, time.Since(started))
			delete(sentAt, event.Payload.Bid.Amount)
		}
		mu.Unlock()
	}
}

func postJSON(client *http.Client, targetURL, origin string, payload any) (int, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest(http.MethodPost, targetURL, strings.NewReader(string(data)))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Ceil(float64(len(sorted))*p)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
