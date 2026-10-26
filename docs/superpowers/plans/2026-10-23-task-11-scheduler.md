# Task 11: Scheduler dan Restart Recovery

> **Untuk implementasi manual:** panduan ini menjelaskan flow, keputusan desain, kode, dan test yang perlu dibuat. Kerjakan per langkah, jalankan test setelah setiap perubahan, lalu commit.

**Goal:** Lelang berhenti otomatis saat `ends_at` lewat, termasuk saat server baru hidup kembali setelah restart.

**Architecture:** `app.Scheduler` berjalan setiap satu detik, memanggil `AuctionService.CloseDue`, lalu mem-publish event `ended` ke `RoomManager`. `AuctionService.CloseDue` tetap memakai transaksi dan row lock yang sama dengan admin `End`, sehingga admin dan scheduler tidak membuat pemenang ganda.

**Tech Stack:** Go, `net/http`, `pgx/v5`, PostgreSQL, `log/slog`, `realtime.RoomManager`.

**Spec:** `docs/superpowers/specs/2026-10-20-auction-backend-design.md` bagian 9.

---

## Global Constraints

- Tidak memakai ORM atau framework router.
- Semua ID memakai `uuid.UUID` dan UUIDv7 dari PostgreSQL.
- Uang memakai `int64`, bukan `float64`.
- Sumber kebenaran tetap PostgreSQL.
- Event `ended` hanya dikirim untuk perubahan state yang valid.
- Error scheduler dicatat, tetapi ticker tetap berjalan.
- Startup recovery harus selesai sebelum server menerima request.

---

## Review Focus

- Lelang yang sudah lewat waktu harus berakhir walaupun tidak ada admin yang menekan End.
- `CloseDue` yang dipanggil berulang tidak boleh mengganti pemenang atau membuat pemenang kedua.
- Dua pemanggil End, admin dan scheduler, harus menghasilkan satu transisi final.
- Event `ended` tidak boleh dikirim dua kali untuk versi auction yang sama.
- Kegagalan database saat startup recovery tidak boleh diam-diam diabaikan.

---

## 1. Flow

### 1.1 Startup recovery

```mermaid
sequenceDiagram
    autonumber
    participant M as main
    participant S as Scheduler
    participant AS as AuctionService
    participant DB as PostgreSQL
    participant RM as RoomManager

    M->>S: NewScheduler(...)
    M->>S: CloseDue(startupCtx)
    S->>AS: CloseDue(ctx)
    AS->>DB: ListDue(now)
    DB-->>AS: auction live dengan ends_at <= now
    loop setiap auction
        AS->>DB: End dalam transaksi + row lock
        DB-->>AS: auction ended
    end
    AS-->>S: []AuctionResponse
    S->>RM: PublishEnded(auction)
    RM-->>S: event ended ke room yang ada
    M->>S: go Run(ctx)
```

Startup recovery penting karena server bisa mati saat ada lelang yang sedang berjalan. Saat server hidup lagi, lelang yang sudah lewat waktu tidak boleh tetap berstatus live.

### 1.2 Ticker scheduler

```mermaid
flowchart TD
    A[Run ctx] --> B[ticker setiap 1 detik]
    B --> C[CloseDue]
    C --> D{error?}
    D -->|ya| E[log error]
    E --> B
    D -->|tidak| B
    B -->|ctx.Done| F[return]
```

Scheduler tidak berhenti hanya karena satu tick gagal. Error dicatat, lalu tick berikutnya tetap berjalan.

### 1.3 Admin dan scheduler berebut lelang yang sama

```mermaid
sequenceDiagram
    autonumber
    participant A as Admin
    participant S as Scheduler
    participant DB as PostgreSQL
    participant R as Room

    A->>DB: End, lock row
    S->>DB: End, menunggu lock
    DB-->>A: ended versi 2
    A->>R: PublishEnded versi 2
    DB-->>S: row sudah ended versi 2
    S->>R: PublishEnded versi 2
    R-->>R: versi <= versi room, abaikan
```

`AuctionService.End` harus idempoten. Scheduler boleh menerima response yang sama, tetapi `RoomManager` harus mengabaikan event dengan versi yang sudah pernah diproses.

### 1.4 Mengapa event ended dicek versinya

Tanpa pengecekan versi:

```text
Admin End -> publish ended versi 2
Scheduler  -> publish ended versi 2 lagi
Client     -> menerima ended dua kali
```

Dengan pengecekan versi:

```text
Admin End -> publish ended versi 2
Scheduler  -> versi 2 <= room version 2
           -> tidak mengirim event
```

---

## 2. File yang berubah

```text
server/internal/realtime/manager.go
server/internal/realtime/manager_test.go

server/internal/service/auction_service.go
server/internal/service/auction_service_test.go

server/internal/app/scheduler.go
server/internal/app/scheduler_test.go

server/internal/controller/auction_controller_test.go
server/cmd/api/main.go
```

---

## 3. Interface yang dipakai

### 3.1 Interface service setelah perubahan

```go
type AuctionService interface {
	Current(ctx context.Context) (auction_response.AuctionDetailResponse, error)
	Detail(ctx context.Context, id uuid.UUID) (auction_response.AuctionDetailResponse, error)
	Bids(ctx context.Context, auctionID uuid.UUID, beforeID *uuid.UUID, limit int) ([]auction_response.BidResponse, error)
	Results(ctx context.Context, limit int) ([]auction_response.ResultResponse, error)
	List(ctx context.Context, limit int, offset int) ([]auction_response.AuctionResponse, error)
	CreateDraft(ctx context.Context, adminID uuid.UUID, req auction_request.CreateAuctionRequest) (auction_response.AuctionResponse, error)
	Start(ctx context.Context, adminID uuid.UUID, id uuid.UUID) (auction_response.AuctionResponse, error)
	End(ctx context.Context, adminID uuid.UUID, id uuid.UUID) (auction_response.AuctionResponse, error)
	DeleteDraft(ctx context.Context, adminID uuid.UUID, id uuid.UUID) error
	CloseDue(ctx context.Context) ([]auction_response.AuctionResponse, error)
}
```

Perubahan penting:

```go
// Sebelum
CloseDue(ctx context.Context) ([]domain.Auction, error)

// Sesudah
CloseDue(ctx context.Context) ([]auction_response.AuctionResponse, error)
```

Alasannya: scheduler membutuhkan response yang siap dikirim ke `RoomManager.PublishEnded`, termasuk `version`, `winner_id`, dan `final_price`.

### 3.2 Interface scheduler

```go
type Scheduler struct {
	auctions service.AuctionService
	rooms    *realtime.RoomManager
	interval time.Duration
	now      func() time.Time
}

func NewScheduler(
	auctions service.AuctionService,
	rooms *realtime.RoomManager,
	interval time.Duration,
	now func() time.Time,
) *Scheduler

func (s *Scheduler) Run(ctx context.Context)
func (s *Scheduler) CloseDue(ctx context.Context) error
```

`now` dipakai untuk timestamp log scheduler. Clock bisnis tetap dimiliki `AuctionService`.

---

## 4. Implementasi

### 4.1 Tambahkan version guard di `RoomManager.PublishEnded`

Buka `server/internal/realtime/manager.go`.

Saat ini:

```go
func (m *RoomManager) PublishEnded(auction auction_response.AuctionResponse) {
	m.mu.Lock()
	room := m.rooms[auction.ID]
	m.mu.Unlock()
	if room == nil {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	room.version = auction.Version
	room.auction = auction

	event := room.nextEvent("ended", auction.ID, map[string]any{
		"auction": auction,
	})
	room.broadcast(event)
}
```

Tambahkan pengecekan setelah `room.mu.Lock()`:

```go
if auction.Version <= room.version {
	return
}
```

Hasil akhir:

```go
func (m *RoomManager) PublishEnded(auction auction_response.AuctionResponse) {
	m.mu.Lock()
	room := m.rooms[auction.ID]
	m.mu.Unlock()
	if room == nil {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if auction.Version <= room.version {
		return
	}

	room.version = auction.Version
	room.auction = auction

	event := room.nextEvent("ended", auction.ID, map[string]any{
		"auction": auction,
	})
	room.broadcast(event)
}
```

Kenapa aman?

- Admin `End` menaikkan `version`.
- Scheduler `End` juga menaikkan `version`.
- Jika keduanya memproses auction yang sama, hanya versi baru pertama yang dikirim.
- Jika tidak ada room, tidak ada client yang perlu dikirimi event.

---

### 4.2 Ubah `AuctionService.CloseDue`

Buka `server/internal/service/auction_service.go`.

Ubah interface:

```go
CloseDue(ctx context.Context) ([]auction_response.AuctionResponse, error)
```

Ubah implementasi:

```go
func (s *auctionService) CloseDue(ctx context.Context) ([]auction_response.AuctionResponse, error) {
	var due []domain.Auction

	if err := s.tx.WithinTx(ctx, func(tx pgx.Tx) error {
		auctions, err := s.auctions.ListDue(ctx, tx, s.now().UTC())
		if err != nil {
			return err
		}
		due = auctions
		return nil
	}); err != nil {
		return nil, err
	}

	var ended []auction_response.AuctionResponse
	for _, auction := range due {
		updated, err := s.End(ctx, uuid.Nil, auction.ID)
		if err != nil {
			return nil, err
		}
		ended = append(ended, updated)
	}

	return ended, nil
}
```

Penjelasan:

1. `ListDue` mencari auction live yang `ends_at <= now`.
2. Setiap auction diproses lewat `End`.
3. `End` memakai `LockByID` dan transaksi sendiri.
4. Response yang dikembalikan sudah berisi state ended, bukan state lama.
5. `uuid.Nil` dipakai karena scheduler bukan user admin.

`End` tetap idempoten:

```text
status live  -> update ke ended, version++
status ended -> return state ended yang sudah ada
status lain  -> error
```

Jadi admin dan scheduler tidak membuat dua pemenang.

---

### 4.3 Buat `server/internal/app/scheduler.go`

```go
package app

import (
	"context"
	"log/slog"
	"time"

	"live-auction/server/internal/model/web/auction_response"
	"live-auction/server/internal/realtime"
	"live-auction/server/internal/service"
)

type Scheduler struct {
	auctions service.AuctionService
	rooms    *realtime.RoomManager
	interval time.Duration
	now      func() time.Time
}

func NewScheduler(
	auctions service.AuctionService,
	rooms *realtime.RoomManager,
	interval time.Duration,
	now func() time.Time,
) *Scheduler {
	if interval <= 0 {
		interval = time.Second
	}
	if now == nil {
		now = time.Now
	}

	return &Scheduler{
		auctions: auctions,
		rooms:    rooms,
		interval: interval,
		now:      now,
	}
}

func (s *Scheduler) CloseDue(ctx context.Context) error {
	ended, err := s.auctions.CloseDue(ctx)
	if err != nil {
		return err
	}

	for _, auction := range ended {
		s.rooms.PublishEnded(auction)
	}

	if len(ended) > 0 {
		slog.Info(
			"scheduler closed due auctions",
			"count", len(ended),
			"at", s.now().UTC(),
		)
	}

	return nil
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.CloseDue(ctx); err != nil {
				slog.Error("scheduler close due", "error", err)
			}
		}
	}
}
```

Perhatikan:

- `CloseDue` tidak menelan error.
- `Run` mencatat error dan tetap lanjut.
- `PublishEnded` hanya mengirim ke room yang sudah ada.
- Tidak ada room baru yang dibuat di sini. Room tetap dibuat saat client join dari snapshot database.

---

### 4.4 Wiring di `server/cmd/api/main.go`

Pindahkan signal context ke awal, sebelum scheduler dipakai:

```go
ctx, stop := signal.NotifyContext(
	context.Background(),
	os.Interrupt,
	syscall.SIGTERM,
)
defer stop()
```

Setelah `rooms` dibuat:

```go
scheduler := app.NewScheduler(
	auctionService,
	rooms,
	time.Second,
	time.Now,
)

startupCtx, cancelStartup := context.WithTimeout(ctx, 10*time.Second)
if err := scheduler.CloseDue(startupCtx); err != nil {
	cancelStartup()
	slog.Error("close due auctions on startup", "error", err)
	os.Exit(1)
}
cancelStartup()

go scheduler.Run(ctx)
```

Urutan startup:

```text
1. load config
2. connect database
3. buat service
4. buat RoomManager
5. buat Scheduler
6. CloseDue sekali sebelum server menerima request
7. jalankan scheduler.Run(ctx) di goroutine
8. ListenAndServe
```

Saat shutdown:

```text
signal diterima
  |
  v
ctx dibatalkan
  |
  +--> scheduler.Run berhenti
  |
  v
server.Shutdown
```

---

## 5. Test

### 5.1 Test `RoomManager.PublishEnded` idempoten

Tambahkan di `server/internal/realtime/manager_test.go`:

```go
func TestRoomManagerPublishEndedIgnoresDuplicateVersion(t *testing.T) {
	now := testTime()
	m := newTestManager(&now)
	auctionID := uuid.New()

	client, _ := m.Join(auctionID, nil, Snapshot{
		Version: 1,
		Auction: testAuction(auctionID, 1),
	})
	drainClient(client)

	ended := testAuction(auctionID, 2)
	ended.Status = domain.AuctionStatusEnded

	m.PublishEnded(ended)
	if event := receiveEvent(t, client); event.Type != "ended" {
		t.Fatalf("event = %q, want ended", event.Type)
	}

	m.PublishEnded(ended)

	if len(client.Send) != 0 {
		t.Fatalf("duplicate ended event was published")
	}
}
```

---

### 5.2 Test service `CloseDue`

Tambahkan di `server/internal/service/auction_service_test.go`:

#### Lelang yang belum due tidak berubah

```go
func TestAuctionServiceCloseDueSkipsNotDue(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	f.start(t, draft.ID)

	ended, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("CloseDue() error = %v", err)
	}
	if len(ended) != 0 {
		t.Fatalf("CloseDue() length = %d, want 0", len(ended))
	}

	if _, err := f.svc.Current(context.Background()); err != nil {
		t.Fatalf("Current() error = %v", err)
	}
}
```

#### Lelang yang due mengembalikan state ended

```go
func TestAuctionServiceCloseDueReturnsEndedResponse(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	f.start(t, draft.ID)
	f.clock.Advance(2 * time.Minute)

	ended, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("CloseDue() error = %v", err)
	}
	if len(ended) != 1 {
		t.Fatalf("CloseDue() length = %d, want 1", len(ended))
	}
	if ended[0].Status != domain.AuctionStatusEnded {
		t.Errorf("Status = %q, want ended", ended[0].Status)
	}
	if ended[0].Version == 0 {
		t.Error("Version = 0, want incremented version")
	}
}
```

#### Closing dua kali tidak menghasilkan transisi kedua

```go
func TestAuctionServiceCloseDueTwice(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	f.start(t, draft.ID)
	f.clock.Advance(2 * time.Minute)

	first, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("first CloseDue() error = %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("first length = %d, want 1", len(first))
	}

	second, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("second CloseDue() error = %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second length = %d, want 0", len(second))
	}
}
```

#### Admin End lalu scheduler tidak membuat pemenang baru

```go
func TestAuctionServiceCloseDueAfterAdminEnd(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	started := f.start(t, draft.ID)
	bidderID := f.setHighestBid(t, started.ID, 1500)
	f.clock.Advance(2 * time.Minute)

	adminEnded, err := f.svc.End(context.Background(), f.adminID, started.ID)
	if err != nil {
		t.Fatalf("End() error = %v", err)
	}

	ended, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("CloseDue() error = %v", err)
	}
	if len(ended) != 0 {
		t.Fatalf("CloseDue() length = %d, want 0", len(ended))
	}

	if adminEnded.WinnerID == nil || *adminEnded.WinnerID != bidderID {
		t.Errorf("WinnerID = %v, want %v", adminEnded.WinnerID, bidderID)
	}
}
```

#### Restart recovery

Test ini sama dengan memanggil `CloseDue` setelah waktu dimajukan. Itu yang dilakukan startup recovery.

```go
func TestAuctionServiceCloseDueRecoversOverdueAuction(t *testing.T) {
	f := newAuctionFixture(t)
	draft := f.createDraft(t)
	f.start(t, draft.ID)

	// Simulate server restart after the auction has passed its end time.
	f.clock.Advance(2 * time.Minute)

	ended, err := f.svc.CloseDue(context.Background())
	if err != nil {
		t.Fatalf("CloseDue() error = %v", err)
	}
	if len(ended) != 1 {
		t.Fatalf("CloseDue() length = %d, want 1", len(ended))
	}
}
```

---

### 5.3 Test scheduler

Buat `server/internal/app/scheduler_test.go`.

Fake service dapat memakai embedding interface supaya tidak perlu menulis semua method:

```go
package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"live-auction/server/internal/app"
	"live-auction/server/internal/model/domain"
	"live-auction/server/internal/model/web/auction_response"
	"live-auction/server/internal/realtime"
	"live-auction/server/internal/service"
)

type fakeSchedulerService struct {
	service.AuctionService
	closeDueResult []auction_response.AuctionResponse
	closeDueErr    error
	closeDueCalls  int
}

func (f *fakeSchedulerService) CloseDue(
	ctx context.Context,
) ([]auction_response.AuctionResponse, error) {
	f.closeDueCalls++
	return f.closeDueResult, f.closeDueErr
}

func receiveSchedulerEvent(t *testing.T, client *realtime.Client) realtime.Event {
	t.Helper()

	select {
	case data, ok := <-client.Send:
		if !ok {
			t.Fatal("client channel closed")
		}

		var event realtime.Event
		if err := json.Unmarshal(data, &event); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return realtime.Event{}
	}
}
```

#### Scheduler mem-publish event ended

```go
func TestSchedulerCloseDuePublishesEnded(t *testing.T) {
	auctionID := uuid.New()
	rooms := realtime.NewRoomManager()
	client, _ := rooms.Join(auctionID, nil, realtime.Snapshot{
		Version: 1,
		Auction: auction_response.AuctionResponse{
			ID:      auctionID,
			Status:  domain.AuctionStatusLive,
			Version: 1,
		},
	})

	for len(client.Send) > 0 {
		<-client.Send
	}

	svc := &fakeSchedulerService{
		closeDueResult: []auction_response.AuctionResponse{{
			ID:      auctionID,
			Status:  domain.AuctionStatusEnded,
			Version: 2,
		}},
	}
	scheduler := app.NewScheduler(svc, rooms, time.Second, time.Now)

	if err := scheduler.CloseDue(context.Background()); err != nil {
		t.Fatalf("CloseDue() error = %v", err)
	}

	event := receiveSchedulerEvent(t, client)
	if event.Type != "ended" {
		t.Fatalf("event type = %q, want ended", event.Type)
	}
	if event.Version != 2 {
		t.Errorf("event version = %d, want 2", event.Version)
	}
}
```

#### Scheduler tidak mengirim ended dua kali

```go
func TestSchedulerCloseDueIgnoresDuplicateEnded(t *testing.T) {
	auctionID := uuid.New()
	rooms := realtime.NewRoomManager()
	client, _ := rooms.Join(auctionID, nil, realtime.Snapshot{
		Version: 1,
		Auction: auction_response.AuctionResponse{
			ID:      auctionID,
			Status:  domain.AuctionStatusLive,
			Version: 1,
		},
	})

	for len(client.Send) > 0 {
		<-client.Send
	}

	ended := auction_response.AuctionResponse{
		ID:      auctionID,
		Status:  domain.AuctionStatusEnded,
		Version: 2,
	}
	svc := &fakeSchedulerService{
		closeDueResult: []auction_response.AuctionResponse{ended},
	}
	scheduler := app.NewScheduler(svc, rooms, time.Second, time.Now)

	if err := scheduler.CloseDue(context.Background()); err != nil {
		t.Fatalf("first CloseDue() error = %v", err)
	}
	receiveSchedulerEvent(t, client)

	if err := scheduler.CloseDue(context.Background()); err != nil {
		t.Fatalf("second CloseDue() error = %v", err)
	}
	if len(client.Send) != 0 {
		t.Fatal("duplicate ended event was published")
	}
}
```

#### Error service diteruskan

```go
func TestSchedulerCloseDueReturnsServiceError(t *testing.T) {
	svc := &fakeSchedulerService{closeDueErr: errors.New("database down")}
	scheduler := app.NewScheduler(svc, realtime.NewRoomManager(), time.Second, time.Now)

	if err := scheduler.CloseDue(context.Background()); err == nil {
		t.Fatal("CloseDue() error = nil, want service error")
	}
}
```

#### `Run` berhenti saat context dibatalkan

```go
func TestSchedulerRunStopsOnContextCancel(t *testing.T) {
	svc := &fakeSchedulerService{}
	scheduler := app.NewScheduler(svc, realtime.NewRoomManager(), time.Hour, time.Now)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		scheduler.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run() did not stop after context cancellation")
	}
}
```

---

### 5.4 Sesuaikan fake service controller

Di `server/internal/controller/auction_controller_test.go`, ubah field dan method fake:

```go
closeDue []auction_response.AuctionResponse
```

```go
func (f *fakeAuctionService) CloseDue(
	ctx context.Context,
) ([]auction_response.AuctionResponse, error) {
	return f.closeDue, f.closeDueErr
}
```

Hapus import `domain` hanya jika masih dipakai untuk hal lain. `domain` masih dipakai untuk role admin, jadi biarkan.

---

## 6. Urutan pengerjaan

### Langkah 1: Test version guard

- [ ] Tambahkan test duplicate ended di `manager_test.go`.
- [ ] Jalankan test dan pastikan gagal karena event kedua masih dikirim.
- [ ] Tambahkan guard `auction.Version <= room.version`.
- [ ] Jalankan test dan pastikan lulus.

### Langkah 2: Ubah `CloseDue`

- [ ] Ubah return type interface menjadi `[]auction_response.AuctionResponse`.
- [ ] Ubah implementasi agar mengembalikan response dari `End`.
- [ ] Sesuaikan fake service di controller test.
- [ ] Tambahkan test service `CloseDue`.
- [ ] Jalankan test service.

### Langkah 3: Buat scheduler

- [ ] Buat `app/scheduler.go`.
- [ ] Buat `app/scheduler_test.go`.
- [ ] Jalankan test scheduler.

### Langkah 4: Wiring main

- [ ] Pindahkan signal context ke awal.
- [ ] Buat scheduler.
- [ ] Panggil `CloseDue` sekali sebelum server melayani request.
- [ ] Jalankan `go scheduler.Run(ctx)`.
- [ ] Pastikan shutdown membatalkan context.

### Langkah 5: Verifikasi

```powershell
cd D:\live-auction\server

$env:TEST_DATABASE_URL="postgres://postgres:root@localhost:5432/live_auction_test?sslmode=disable"

go test -count=1 ./internal/app ./internal/service ./internal/realtime
go test -count=1 ./...
go build ./...
go vet ./...
```

Jika ingin race detector dan CGO tersedia:

```powershell
go test -race ./internal/app ./internal/service ./internal/realtime
```

### Langkah 6: Commit

```powershell
git add server
git commit -m "feat(server): add lifecycle scheduler and restart recovery"
```

---

## 7. Checklist

- [ ] `PublishEnded` mengabaikan versi yang sudah diproses.
- [ ] `CloseDue` mengembalikan `[]AuctionResponse`.
- [ ] `CloseDue` memakai `End` yang idempoten.
- [ ] Scheduler mem-publish event ended untuk setiap response.
- [ ] Scheduler mencatat error dan tetap berjalan.
- [ ] Scheduler berhenti saat context dibatalkan.
- [ ] Startup menjalankan `CloseDue` sebelum `ListenAndServe`.
- [ ] Startup gagal dengan jelas jika recovery database gagal.
- [ ] Test duplicate ended lulus.
- [ ] Test not-due lulus.
- [ ] Test closing twice lulus.
- [ ] Test admin end lalu scheduler lulus.
- [ ] Test restart recovery lulus.
- [ ] `go test ./...` lulus.
- [ ] `go build ./...` lulus.
- [ ] `go vet ./...` lulus.

---

## 8. Ringkasan

```text
PostgreSQL
  |
  | ListDue(now)
  v
AuctionService.CloseDue
  |
  | End + row lock + idempotent check
  v
[]AuctionResponse
  |
  v
Scheduler.CloseDue
  |
  v
RoomManager.PublishEnded
  |
  | version guard
  v
WebSocket clients
```

Inti Task 11:

```text
Scheduler hanya memicu penutupan.
AuctionService tetap yang memutuskan transisi state.
RoomManager hanya menyebarkan event.
```
