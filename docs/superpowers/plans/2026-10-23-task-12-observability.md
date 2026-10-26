# Task 12: Observability, Seed, Load Test, dan Runbook

> **Untuk implementasi manual:** kerjakan per bagian, tulis test terlebih dahulu, jalankan test, lalu implementasi. Jangan commit sebelum seluruh verifikasi lulus.

**Goal:** Backend punya request logging, metrics Prometheus, health check yang mengecek database, seed data demo, load test lokal, dan runbook yang bisa dijalankan orang lain.

**Architecture:** Logging dan metrics dipasang di boundary HTTP. Metrics dikumpulkan di package terpisah agar controller dan realtime bisa mengirim observasi tanpa import cycle. Seed ditempatkan di package internal supaya bisa diuji, sedangkan `cmd/seed` hanya menjadi entrypoint.

**Tech Stack:** Go, `net/http`, `pgx/v5`, PostgreSQL, `log/slog`, `prometheus/client_golang`, `coder/websocket`, `bcrypt`.

**Spec:** `docs/superpowers/specs/2026-10-20-auction-backend-design.md` bagian 11, 12, 13, dan 14.

---

## Global Constraints

- Log memakai `slog` JSON.
- Jangan log password, token, cookie, email mentah, atau body request.
- Metrics memakai registry sendiri, bukan global default registry.
- `/metrics` boleh publik di local dan staging, tetapi jangan mengekspos data sensitif.
- `/healthz` mengembalikan 503 jika database tidak bisa di-ping.
- Seed harus idempotent.
- Load test hanya untuk local atau staging, bukan production.
- Tidak menambah ORM atau framework router.
- Semua ID tetap UUID.

---

## Review Focus

- Logging tidak boleh membocorkan cookie, token, email, atau password.
- Logging middleware tidak boleh merusak WebSocket upgrade. Response writer wrapper harus meneruskan `http.Hijacker`.
- Metrics tidak boleh panic jika controller diuji tanpa metrics.
- Metrics endpoint harus mengembalikan nama metric yang stabil.
- Health check tidak boleh mengembalikan 200 saat database mati.
- Seed yang dijalankan dua kali tidak boleh membuat user, auction, atau live auction ganda.
- Load test tidak boleh diarahkan ke production secara tidak sengaja.

---

## 1. Flow

### 1.1 Request logging

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant L as Logging Middleware
    participant R as Router
    participant H as Handler
    participant W as ResponseWriter

    C->>L: HTTP request
    L->>L: buat request_id
    L->>W: set X-Request-ID
    L->>R: next.ServeHTTP
    R->>H: handler
    H-->>W: status + body
    L->>L: hitung duration_ms
    L-->>C: response
    L->>L: slog JSON
```

Log yang aman:

```json
{
  "level": "INFO",
  "msg": "http request",
  "request_id": "uuid",
  "method": "POST",
  "path": "/api/auctions/uuid/bids",
  "status": 201,
  "duration_ms": 12,
  "user_id": "uuid",
  "auction_id": "uuid"
}
```

Yang tidak boleh masuk log:

```text
email
cookie
access_token
refresh_token
password
request body
```

### 1.2 Metrics

```mermaid
flowchart LR
    B[Browser] -->|bid| BC[BidController]
    B -->|WebSocket| WS[WebSocketController]
    BC --> M[Metrics]
    WS --> M
    RM[RoomManager] --> M
    M --> P[Prometheus registry]
    P --> E[/metrics]
```

Metrics yang wajib ada:

```text
auction_connected_clients{auction_id}
auction_bids_total{result}
auction_bid_duration_seconds
auction_broadcast_duration_seconds
auction_live_auctions
auth_refresh_total{result}
auth_login_total{result}
```

### 1.3 Health check

```mermaid
flowchart TD
    A[GET /healthz] --> B[db.Ping]
    B -->|sukses| C[200 status ok]
    B -->|gagal| D[503 status unhealthy]
```

### 1.4 Seed

```mermaid
flowchart TD
    A[go run ./cmd/seed] --> B[load .env]
    B --> C[connect PostgreSQL]
    C --> D[ensure admin]
    D --> E[ensure bidder]
    E --> F[ensure live demo auction]
    F --> G[ensure draft demo auction]
    G --> H[commit]
```

---

## 2. Keputusan struktur file

### Ruling: metrics bukan di `internal/app`

Rencana awal menyebut `server/internal/app/metrics.go`. Itu akan menimbulkan import cycle:

```text
app -> controller -> app
```

Karena `controller` dan `realtime` perlu mengirim metrics, metrics dipindahkan ke package sendiri:

```text
server/internal/metrics/metrics.go
```

`app` hanya menyediakan route `/metrics` dan menerima `http.Handler`.

### Ruling: seed logic di `internal/seed`

`cmd/seed/main.go` tidak bisa di-import oleh test karena package `main`.

Supaya seed bisa diuji, logika seed diletakkan di:

```text
server/internal/seed/seed.go
server/internal/seed/seed_test.go
```

`cmd/seed/main.go` hanya:

1. load `.env`;
2. load config;
3. connect database;
4. memanggil `seed.Run`;
5. log hasil.

---

## 3. File yang dibuat dan diubah

```text
Create:
  server/internal/middleware/logging.go
  server/internal/middleware/logging_test.go

  server/internal/metrics/metrics.go
  server/internal/metrics/metrics_test.go

  server/internal/seed/seed.go
  server/internal/seed/seed_test.go

  server/cmd/seed/main.go

  server/loadtest/main.go

Modify:
  server/internal/middleware/auth.go
  server/internal/controller/health_controller.go
  server/internal/controller/health_controller_test.go

  server/internal/controller/bid_controller.go
  server/internal/controller/bid_controller_test.go

  server/internal/controller/auth_controller.go
  server/internal/controller/auth_controller_test.go

  server/internal/realtime/manager.go
  server/internal/realtime/manager_test.go

  server/internal/service/auction_service.go

  server/internal/app/router.go
  server/internal/app/router_test.go

  server/cmd/api/main.go
  server/README.md
```

---

## 4. Logging middleware

### 4.1 Interface

```go
func Logging(logger *slog.Logger, next http.Handler) http.Handler
```

### 4.2 Response writer yang aman untuk WebSocket

Buat wrapper:

```go
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(data)
}

func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}
```

`Hijack` wajib. Tanpa itu, `websocket.Accept` tidak bisa melakukan upgrade karena wrapper tidak dianggap `http.Hijacker`.

### 4.3 Request ID dan log metadata

Buat helper:

```go
type requestIDKey struct{}
type logMetaKey struct{}

type logMeta struct {
	UserID *uuid.UUID
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestIDFrom(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

func logMetaFrom(ctx context.Context) *logMeta {
	meta, _ := ctx.Value(logMetaKey{}).(*logMeta)
	return meta
}
```

Kenapa perlu `logMeta`?

Karena `Auth` middleware membuat context baru dengan `r.WithContext`, sehingga logging middleware tidak bisa melihat principal hanya dari context awalnya. `logMeta` adalah pointer yang bisa diisi oleh `WithPrincipal`.

Ubah `server/internal/middleware/auth.go`:

```go
func WithPrincipal(ctx context.Context, principal auth.Principal) context.Context {
	ctx = context.WithValue(ctx, principalContextKey{}, principal)

	if meta := logMetaFrom(ctx); meta != nil {
		userID := principal.UserID
		meta.UserID = &userID
	}

	return ctx
}
```

`logMeta` hanya menyimpan user ID. Jangan menyimpan password, email, token, atau cookie.

### 4.4 Handler logging

```go
func Logging(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.NewString()
		meta := &logMeta{}

		ctx := WithRequestID(r.Context(), requestID)
		ctx = context.WithValue(ctx, logMetaKey{}, meta)
		r = r.WithContext(ctx)

		w.Header().Set("X-Request-ID", requestID)

		recorder := &responseRecorder{ResponseWriter: w}
		started := time.Now()

		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		attrs := []any{
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", time.Since(started).Milliseconds(),
		}

		if meta.UserID != nil {
			attrs = append(attrs, "user_id", meta.UserID.String())
		}

		if auctionID, err := uuid.Parse(r.PathValue("id")); err == nil {
			attrs = append(attrs, "auction_id", auctionID.String())
		}

		logger.Info("http request", attrs...)
	})
}
```

Catatan:

- Jangan log `r.URL.RawQuery`.
- Jangan log header.
- Jangan log body.
- `auction_id` hanya diisi jika path `{id}` berisi UUID valid.
- `user_id` diisi oleh `WithPrincipal` melalui `logMeta`.

### 4.5 Pasang di `main.go`

Setelah router dibuat:

```go
handler = middleware.Logging(logger, handler)
server := app.NewServer(cfg, handler)
```

### 4.6 Test logging

Buat `server/internal/middleware/logging_test.go`:

```go
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

	req := httptest.NewRequest(http.MethodPost, "/api/auctions/123/bids?email=secret@example.com", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "super-secret"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID is empty")
	}

	logLine := output.String()
	if !strings.Contains(logLine, "request_id") {
		t.Error("log does not contain request_id")
	}
	if !strings.Contains(logLine, userID.String()) {
		t.Error("log does not contain user_id")
	}
	if strings.Contains(logLine, "secret@example.com") {
		t.Error("log contains email from query")
	}
	if strings.Contains(logLine, "super-secret") {
		t.Error("log contains cookie value")
	}
}
```

Catatan: `r.WithContext(ctx)` tidak perlu dipakai handler karena logging sudah memegang pointer `logMeta`. Yang penting `WithPrincipal` menerima context yang memiliki `logMeta`.

Tambahkan test bahwa status yang ditulis handler ikut tercatat.

---

## 5. Metrics

### 5.1 Dependency

```powershell
cd D:\live-auction\server
go get github.com/prometheus/client_golang@latest
go mod tidy
```

### 5.2 Package metrics

Buat `server/internal/metrics/metrics.go`.

Interface internal:

```go
type Metrics struct {
	registry           *prometheus.Registry
	bidsTotal          *prometheus.CounterVec
	bidDuration        prometheus.Histogram
	broadcastDuration  prometheus.Histogram
	connectedClients   *prometheus.GaugeVec
	liveAuctions       prometheus.Gauge
	loginTotal         *prometheus.CounterVec
	refreshTotal       *prometheus.CounterVec
}

func New() *Metrics
func (m *Metrics) Handler() http.Handler

func (m *Metrics) ObserveBid(result string, duration time.Duration)
func (m *Metrics) ObserveBroadcast(duration time.Duration)
func (m *Metrics) SetConnectedClients(auctionID string, count int)
func (m *Metrics) SetLiveAuctions(count int)
func (m *Metrics) ObserveLogin(result string)
func (m *Metrics) ObserveRefresh(result string)
```

Semua method harus nil-safe:

```go
if m == nil {
	return
}
```

Ini supaya test controller tidak wajib membuat metrics.

Nama metric:

```go
prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "auction_bids_total",
		Help: "Total bid requests by result.",
	},
	[]string{"result"},
)

prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "auction_bid_duration_seconds",
		Help:    "Bid service duration.",
		Buckets: prometheus.DefBuckets,
	},
)

prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Name:    "auction_broadcast_duration_seconds",
		Help:    "WebSocket broadcast duration.",
		Buckets: prometheus.DefBuckets,
	},
)

prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "auction_connected_clients",
		Help: "Connected WebSocket clients per auction.",
	},
	[]string{"auction_id"},
)

prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "auction_live_auctions",
		Help: "Number of live auctions.",
	},
)

prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "auth_login_total",
		Help: "Login attempts by result.",
	},
	[]string{"result"},
)

prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "auth_refresh_total",
		Help: "Refresh attempts by result.",
	},
	[]string{"result"},
)
```

Gunakan registry baru:

```go
registry := prometheus.NewRegistry()
registry.MustRegister(
	bidsTotal,
	bidDuration,
	broadcastDuration,
	connectedClients,
	liveAuctions,
	loginTotal,
	refreshTotal,
)
```

`Handler`:

```go
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
```

### 5.3 Test metrics

Buat `server/internal/metrics/metrics_test.go`:

```go
func TestMetricsHandlerExposesMetrics(t *testing.T) {
	m := metrics.New()
	m.ObserveBid("success", 10*time.Millisecond)
	m.SetConnectedClients("auction-id", 3)
	m.SetLiveAuctions(1)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	body := rec.Body.String()
	for _, name := range []string{
		"auction_bids_total",
		"auction_bid_duration_seconds",
		"auction_broadcast_duration_seconds",
		"auction_connected_clients",
		"auction_live_auctions",
		"auth_login_total",
		"auth_refresh_total",
	} {
		if !strings.Contains(body, name) {
			t.Errorf("metrics body does not contain %s", name)
		}
	}
}
```

### 5.4 Instrumentasi BidController

Ubah constructor:

```go
func NewBidController(
	bidService service.BidService,
	rooms *realtime.RoomManager,
	instrument *metrics.Metrics,
) *BidController
```

Di `PlaceBid`:

```go
started := time.Now()

result, err := c.bidService.PlaceBid(...)
if err != nil {
	c.metrics.ObserveBid("error", time.Since(started))
	exception.WriteError(w, err)
	return
}

resultLabel := "success"
if result.Replay {
	resultLabel = "replay"
}
c.metrics.ObserveBid(resultLabel, time.Since(started))

status := http.StatusCreated
if result.Replay {
	status = http.StatusOK
}
helper.WriteJSON(w, status, result)
c.rooms.PublishBid(...)
```

Update test constructor dengan parameter `nil`.

### 5.5 Instrumentasi AuthController

Ubah constructor:

```go
func NewAuthController(
	authService service.AuthService,
	cookies auth.Cookies,
	instrument *metrics.Metrics,
) *AuthController
```

Di `Login`:

```go
result, err := c.authService.Login(r.Context(), req)
if err != nil {
	c.metrics.ObserveLogin("error")
	exception.WriteError(w, err)
	return
}

c.metrics.ObserveLogin("success")
```

Di `Refresh`:

```go
result, err := c.authService.Refresh(r.Context(), rawRefresh)
if err != nil {
	c.metrics.ObserveRefresh("error")
	c.cookies.Clear(w)
	exception.WriteError(w, err)
	return
}

c.metrics.ObserveRefresh("success")
```

Update test constructor dengan parameter `nil`.

### 5.6 Instrumentasi RoomManager

Tambahkan field:

```go
type RoomManager struct {
	mu       sync.Mutex
	rooms    map[uuid.UUID]*Room
	now      func() time.Time
	metrics  *metrics.Metrics
}
```

Ubah constructor menjadi variadic agar test lama tetap compile:

```go
func NewRoomManager(instrument ...*metrics.Metrics) *RoomManager {
	m := &RoomManager{
		rooms: make(map[uuid.UUID]*Room),
		now:   time.Now,
	}
	if len(instrument) > 0 {
		m.metrics = instrument[0]
	}
	return m
}
```

Di `Join`, setelah menambah client:

```go
m.metrics.SetConnectedClients(auctionID.String(), len(room.clients))
```

Di `Leave`, setelah menghapus client:

```go
m.metrics.SetConnectedClients(auctionID.String(), len(room.clients))
```

Di `PublishBid`, ukur broadcast:

```go
started := time.Now()
// ... update room dan broadcast
m.metrics.ObserveBroadcast(time.Since(started))
```

### 5.7 Instrumentasi AuctionService

Agar `auction_live_auctions` terisi, tambahkan metrics opsional ke `NewAuctionService`:

```go
func NewAuctionService(
	auctions auctionrepo.AuctionRepository,
	bids bidrepo.BidRepository,
	tx repository.TxManager,
	now func() time.Time,
	instrument ...*metrics.Metrics,
) AuctionService
```

Simpan metrics jika ada.

Setelah `Start` sukses:

```go
s.metrics.SetLiveAuctions(1)
```

Setelah `End` sukses:

```go
s.metrics.SetLiveAuctions(0)
```

Jika metrics nil, method nil-safe.

### 5.8 Route `/metrics`

Tambahkan ke `RouterDeps`:

```go
Metrics http.Handler
```

Daftarkan sebelum guard `Auth == nil`:

```go
if deps.Metrics != nil {
	mux.Handle("GET /metrics", deps.Metrics)
}
```

Di `main.go`:

```go
instrument := metrics.New()

auctionService := service.NewAuctionService(
	auctionRepository,
	bidRepository,
	txManager,
	time.Now,
	instrument,
)
```

Lalu:

```go
rooms := realtime.NewRoomManager(instrument)
```

Bid controller:

```go
bidController := controller.NewBidController(bidService, rooms, instrument)
```

Auth controller:

```go
authController := controller.NewAuthController(authService, cookies, instrument)
```

Router deps:

```go
Metrics: instrument.Handler(),
```

Setelah startup recovery, sinkronkan gauge live auction:

```go
if _, err := auctionService.Current(ctx); err == nil {
	instrument.SetLiveAuctions(1)
} else if errors.Is(err, exception.ErrAuctionNotFound) {
	instrument.SetLiveAuctions(0)
}
```

`main.go` perlu import `exception` untuk pengecekan ini. Gauge ini penting karena server bisa hidup kembali dengan lelang yang masih live.

---

## 6. Health check

### 6.1 Ubah controller

```go
type HealthChecker interface {
	Ping(ctx context.Context) error
}

type HealthController struct {
	db HealthChecker
}

func NewHealthController(db HealthChecker) *HealthController {
	return &HealthController{db: db}
}

func (h *HealthController) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := h.db.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
```

Jangan tampilkan detail error database ke response.

### 6.2 Test health

Fake checker:

```go
type fakeHealthChecker struct {
	err error
}

func (f *fakeHealthChecker) Ping(ctx context.Context) error {
	return f.err
}
```

Test:

- ping sukses -> 200, `{"status":"ok"}`;
- ping gagal -> 503, `{"status":"unhealthy"}`.

### 6.3 Wiring

```go
Health: controller.NewHealthController(pool),
```

`*pgxpool.Pool` sudah memiliki method `Ping(context.Context) error`.

---

## 7. Seed

### 7.1 Package seed

Buat `server/internal/seed/seed.go`.

Interface:

```go
type Config struct {
	AdminEmail     string
	AdminPassword  string
	BidderPassword string
}

func Run(ctx context.Context, pool *pgxpool.Pool, cfg Config) error
```

Validasi:

```go
if cfg.AdminEmail == "" || cfg.AdminPassword == "" || cfg.BidderPassword == "" {
	return errors.New("seed credentials are required")
}
```

### 7.2 Data yang dibuat

```text
admin@example.com        role admin, display_name Admin Demo
bidder@example.com       role user,  display_name Bidder Demo
LOT-DEMO-LIVE            live, 10 menit, current_price 1.000.000
LOT-DEMO-DRAFT           draft, current_price 500.000
```

Semua password di-hash dengan bcrypt cost 12.

### 7.3 Idempotensi user

Gunakan pola:

```go
func ensureUser(
	ctx context.Context,
	tx pgx.Tx,
	email string,
	password string,
	displayName string,
	role string,
) (uuid.UUID, error) {
	var id uuid.UUID

	err := tx.QueryRow(
		ctx,
		`select id from users where lower(email) = lower($1)`,
		email,
	).Scan(&id)

	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return uuid.Nil, err
	}

	err = tx.QueryRow(
		ctx,
		`insert into users (email, password_hash, display_name, role)
		 values ($1, $2, $3, $4)
		 returning id`,
		email,
		string(hash),
		displayName,
		role,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}
```

### 7.4 Idempotensi auction

Gunakan lot sebagai kunci:

```go
func ensureAuction(
	ctx context.Context,
	tx pgx.Tx,
	createdBy uuid.UUID,
	lot string,
	title string,
	description string,
	startingPrice int64,
	minIncrement int64,
	status string,
	startsAt *time.Time,
	endsAt *time.Time,
) (uuid.UUID, error) {
	var id uuid.UUID

	err := tx.QueryRow(
		ctx,
		`select id from auctions where lot = $1`,
		lot,
	).Scan(&id)

	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}

	err = tx.QueryRow(
		ctx,
		`insert into auctions (
			created_by, lot, title, description, condition,
			starting_price, min_increment, duration_seconds,
			current_price, status, starts_at, ends_at,
			includes, image_urls, specs
		) values (
			$1, $2, $3, $4, 'Used',
			$5, $6, 600,
			$5, $7, $8, $9,
			'{}', '{}', '[]'::jsonb
		) returning id`,
		createdBy,
		lot,
		title,
		description,
		startingPrice,
		minIncrement,
		status,
		startsAt,
		endsAt,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}
```

Untuk live auction, cek dulu apakah sudah ada live lain:

```go
var liveID uuid.UUID
err := tx.QueryRow(ctx, `select id from auctions where status = 'live' limit 1`).Scan(&liveID)
if err != nil && !errors.Is(err, pgx.ErrNoRows) {
	return err
}
if errors.Is(err, pgx.ErrNoRows) {
	// aman membuat live demo
}
```

Jika sudah ada live, jangan membuat live kedua. Cukup lewati live demo.

### 7.5 Test seed

Buat `server/internal/seed/seed_test.go`:

- skip jika `TEST_DATABASE_URL` kosong;
- jalankan `seed.Run` dua kali;
- hitung jumlah user dengan email seed;
- hitung jumlah auction dengan lot seed;
- pastikan tidak ada dua live auction;
- cleanup data seed setelah test.

Gunakan helper:

```go
pool := helper.OpenTestDB(t)
```

Hati-hati: seed memakai transaksi commit, jadi cleanup harus eksplisit.

### 7.6 Command seed

Buat `server/cmd/seed/main.go`:

```go
func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := app.NewDB(ctx, cfg)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := seed.Run(ctx, pool, seed.Config{
		AdminEmail:     cfg.SeedAdminEmail,
		AdminPassword:  cfg.SeedAdminPassword,
		BidderPassword: cfg.SeedBidderPassword,
	}); err != nil {
		slog.Error("run seed", "error", err)
		os.Exit(1)
	}

	slog.Info("seed complete")
}
```

Jalankan dari folder `server`:

```powershell
cd D:\live-auction\server
go run ./cmd/seed
```

---

## 8. Load test

Buat `server/loadtest/main.go`.

### Flags

```go
baseURL := flag.String("base-url", "http://localhost:8080", "API base URL")
auctionID := flag.String("auction-id", "", "auction UUID")
viewers := flag.Int("viewers", 50, "number of WebSocket viewers")
bidders := flag.Int("bidders", 5, "number of bidder accounts")
ratePerSecond := flag.Int("rate", 5, "total bids per second")
duration := flag.Duration("duration", 30*time.Second, "test duration")
password := flag.String("password", "", "password for load test bidders")
```

Validasi:

```go
if *baseURL == "" || *auctionID == "" {
	log.Fatal("-base-url and -auction-id are required")
}
if strings.Contains(*baseURL, "prod") {
	log.Fatal("refusing to run against production")
}
```

### Langkah load test

1. Parse auction UUID dengan `uuid.Parse`.
2. Ambil detail auction untuk mendapatkan `current_price` dan `min_increment`.
3. Buat bidder:
   - login ke `/api/auth/login`;
   - jika akun belum ada, register ke `/api/auth/register`;
   - gunakan `http.CookieJar` per bidder.
4. Buka `viewers` koneksi WebSocket guest.
5. Setiap viewer mencatat waktu terima event `bid`.
6. Kirim bid lewat REST dengan rate yang ditentukan.
7. Cocokkan event `bid` dengan bid yang dikirim.
8. Hitung:
   - p50/p95/p99 broadcast latency;
   - accepted bids;
   - rejected bids;
   - errors.
9. Cetak ringkasan.

Jangan mengklaim angka performa sebelum benar-benar dijalankan.

Batasan awal:

- rate limit register dan bid dapat membuat load test kecil gagal;
- gunakan `-bidders` kecil, misalnya 3 sampai 10;
- load test hanya untuk local atau staging.

---

## 9. Runbook

Update `server/README.md`.

Ganti deskripsi “masih kerangka” dengan status aktual, lalu tambahkan:

```text
1. Prasyarat
2. Environment variables
3. PostgreSQL
4. Migrations
5. Seed
6. Menjalankan API
7. Menjalankan test
8. Menjalankan load test
9. Catatan cookie dan refresh token
```

Contoh isi:

````markdown
## Menjalankan API

Jalankan dari folder `server` agar `.env` ditemukan.

```powershell
cd server
go run ./cmd/api
```

## Menjalankan seed

```powershell
cd server
go run ./cmd/seed
```

## Menjalankan test

```powershell
$env:TEST_DATABASE_URL="postgres://postgres:root@localhost:5432/live_auction_test?sslmode=disable"
go test ./...
```
````

Tambahkan catatan:

```text
Refresh token disimpan di cookie httpOnly.
Jangan menyimpan refresh token di localStorage.
```

---

## 10. Test checklist

### Logging

- [ ] request ID dibuat dan dikirim di header.
- [ ] log memuat method, path, status, duration_ms.
- [ ] log memuat user_id jika ada principal.
- [ ] log memuat auction_id jika path berisi UUID.
- [ ] log tidak memuat email, cookie, token, atau body.
- [ ] WebSocket masih bisa upgrade saat logging aktif.

### Metrics

- [ ] `/metrics` mengembalikan 200.
- [ ] metric bid, auth, broadcast, connected clients, dan live auctions tersedia.
- [ ] metrics nil-safe saat controller diuji tanpa metrics.
- [ ] registry tidak memakai global default.

### Health

- [ ] `/healthz` 200 saat ping sukses.
- [ ] `/healthz` 503 saat ping gagal.
- [ ] response tidak membocorkan error database.

### Seed

- [ ] seed membuat admin.
- [ ] seed membuat bidder.
- [ ] seed membuat live demo.
- [ ] seed membuat draft demo.
- [ ] seed kedua tidak membuat data ganda.
- [ ] seed tidak membuat dua live auction.

### Load test

- [ ] flag tervalidasi.
- [ ] menolak URL yang terlihat seperti production.
- [ ] jumlah viewer dan bidder sesuai flag.
- [ ] ringkasan latency dicetak.

### Runbook

- [ ] perintah dijalankan dari folder `server`.
- [ ] migration dijelaskan.
- [ ] test database dijelaskan.
- [ ] cookie httpOnly dijelaskan.

---

## 11. Verifikasi

```powershell
cd D:\live-auction\server

$env:TEST_DATABASE_URL="postgres://postgres:root@localhost:5432/live_auction_test?sslmode=disable"

go test -count=1 ./...
go build ./...
go vet ./...
```

Jika CGO dan race detector tersedia:

```powershell
go test -race ./...
```

Jalankan seed dua kali:

```powershell
go run ./cmd/seed
go run ./cmd/seed
```

Pastikan jumlah data tidak bertambah.

Jalankan API dan cek:

```powershell
Invoke-RestMethod http://localhost:8080/healthz
Invoke-WebRequest http://localhost:8080/metrics
```

---

## 12. Urutan pengerjaan

```text
1. Logging middleware + test
2. Health check + test
3. Metrics package + test
4. Instrumentasi BidController, AuthController, RoomManager, AuctionService
5. Route /metrics + wiring main
6. Seed package + command + test
7. Load test
8. README runbook
9. Verifikasi penuh
10. Commit
```

Commit message yang disarankan:

```text
feat(server): add observability seed loadtest and runbook
```
