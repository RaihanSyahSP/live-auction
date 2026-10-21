# Auction Backend: Implementation Design

Status: Draft untuk review
Tanggal: 2026-10-20
Tujuan: panduan implementasi backend Go untuk LELANG
Eksekusi: manual oleh pemilik proyek

Dokumen terkait:

- `docs/superpowers/specs/2026-10-20-live-auction-design.md`: desain produk dan teknis utama.
- `docs/prd/2026-10-20-live-auction-prd.md`: requirement produk.
- `server/`: kerangka package yang sudah dibuat.
- `D:\belajar-golang\golang-restful`: referensi struktur layer.

## 1. Scope

Backend ini menangani:

- Auth dengan access token dan refresh token.
- Register, login, logout, refresh, dan `me`.
- Daftar dan detail lelang.
- Bid dengan validasi server, idempotency, dan anti-sniping.
- Admin create, preview-support, start, end, delete draft, dan list.
- WebSocket untuk snapshot dan event realtime.
- Scheduler penutupan lelang.
- Health check, structured log, dan metrics.

Di luar scope:

- Payment, ongkir, KYC, chat, email notification.
- Multi-live auction.
- Redis dan horizontal scaling.
- Object storage untuk foto.
- Refresh token di localStorage.
- OAuth atau login pihak ketiga.

## 2. Keputusan teknis

| Area | Keputusan |
|---|---|
| Bahasa | Go 1.26 |
| Router | `net/http` dengan pattern method dan path |
| Database | PostgreSQL 18, `pgx/v5` dan `pgxpool` |
| ID | UUIDv7, default `uuidv7()`, semua primary key dan foreign key `uuid` |
| Migration | `golang-migrate` |
| WebSocket | `github.com/coder/websocket` |
| Validasi request | `github.com/go-playground/validator/v10` |
| Password | bcrypt, cost 12 |
| Access token | JWT HS256, 15 menit |
| Refresh token | Opaque random 32 byte, 7 hari, rotasi |
| Cookie | `httpOnly`, `SameSite=Lax`, `Secure` di production |
| Logging | `log/slog` JSON |
| Metrics | `prometheus/client_golang` |
| Rate limit | `golang.org/x/time/rate` |
| DI | Manual wiring di `cmd/api/main.go` |
| ORM | Tidak ada. SQL eksplisit |

Referensi `golang-restful` dipakai untuk layer controller, service, repository, model, middleware, exception, dan helper. Perbedaannya:

- Tidak memakai Wire.
- Tidak memakai `httprouter`.
- Tidak memakai `lib/pq`.
- Tidak memakai panic/recover sebagai alur error.
- Tidak memakai ORM.

## 3. Struktur package

```text
server/
  cmd/
    api/main.go                  wiring dan start server
    seed/main.go                 seed admin, bidder, dan lelang demo
  internal/
    config/                      load dan validasi environment
    app/                         server, router, database pool, validator, scheduler
    auth/                        access token, refresh token, cookie
    controller/                  HTTP handler per resource
    service/                     aturan bisnis
    repository/                  query PostgreSQL dan transaksi
    model/
      domain/                    User, Auction, Bid, RefreshToken
      web/                       request dan response DTO
    middleware/                  auth, admin, origin, logging, recovery, rate limit
    exception/                   error domain dan mapping HTTP
    helper/                      JSON, parsing, test helper
    realtime/                    room manager, client, event envelope
    auction/                     aturan lelang dan lifecycle room
  migrations/
    README.md
  .env.example
  go.mod
  README.md
```

Tanggung jawab:

- `controller`: decode request, panggil service, tulis response. Tidak ada SQL.
- `service`: aturan bisnis, transaksi lewat repository, token lewat `auth`.
- `repository`: query SQL dan row lock. Tidak ada HTTP.
- `auth`: membuat dan memvalidasi JWT, membuat dan menghash refresh token, set dan clear cookie.
- `auction`: aturan minimum bid, anti-sniping, status transition, dan helper room.
- `realtime`: client WebSocket, room fan-out, snapshot, sequence, epoch.
- `app`: wiring HTTP server, router, database pool, validator, scheduler.
- `exception`: error domain yang dipetakan ke status HTTP.

## 4. Skema database

```sql
create table users (
  id            uuid primary key default uuidv7(),
  email         text not null,
  password_hash text not null,
  display_name  text not null,
  role          text not null default 'user' check (role in ('user', 'admin')),
  created_at    timestamptz not null default now()
);

create unique index users_email_lower_idx on users (lower(email));

create table auctions (
  id                uuid primary key default uuidv7(),
  created_by        uuid not null references users(id),
  lot               text not null unique,
  title             text not null,
  subtitle          text not null default '',
  description       text not null,
  condition         text not null,
  includes          text[] not null default '{}',
  specs             jsonb not null default '[]',
  image_urls        text[] not null default '{}',
  starting_price    bigint not null check (starting_price > 0),
  min_increment     bigint not null check (min_increment > 0),
  duration_seconds  int not null check (duration_seconds between 60 and 3600),
  current_price     bigint not null check (current_price > 0),
  highest_bidder_id uuid references users(id),
  status            text not null default 'draft' check (status in ('draft', 'live', 'ended')),
  starts_at         timestamptz,
  ends_at           timestamptz,
  ended_at          timestamptz,
  extension_count   int not null default 0 check (extension_count between 0 and 5),
  version           bigint not null default 0,
  winner_id         uuid references users(id),
  final_price       bigint check (final_price is null or final_price > 0),
  created_at        timestamptz not null default now(),
  check (status <> 'draft' or (starts_at is null and ends_at is null)),
  check (status = 'draft' or (starts_at is not null and ends_at is not null)),
  check (status <> 'ended' or ended_at is not null),
  check (ends_at is null or starts_at is null or ends_at > starts_at)
);

create unique index one_live_auction
  on auctions ((true))
  where status = 'live';

create index auctions_status_created_idx on auctions (status, created_at desc);

create table bids (
  id              uuid primary key default uuidv7(),
  auction_id      uuid not null references auctions(id),
  user_id         uuid not null references users(id),
  amount          bigint not null check (amount > 0),
  idempotency_key uuid not null,
  created_at      timestamptz not null default now()
);

create unique index bids_idempotency_idx
  on bids (auction_id, user_id, idempotency_key);

create index bids_auction_idx on bids (auction_id, id desc);
create index bids_user_idx on bids (user_id, id desc);

create table refresh_tokens (
  id             uuid primary key default uuidv7(),
  user_id        uuid not null references users(id) on delete cascade,
  family_id      uuid not null,
  token_hash     bytea not null unique,
  expires_at     timestamptz not null,
  created_at     timestamptz not null default now(),
  rotated_at     timestamptz,
  replaced_by_id uuid references refresh_tokens(id),
  revoked_at     timestamptz
);

create index refresh_tokens_user_idx on refresh_tokens (user_id);
create index refresh_tokens_family_idx on refresh_tokens (family_id);
create index refresh_tokens_expires_idx on refresh_tokens (expires_at);
```

Catatan:

- Semua primary key dan foreign key memakai tipe native `uuid`, bukan `bigint`, `text`, atau `varchar(36)`.
- Default `uuidv7()` membutuhkan PostgreSQL 18. Jika memakai versi lebih lama, ganti default menjadi `gen_random_uuid()`.
- UUIDv7 dipilih karena berurutan menurut waktu. `order by id desc` pada bids masih mendekati urutan insert. Jika butuh urutan ketat, gunakan `created_at, id`.
- `current_price` adalah harga tertinggi yang sudah commit.
- `version` naik setiap perubahan state lelang dan dipakai untuk mengurutkan event realtime.
- `highest_bidder_id` dan `winner_id` memakai user ID, bukan nama.
- `bids` menyimpan idempotency key per auction dan per user.
- `refresh_tokens.token_hash` berisi SHA-256 dari token mentah. Token mentah tidak pernah disimpan.
- `family_id` menghubungkan token hasil rotasi. Jika terjadi reuse, seluruh family dicabut.

## 5. Auth

### 5.1 Access token

- Format JWT HS256.
- Umur 15 menit.
- Claims: `sub` berisi UUID user dalam bentuk string, `role`, `iat`, `exp`, `iss` = `live-auction`, `aud` = `live-auction-web`.
- Secret dari `JWT_SECRET`, minimal 32 byte.
- Disimpan di cookie `access_token`.
- Cookie: `Path=/`, `HttpOnly`, `SameSite=Lax`, `Secure` mengikuti environment.

### 5.2 Refresh token

- 32 byte acak, di-encode base64url tanpa padding.
- Umur 7 hari.
- Disimpan di cookie `refresh_token`.
- Cookie: `Path=/api/auth`, `HttpOnly`, `SameSite=Lax`, `Secure` mengikuti environment.
- Database hanya menyimpan SHA-256 dari token mentah.
- Setiap refresh membuat token baru dalam `family_id` yang sama.
- Token lama diisi `rotated_at` dan `replaced_by_id`.
- Reuse detection:
  - Jika token sudah dirotasi dan masih dalam grace period 10 detik, kembalikan `409 REFRESH_RACE` tanpa mencabut family.
  - Jika reuse terjadi setelah grace period, cabut seluruh family dan kembalikan `401 REFRESH_REUSED`.
  - Jika token revoked atau expired, kembalikan `401 REFRESH_INVALID`.
- Client harus melakukan single-flight refresh agar dua tab tidak berebut token.
- Logout mencabut seluruh family, lalu menghapus kedua cookie.
- Access token yang sudah terlanjur diterbitkan tetap valid sampai 15 menit. Ini trade-off yang diterima untuk MVP.

### 5.3 Endpoint auth

| Method | Path | Keterangan |
|---|---|---|
| POST | `/api/auth/register` | Body email, password, display_name. Set dua cookie. |
| POST | `/api/auth/login` | Body email, password. Set dua cookie. |
| POST | `/api/auth/refresh` | Baca refresh cookie, rotasi token, set dua cookie. |
| POST | `/api/auth/logout` | Cabut family refresh, hapus dua cookie. |
| GET | `/api/auth/me` | Validasi access token, kembalikan user. |

Cookie yang dihapus harus memakai path yang sama saat dibuat. Jika tidak, browser tidak menghapusnya.

### 5.4 Otorisasi

- Register publik selalu menghasilkan role `user`.
- Admin dibuat lewat `cmd/seed`.
- Endpoint `/api/admin/*` memerlukan role `admin`.
- Pembuat lelang tidak boleh bid pada lelangnya sendiri.
- User harus login untuk bid.
- Guest boleh membuka halaman lelang dan WebSocket read-only.

### 5.5 Rate limit

- Login: 10 percobaan per menit per IP.
- Refresh: 30 percobaan per menit per IP.
- Bid: 3 bid per detik dengan burst 5 per user.
- Rate limit disimpan di memori. Untuk multi-instance, pindahkan ke Redis.

### 5.6 CSRF dan Origin

- Semua request yang mengubah state memeriksa header `Origin` terhadap `APP_ORIGIN`.
- Origin kosong diizinkan untuk API client dan curl. Browser modern mengirim Origin pada POST.
- `SameSite=Lax` menjadi lapisan kedua.
- Request JSON wajib `Content-Type: application/json`.
- WebSocket memeriksa Origin saat handshake.
- Tidak ada token CSRF terpisah di MVP.

## 6. REST API

Success response mengembalikan payload langsung. Error memakai envelope:

```json
{
  "error": {
    "code": "BID_TOO_LOW",
    "message": "Bid minimal Rp7.300.000",
    "details": { "minimum": 7300000 }
  }
}
```

### 6.1 Publik

| Method | Path | Keterangan |
|---|---|---|
| GET | `/api/auctions/current` | Lelang live, 50 bid terbaru, server_time. 204 jika tidak ada. |
| GET | `/api/auctions/{id}` | Detail lelang dan 50 bid terbaru. |
| GET | `/api/auctions/{id}/bids?before_id=&limit=50` | Riwayat lebih lama. |
| GET | `/api/results?limit=20` | Daftar lelang ended, pemenang, harga akhir. |
| POST | `/api/auctions/{id}/bids` | Body amount dan idempotency_key. 201 atau 200 untuk replay. |

Semua `{id}`, `before_id`, dan ID di response adalah UUID string. Format UUID yang tidak valid menghasilkan `VALIDATION_ERROR` 422.

### 6.2 Admin

| Method | Path | Keterangan |
|---|---|---|
| GET | `/api/admin/auctions` | Semua lelang, terbaru lebih dulu. |
| POST | `/api/admin/auctions` | Buat draft. |
| POST | `/api/admin/auctions/{id}/start` | Draft menjadi live. |
| POST | `/api/admin/auctions/{id}/end` | End lebih awal. |
| DELETE | `/api/admin/auctions/{id}` | Hapus draft. |

Body create draft:

```json
{
  "title": "Fujifilm X-T2",
  "subtitle": "Body only",
  "description": "Mirrorless APS-C 24 MP.",
  "condition": "Used, 7/10",
  "includes": ["Body", "Baterai", "Charger"],
  "specs": [
    { "label": "Sensor", "value": "APS-C 24 MP" },
    { "label": "Mount", "value": "Fujifilm X" }
  ],
  "image_urls": [],
  "starting_price": 8500000,
  "min_increment": 100000,
  "duration_minutes": 10
}
```

### 6.3 Operasional

| Method | Path | Keterangan |
|---|---|---|
| GET | `/healthz` | Proses dan ping database. |
| GET | `/metrics` | Prometheus. |
| GET | `/ws/auctions/{id}` | WebSocket. |

## 7. WebSocket

Endpoint: `GET /ws/auctions/{id}`

Auth opsional:

- Jika cookie access token valid, client terhubung sebagai user.
- Jika tidak ada atau expired, client tetap terhubung sebagai guest.
- Bid tetap lewat REST dan memerlukan auth.

Snapshot:

```json
{
  "type": "snapshot",
  "epoch": "uuid-per-room",
  "seq": 42,
  "version": 17,
  "server_time": "2026-10-20T10:00:00Z",
  "auction": { "...": "..." },
  "bids": [],
  "viewer_count": 12
}
```

Event:

```json
{
  "type": "bid",
  "auction_id": "018f4c8e-6c2a-7d0e-9b1a-2f3c4d5e6f70",
  "epoch": "uuid",
  "seq": 43,
  "version": 18,
  "server_time": "2026-10-20T10:00:05Z",
  "payload": { "...": "..." }
}
```

Tipe event:

- `bid`: semua client.
- `outbid`: hanya koneksi bidder sebelumnya, berdasarkan user ID.
- `presence`: semua client, debounce maksimal 1 per detik.
- `extended`: semua client.
- `ended`: semua client.

Aturan:

- `auction_id` adalah UUID string.
- `seq` naik per event room.
- `version` berasal dari kolom `auctions.version`.
- Room mengabaikan event dengan `version` lebih kecil atau sama dengan versi terakhir.
- Auction ID yang malformed atau tidak ada ditutup dengan close code 4004.
- Reconnect selalu meminta snapshot baru.
- Heartbeat ping setiap 20 detik, pong timeout 10 detik.
- Client reconnect dengan exponential backoff 0.5 detik sampai 8 detik.
- Server tidak memutar ulang event lama.

## 8. Alur bid

1. Request masuk ke `BidController` dengan access token.
2. Middleware auth mengisi user di context.
3. Rate limit per user diperiksa.
4. `BidService.PlaceBid` memulai transaksi.
5. `BidRepository.LockAuction` menjalankan `SELECT ... FOR UPDATE`.
6. Validasi:
   - status harus `live`;
   - `now < ends_at`;
   - user bukan pembuat lelang;
   - user bukan highest bidder saat ini;
   - amount integer, lebih dari 0, maksimal 1 triliun;
   - amount minimal `current_price + min_increment`, atau `starting_price` jika belum ada bid.
7. Cek idempotency:
   - Cari bid dengan `(auction_id, user_id, idempotency_key)`.
   - Jika ada dan amount sama, kembalikan bid lama dengan flag replay.
   - Jika ada dan amount berbeda, kembalikan `IDEMPOTENCY_CONFLICT`.
8. Insert bid.
9. Update auction:
   - `current_price = amount`;
   - `highest_bidder_id = user_id`;
   - `version = version + 1`;
   - jika dalam 60 detik terakhir dan `extension_count < 5`, `ends_at = ends_at + 60 detik`, `extension_count = extension_count + 1`.
10. Commit.
11. Kirim hasil ke room manager untuk broadcast.
12. Return 201. Replay return 200.

Jika insert kena unique violation karena race idempotency, baca bid yang sudah ada dan kembalikan replay atau konflik.

`SELECT FOR UPDATE` adalah titik serialisasi bid. Actor room tidak melakukan validasi harga.

## 9. Room manager dan scheduler

- `realtime.RoomManager` menyimpan satu room per auction yang sedang live.
- Room menyimpan client, `seq`, `epoch`, `viewer_count`, dan versi terakhir.
- Room menerima event yang sudah commit dari service.
- Room mengabaikan event dengan `version` lama.
- Saat client join, room mengirim snapshot dari state terbaru.
- Saat room kosong, room tetap hidup sampai lelang ended. Ini menjaga timer dan state.
- `app.Scheduler` berjalan setiap 1 detik:
  - Cari lelang live yang `ends_at <= now`.
  - Panggil `AuctionService.EndAuction` secara idempoten.
  - Broadcast `ended`.
- Saat boot:
  - Muat semua lelang live.
  - Jika `ends_at` sudah lewat, end segera.
  - Buat room untuk lelang yang masih live.
- Admin end memakai service yang sama dan idempoten.

## 10. Error

| Code | HTTP | Kapan |
|---|---|---|
| `VALIDATION_ERROR` | 422 | Body atau field tidak valid. |
| `UNAUTHENTICATED` | 401 | Tidak ada access token. |
| `TOKEN_EXPIRED` | 401 | Access token expired. Client harus refresh. |
| `FORBIDDEN` | 403 | Role tidak cukup. |
| `CREATOR_CANNOT_BID` | 403 | Pembuat lelang mencoba bid. |
| `AUCTION_NOT_FOUND` | 404 | ID tidak ada. |
| `AUCTION_NOT_LIVE` | 409 | Lelang belum live atau sudah ended. |
| `AUCTION_NOT_DRAFT` | 409 | Start atau delete pada bukan draft. |
| `AUCTION_ALREADY_LIVE` | 409 | Masih ada lelang live lain. |
| `BID_TOO_LOW` | 422 | Nominal di bawah minimum. |
| `ALREADY_HIGHEST` | 409 | User sudah memegang harga tertinggi. |
| `INVALID_AMOUNT` | 422 | Nominal bukan integer atau di luar batas. |
| `IDEMPOTENCY_CONFLICT` | 409 | Key sama, payload berbeda. |
| `RATE_LIMITED` | 429 | Melebihi rate limit. |
| `REFRESH_INVALID` | 401 | Refresh token tidak valid, expired, atau revoked. |
| `REFRESH_REUSED` | 401 | Refresh token dipakai ulang di luar grace period. |
| `REFRESH_RACE` | 409 | Refresh bersamaan dalam grace period. Client retry. |
| `INTERNAL` | 500 | Kesalahan tak terduga. |

Pesan user-facing memakai bahasa Indonesia. Detail internal tidak dibocorkan.

## 11. Config

| Variable | Default | Wajib |
|---|---|---|
| `PORT` | `8080` | Tidak |
| `DATABASE_URL` | none | Ya |
| `APP_ORIGIN` | none | Ya |
| `JWT_SECRET` | none | Ya, minimal 32 byte |
| `ACCESS_TOKEN_TTL` | `15m` | Tidak |
| `REFRESH_TOKEN_TTL` | `168h` | Tidak |
| `REFRESH_GRACE_PERIOD` | `10s` | Tidak |
| `COOKIE_SECURE` | `false` | Tidak |
| `SEED_ADMIN_EMAIL` | none | Untuk seed |
| `SEED_ADMIN_PASSWORD` | none | Untuk seed |
| `SEED_BIDDER_PASSWORD` | none | Untuk seed |

Config divalidasi saat startup. Server gagal start jika nilai wajib kosong.

## 12. Testing

Unit:

- Aturan bid: bid pertama, increment, highest bidder, creator.
- Anti-sniping: batas 60 detik dan cap 5.
- Access token: valid, expired, signature salah, role.
- Refresh token: hash, expiry, rotasi, reuse, grace race.
- Error mapping.

Integration dengan Postgres asli:

- Constraint satu lelang live.
- Unique email lowercase.
- Idempotency per auction.
- Dua bid bersamaan pada nominal sama: satu diterima.
- Bid setelah `ends_at` ditolak.
- Anti-sniping mengubah `ends_at` dan `extension_count`.
- Login, refresh rotation, reuse detection, logout.
- Restart recovery: lelang live dimuat dan due auction ditutup.

WebSocket:

- Dua client menerima event yang sama.
- Outbid hanya ke koneksi bidder sebelumnya.
- Reconnect menerima snapshot.
- Event dengan version lama diabaikan.

Perintah:

```powershell
go test -race ./...
```

Playwright dan load test dikerjakan setelah backend berjalan.

## 13. Observability

- `slog` JSON.
- Field log: request_id, user_id, auction_id, bid_id, duration_ms, error_code.
- Jangan log password, token, cookie, atau email mentah.
- Metrics:
  - `auction_connected_clients{auction_id}`
  - `auction_bids_total{result}`
  - `auction_bid_duration_seconds`
  - `auction_broadcast_duration_seconds`
  - `auction_live_auctions`
  - `auth_refresh_total{result}`
  - `auth_login_total{result}`
- `/healthz` melakukan `SELECT 1`.

## 14. Migrations dan run

```powershell
docker compose up -d postgres
go run ./cmd/api
go run ./cmd/seed
```

Migration:

```powershell
migrate -path migrations -database "$env:DATABASE_URL" up
```

`cmd/api` tidak menjalankan migration otomatis di production. Untuk development, migration boleh dijalankan saat startup jika `RUN_MIGRATIONS=true`.

## 15. Global constraints

- Go 1.26.
- PostgreSQL 18 untuk default `uuidv7()`.
- Tanpa ORM, Wire, httprouter, atau lib/pq.
- Semua primary key dan foreign key bertipe `uuid`. Tidak ada ID `bigint`.
- Semua harga integer rupiah.
- Semua timestamp UTC.
- Access token 15 menit, refresh token 7 hari.
- Refresh token hanya di cookie `httpOnly`.
- Refresh token di database hanya hash.
- Satu lelang live pada satu waktu.
- Server adalah sumber kebenaran waktu, harga, dan pemenang.
- Semua endpoint yang mengubah state memeriksa Origin.
- Tidak ada secret di log.
- Semua query memakai parameter pgx.
- Setiap task plan harus lulus `go test -race ./...` sebelum commit.

## 16. Out of scope

- Refresh token di localStorage.
- Token revocation instan untuk access token.
- OAuth.
- Redis pub/sub.
- Multi-instance WebSocket.
- Multi-live auction.
- Payment, KYC, ongkir, chat.
