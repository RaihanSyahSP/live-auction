# Live Auction: Desain Produk & Teknis

Tanggal: 2026-10-20
Status: Draft untuk review
Jenis: Proyek portofolio / pembelajaran

Catatan: prototipe UI saat ini ada di `prototype/` dan memakai HTML, CSS, serta JavaScript tanpa backend. Backend Go pada dokumen ini adalah target implementasi berikutnya.

## 1. Ringkasan

Aplikasi lelang real-time untuk barang elektronik bekas. Satu sesi lelang berlangsung pada satu waktu. Semua penonton menerima harga terbaru, riwayat bid, dan status sesi secara langsung melalui WebSocket. Server adalah satu-satunya sumber kebenaran: client mengirim niat bid, server memvalidasi, menyimpan, lalu menyiarkan hasilnya.

Stack:

- Backend: Go (REST + WebSocket), PostgreSQL.
- Frontend: React Router v7 framework mode dengan SSR, TypeScript, Tailwind CSS.
- Tanpa Redis, tanpa layanan realtime pihak ketiga.

## 2. Tujuan & kriteria sukses

Tujuan:

- Menunjukkan pemahaman konkurensi, konsistensi state, dan komunikasi real-time.
- Menghasilkan demo yang bisa dijalankan dan dijelaskan dalam wawancara.
- Menjaga scope tetap selesai, bukan platform lelang lengkap.

Kriteria sukses:

1. Dua browser dapat bid dan melihat harga yang sama secara real-time.
2. Tidak ada bid ganda, bid di bawah increment, atau bid setelah lelang berakhir yang diterima.
3. Lelang berakhir otomatis dengan pemenang dan harga final yang benar.
4. Client yang terputus dapat menyambung ulang dan menyelaraskan state tanpa kehilangan bid.
5. Ada bukti load test dan metrik, bukan hanya klaim.

## 3. Pengguna & peran

| Peran | Kemampuan |
|---|---|
| Guest | Melihat halaman lelang dan menerima update real-time. Tidak bisa bid. |
| User | Register, login, melihat lelang, memasang bid, menerima notifikasi outbid. |
| Admin | Membuat, memulai, dan mengakhiri lelang. Tidak boleh bid pada lelang buatannya. |

Admin dibuat melalui seed/CLI, bukan registrasi publik. Registrasi publik selalu menghasilkan role `user`.

## 4. Scope

### Termasuk

- Auth email + password, session cookie.
- Satu lelang live pada satu waktu, satu room per lelang.
- Bid real-time, validasi server, riwayat bid, presence, notifikasi outbid.
- Anti-sniping 60 detik.
- Reconnect + resync.
- Admin lifecycle: draft → live → ended.
- Metrics Prometheus, structured log, load test.

### Tidak termasuk (YAGNI)

- Payment gateway, deposit, KYC, ongkir.
- Auto-bid / proxy bidding, reserve price, buy-now.
- Katalog multi-kategori, multi-lelang bersamaan, watchlist.
- Chat, komentar, email notifikasi, push notification.
- Upload file/gambar (gambar memakai URL).
- Share link/OG image, dark mode, animasi pemenang.
- Mobile app, multi-currency, multi-bahasa.
- Redis, message broker, horizontal scaling.

## 5. Fitur

### 5.1 Auth & sesi

- Register: email, password, display name. Email unik.
- Login menghasilkan JWT yang disimpan di cookie `httpOnly`.
- `GET /api/auth/me` untuk hydrate sesi di SSR dan client.
- Logout menghapus cookie.
- Password di-hash dengan bcrypt cost 12.

### 5.2 Halaman lelang

- Judul, deskripsi, kondisi, daftar URL gambar.
- Harga awal, kelipatan minimum, harga saat ini, nama bidder tertinggi.
- Countdown yang mengacu pada waktu server.
- Jumlah penonton online.
- Riwayat 50 bid terbaru; bisa memuat lebih lama dengan cursor.
- Form bid dengan tombol nominal cepat (harga saat ini + 1×, 2×, dan 5× increment).
- Status `draft`, `live`, `ended`; halaman hasil menampilkan pemenang atau "tidak terjual".

### 5.3 Bid real-time

- Bid hanya untuk user terautentikasi.
- Server memvalidasi status, nominal, increment, identitas, dan rate limit.
- Bid disimpan di PostgreSQL sebelum disiarkan.
- Semua client di room menerima event `bid` dengan urutan yang konsisten.
- Bidder sebelumnya menerima event `outbid` (hanya koneksinya).
- Client menampilkan state pending sampai ack; event WS menjadi konfirmasi akhir.

### 5.4 Aturan bid

- Bid pertama: `amount >= starting_price`.
- Bid berikutnya: `amount >= current_price + min_increment`.
- Bidder yang sedang memegang harga tertinggi tidak boleh bid lagi (`ALREADY_HIGHEST`).
- Admin/pembuat lelang tidak boleh bid (`CREATOR_CANNOT_BID`).
- Nominal berupa integer rupiah, minimum 1, maksimum 1.000.000.000.000.
- Satu user maksimum 3 bid/detik (burst 5).
- Idempotency key UUID per percobaan bid; key yang sama mengembalikan hasil sebelumnya.

### 5.5 Anti-sniping

- Jika bid diterima dalam 60 detik terakhir, `ends_at` diperpanjang 60 detik.
- Maksimum 5 perpanjangan per lelang. Setelah cap tercapai, tidak ada perpanjangan lagi.
- Setiap perpanjangan disiarkan sebagai event `extended` dengan `ends_at` baru.
- Perpanjangan dan bid berada dalam transaksi yang sama.

### 5.6 Presence

- Jumlah koneksi WebSocket aktif di room, termasuk guest.
- Event `presence` dikirim maksimal sekali per detik (debounce) untuk menghindari lonjakan event.
- Presence bersifat sementara dan tidak disimpan di database.

### 5.7 Admin

- Membuat lelang: judul, deskripsi, kondisi, gambar, harga awal, increment, durasi.
- Memulai lelang: status `draft` → `live`, `starts_at = now`, `ends_at = now + duration_seconds` yang sudah ditentukan saat pembuatan.
- Mengakhiri lebih awal: status `live` → `ended`, pemenang dihitung dari bid tertinggi.
- Hanya satu lelang `live` pada satu waktu, dijaga partial unique index.
- UI admin minimal berupa satu halaman form + daftar lelang.

## 6. Arsitektur

```
Browser
  ├── HTTP same-origin ──► Caddy / Vite proxy ──► React Router SSR (Node)
  │                                                    └── loader ──► Go API (cookie diteruskan)
  ├── REST /api/* ──────────────────────────────► Go API
  └── WebSocket /ws/auctions/{id} ──────────────► Go API
                                                       │
                                                       └── PostgreSQL
```

- `web/`: React Router v7 framework mode, SSR, TypeScript, Tailwind.
- `server/`: satu binary Go. REST, WebSocket, auth, metrics.
- PostgreSQL: satu-satunya sumber kebenaran.
- SSR loader mengambil data awal langsung dari Go melalui URL internal dan meneruskan cookie. Client memakai path same-origin `/api` dan `/ws` yang diproksikan ke Go.
- Tidak ada CORS di production karena satu origin. Dev memakai proxy Vite untuk `/api` dan `/ws`.

### 6.1 Actor per room

Setiap lelang live memiliki satu goroutine actor yang memiliki:

- Status lelang dan timer.
- Harga saat ini dan bidder tertinggi.
- Daftar koneksi client.
- Sequence event.
- Buffer perpanjangan anti-sniping.

Semua perintah (`join`, `leave`, `bid`, `tick`, `end`) masuk melalui channel dan diproses berurutan. Tidak ada lock global. Ini menjamin dua bid tidak dapat diproses bersamaan dan urutan broadcast konsisten.

Actor adalah cache/read model. Sebelum mengubah state, bid dipersist ke PostgreSQL. Jika commit gagal, state actor tidak berubah dan tidak ada event yang dikirim.

### 6.2 Pemisahan command dan event

- Command (bid, admin action) melalui REST. Mudah diuji, punya status code dan idempotency.
- Event (perubahan state) melalui WebSocket. Client tidak pernah mengirim harga sebagai fakta.

### 6.3 SSR dan state client

- Loader menyediakan snapshot awal: detail lelang, bid terbaru, `server_time`, `user`.
- Setelah hydrate, store WebSocket (context + reducer) menjadi sumber state live.
- Tidak memakai TanStack Query agar tidak ada dua cache yang bisa berbeda.
- Snapshot dari WebSocket menimpa state loader. Hidrasi tidak boleh menyebabkan countdown berbeda antara server dan client; countdown dihitung dari `server_time` + `performance.now()`.
- Guest tetap mendapat SSR dan WebSocket; form bid menampilkan ajakan login.

## 7. Protokol WebSocket

Endpoint: `GET /ws/auctions/{id}`

### 7.1 Snapshot

Saat koneksi dibuka, server mengirim satu pesan:

```json
{
  "type": "snapshot",
  "epoch": "uuid-per-proses-room",
  "seq": 42,
  "server_time": "2026-10-20T10:00:00.000Z",
  "auction": { "...": "..." },
  "bids": [{ "id": 1, "display_name": "Raihan", "amount": 1500000, "created_at": "..." }],
  "viewer_count": 12
}
```

`bids` berisi maksimum 50 bid terbaru, urut dari terbaru.

### 7.2 Event

Semua event memakai envelope:

```json
{
  "type": "bid",
  "auction_id": 1,
  "epoch": "uuid",
  "seq": 43,
  "server_time": "2026-10-20T10:00:05.000Z",
  "payload": { "...": "..." }
}
```

| Type | Penerima | Payload |
|---|---|---|
| `bid` | semua di room | bid id, amount, display_name, current_price, highest_bidder |
| `outbid` | koneksi bidder sebelumnya | new_price, new_bidder_display_name |
| `presence` | semua di room | viewer_count |
| `extended` | semua di room | ends_at, extension_count |
| `ended` | semua di room | winner, final_price, reason (`timeout` / `admin`) |

### 7.3 Urutan, reconnect, dan heartbeat

- `seq` naik satu untuk setiap event yang disiarkan.
- `epoch` dibuat ulang setiap proses/room dimulai. Jika `epoch` berubah, client wajib memakai snapshot baru.
- Jika client mendeteksi `seq` meloncat saat koneksi masih terbuka, client meminta snapshot ulang (reconnect).
- Reconnect selalu menghasilkan snapshot baru; event lama tidak diputar ulang.
- Exponential backoff 0.5s → 8s dengan jitter.
- Server mengirim ping setiap 20 detik; client harus membalas pong dalam 10 detik. Jika tidak, koneksi ditutup dan client reconnect.
- WebSocket bersifat read-only untuk client. Tidak ada command bid melalui socket.

### 7.4 Close code

| Code | Arti |
|---|---|
| 4004 | Lelang tidak ditemukan |
| 1013 | Server sedang sibuk / mencoba lagi |
| 1001 | Server shutdown; client reconnect |

## 8. Alur bid & lifecycle

### 8.1 Bid

1. Client `POST /api/auctions/{id}/bids` dengan `{amount, idempotency_key}`.
2. Auth diverifikasi dari cookie. Rate limit diperiksa.
3. Actor room memvalidasi status, nominal, increment, dan aturan bidder.
4. Cek idempotency: jika key sudah ada untuk user ini, kembalikan bid lama.
5. Transaksi PostgreSQL:
   - `SELECT ... FOR UPDATE` baris lelang.
   - Insert bid.
   - Update `current_price`, `highest_bidder_id`.
   - Jika dalam 60 detik terakhir dan `extension_count < 5`, perpanjang `ends_at`, increment `extension_count`.
   - Commit.
6. Setelah commit, actor memperbarui state dan menyiarkan `bid`, lalu `extended` bila ada perpanjangan.
7. Kirim `outbid` ke koneksi bidder sebelumnya.
8. Respons `201` berisi bid yang diterima dan state terbaru. Replay idempotent mengembalikan `200`.

Jika dua bid bersamaan masuk, actor memprosesnya berurutan. Hanya bid pertama yang valid pada harga tersebut yang diterima; bid kedua mendapat `BID_TOO_LOW` kecuali nominalnya memenuhi increment baru.

### 8.2 Mulai lelang

1. Admin `POST /api/admin/auctions/{id}/start`.
2. Transaksi mengunci baris, memastikan status `draft`, dan tidak ada lelang `live`.
3. Set `status = live`, `starts_at = now`, `ends_at = now + duration_seconds`.
4. Actor room dibuat, snapshot tersedia, timer dijadwalkan.
5. Broadcast tidak diperlukan karena belum ada penonton; client yang membuka room mendapat snapshot.

### 8.3 Selesai

- Timer actor memanggil `end` pada `ends_at`.
- Status `live` → `ended`; pemenang adalah `highest_bidder_id`; `final_price = current_price`.
- Jika tidak ada bid, `winner_id` dan `final_price` bernilai null (tidak terjual).
- Transisi idempoten: `end` kedua tidak mengubah apa pun.
- Bid yang datang setelah `ends_at` ditolak dengan `AUCTION_NOT_LIVE`.
- Admin dapat mengakhiri lebih awal dengan alur yang sama, `reason = admin`.

### 8.4 Restart server

- Saat boot, server memuat semua lelang `status = live`.
- Jika `ends_at` sudah lewat, lelang langsung diakhiri.
- Jika belum, timer dijadwalkan ulang dan actor dibuat dari data PostgreSQL.
- Presence dan `seq` lama hilang; client reconnect dan menerima snapshot dengan `epoch` baru.

## 9. Data model

Harga disimpan sebagai integer rupiah (`bigint`), bukan float.

```sql
users (
  id            bigint generated always as identity primary key,
  email         text not null unique,
  password_hash text not null,
  display_name  text not null,
  role          text not null default 'user' check (role in ('user', 'admin')),
  created_at    timestamptz not null default now()
)

auctions (
  id               bigint generated always as identity primary key,
  created_by       bigint not null references users(id),
  title            text not null,
  description      text not null,
  condition        text not null,
  image_urls       text[] not null default '{}',
  starting_price   bigint not null check (starting_price > 0),
  min_increment    bigint not null check (min_increment > 0),
  duration_seconds int not null check (duration_seconds between 60 and 86400),
  current_price    bigint not null,
  highest_bidder_id bigint references users(id),
  status           text not null default 'draft' check (status in ('draft', 'live', 'ended')),
  starts_at        timestamptz,
  ends_at          timestamptz,
  extension_count  int not null default 0,
  winner_id        bigint references users(id),
  final_price      bigint,
  created_at       timestamptz not null default now()
)

-- Maksimal satu lelang live.
create unique index one_live_auction on auctions ((true)) where status = 'live';

bids (
  id              bigint generated always as identity primary key,
  auction_id      bigint not null references auctions(id),
  user_id         bigint not null references users(id),
  amount          bigint not null check (amount > 0),
  idempotency_key uuid not null,
  created_at      timestamptz not null default now(),
  unique (user_id, idempotency_key)
)

create index bids_auction_id_id on bids (auction_id, id desc);
```

Catatan:

- `current_price` diinisialisasi sama dengan `starting_price` saat lelang dibuat.
- `condition` berupa teks bebas terkontrol (mis. "Like new", "Good", "Fair").
- Gambar hanya URL `http`/`https`; tidak ada upload.
- Presence, `seq`, dan timer tidak disimpan di database.

## 10. API

Semua respons error memakai bentuk:

```json
{ "error": { "code": "BID_TOO_LOW", "message": "Bid minimal Rp1.550.000", "details": { "minimum": 1550000 } } }
```

### Auth

| Method | Path | Keterangan |
|---|---|---|
| POST | `/api/auth/register` | Body: email, password, display_name. Set cookie. |
| POST | `/api/auth/login` | Body: email, password. Set cookie. |
| POST | `/api/auth/logout` | Hapus cookie. |
| GET | `/api/auth/me` | User saat ini atau 401. |

### Lelang

| Method | Path | Keterangan |
|---|---|---|
| GET | `/api/auctions/current` | Lelang live + bid terbaru + server_time; 204 jika tidak ada. |
| GET | `/api/auctions/{id}` | Detail + 50 bid terbaru + server_time. |
| GET | `/api/auctions/{id}/bids?before_id=&limit=50` | Riwayat lebih lama. |
| POST | `/api/auctions/{id}/bids` | Body: amount, idempotency_key. 201/200. |

### Admin

| Method | Path | Keterangan |
|---|---|---|
| POST | `/api/admin/auctions` | Buat lelang draft. |
| POST | `/api/admin/auctions/{id}/start` | Mulai lelang. |
| POST | `/api/admin/auctions/{id}/end` | Akhiri lebih awal. |

### Operasional

| Method | Path | Keterangan |
|---|---|---|
| GET | `/healthz` | Cek proses + ping database. |
| GET | `/metrics` | Metrik Prometheus. |
| GET | `/ws/auctions/{id}` | WebSocket. |

## 11. Error handling

| Code | HTTP | Kapan |
|---|---|---|
| `UNAUTHENTICATED` | 401 | Tidak ada/invalid session. |
| `FORBIDDEN` | 403 | Bukan admin. |
| `CREATOR_CANNOT_BID` | 403 | Pembuat lelang mencoba bid. |
| `AUCTION_NOT_FOUND` | 404 | ID tidak ada. |
| `AUCTION_NOT_LIVE` | 409 | Bid/aksi pada lelang yang belum live atau sudah ended. |
| `AUCTION_ALREADY_LIVE` | 409 | Mulai lelang saat ada lelang live lain. |
| `BID_TOO_LOW` | 422 | Nominal di bawah minimum; `details.minimum` diisi. |
| `ALREADY_HIGHEST` | 409 | Bidder sudah memegang harga tertinggi. |
| `INVALID_AMOUNT` | 422 | Bukan integer, ≤ 0, atau melebihi batas. |
| `IDEMPOTENCY_CONFLICT` | 409 | Key sama dipakai dengan payload berbeda. |
| `RATE_LIMITED` | 429 | Melebihi 3 bid/detik; sertakan `Retry-After`. |
| `VALIDATION_ERROR` | 422 | Field wajib/format salah. |
| `INTERNAL` | 500 | Kesalahan tak terduga; detail tidak dibocorkan ke client. |

Aturan:

- Bid tidak pernah disiarkan sebelum commit DB berhasil.
- Error domain selalu punya kode stabil; client memetakan kode ke pesan Indonesia.
- Idempotent replay mengembalikan bid yang sudah ada, bukan error.
- Jika unique constraint `(user_id, idempotency_key)` terlanggar karena race, server membaca bid yang ada dan mengembalikannya.
- WS hanya menerima snapshot/event; tidak ada error domain melalui socket selain close code.

## 12. Keamanan

- bcrypt cost 12; password minimal 8 karakter.
- JWT HS256, umur 24 jam, secret dari environment.
- Cookie `httpOnly`, `SameSite=Lax`, `Secure` di production, `Path=/`.
- Endpoint yang mengubah state memeriksa header `Origin`; WebSocket memeriksa `Origin` saat handshake.
- Rate limit login per IP: 10 percobaan/menit.
- Semua query memakai parameter pgx; tidak ada string SQL yang dirakit dari input.
- Gambar divalidasi hanya `http`/`https`.
- Admin hanya dari seed/CLI; registrasi publik tidak bisa memilih role.
- Log tidak memuat password, token, atau email mentah.
- Pesan error 500 tidak memuat detail internal.

## 13. Observability

Metrik Prometheus:

- `auction_connected_clients{auction_id}`
- `auction_bids_total{result="accepted|rejected"}`
- `auction_bid_duration_seconds` (histogram validasi + commit DB)
- `auction_broadcast_duration_seconds` (histogram)
- `auction_live_auctions`
- `auction_ws_reconnects_total`

Log structured (`slog`, JSON) memuat `request_id`, `user_id`, `auction_id`, `bid_id`, `duration_ms`, `error_code`. Tidak memuat credential.

`/healthz` mengembalikan 200 hanya jika proses hidup dan `SELECT 1` berhasil.

## 14. Pengujian

### Unit (Go)

- Tabel aturan bid: bid pertama, increment, bid di bawah minimum, bidder tertinggi, pembuat lelang.
- Anti-sniping: bid di luar/dalam 60 detik, cap 5 perpanjangan.
- Perhitungan pemenang, termasuk lelang tanpa bid.
- Idempotency: replay mengembalikan bid yang sama; payload berbeda → konflik.
- Format uang integer dan batas nominal.

### Integration (Go + PostgreSQL)

- Dua bid bersamaan pada nominal yang sama: tepat satu diterima, `seq` konsisten.
- Dua client WS menerima event `bid` dengan harga dan urutan yang sama.
- `outbid` hanya diterima koneksi bidder sebelumnya.
- Lelang berakhir otomatis; bid setelah `ends_at` ditolak.
- Restart: lelang live dimuat ulang dan timer tetap berjalan.
- Jalankan `go test -race ./...`.

### End-to-end (Playwright)

- Dua browser context: user A bid, user B melihat harga baru dalam batas waktu.
- Outbid toast muncul pada bidder sebelumnya.
- Reconnect setelah koneksi diputus menampilkan bid yang terlewat.
- Guest tidak bisa bid dan diarahkan ke login.
- Admin membuat dan memulai lelang; room menampilkan status live.

### Load test

- Program Go di `loadtest/` membuat N koneksi WebSocket dan M bidder.
- Parameter: jumlah viewer, jumlah bidder, bid/detik, durasi.
- Output: koneksi berhasil, bid diterima/ditolak, p50/p95/p99 latency broadcast.
- Hasil dicatat di README.

Tidak ada suite test per fungsi. Fokus pada jalur bid, waktu, dan realtime.

## 15. Struktur repo & setup

```
live-auction/
  server/
    cmd/api/
    cmd/seed/
    internal/auction/     # domain + actor room
    internal/httpapi/     # REST, auth, admin
    internal/ws/          # hub, snapshot, broadcast
    internal/store/       # PostgreSQL, migrasi
    migrations/
    go.mod
  web/
    app/                  # routes, loaders, components
    app/stores/           # WebSocket store + reducer
    package.json
  loadtest/
  docker-compose.yml
  .env.example
  README.md
```

Setup lokal:

1. `docker compose up -d postgres`
2. Salin `.env.example` ke `.env`.
3. `cd server && go run ./cmd/api`. Migrasi dijalankan saat startup.
4. `cd web && npm install && npm run dev`.
5. `go run ./cmd/seed` untuk admin, user demo, dan satu lelang draft.

Production: satu domain di belakang Caddy. `/api/*` dan `/ws/*` ke Go, sisanya ke React Router SSR. Semua container dijalankan dengan `docker compose`.

## 16. Target performa

Target yang diverifikasi load test (bukan jaminan produksi):

- 1.000 koneksi WebSocket bersamaan.
- 100 bid/detik selama 60 detik.
- p95 latency broadcast ≤ 250 ms pada mesin lokal 2 vCPU / 4 GB.
- Error rate < 1%.

Jika target tidak tercapai, README mencantumkan angka aktual dan bottleneck yang ditemukan.

## 17. Acceptance criteria

1. User dapat register, login, logout, dan sesi bertahan setelah refresh.
2. Guest dapat menonton; form bid mengarahkan ke login.
3. Dua browser melihat harga, bidder tertinggi, dan riwayat bid yang sama secara real-time.
4. Bid di bawah minimum, bid setelah ended, bid oleh bidder tertinggi, dan bid oleh pembuat lelang ditolak dengan kode yang benar.
5. Dua bid bersamaan pada nominal sama menghasilkan tepat satu bid tersimpan.
6. Bid dalam 60 detik terakhir memperpanjang lelang 60 detik, maksimal 5 kali.
7. Lelang berakhir otomatis; pemenang dan harga final benar; lelang tanpa bid berstatus tidak terjual.
8. Client yang reconnect menerima snapshot dan tidak kehilangan bid.
9. Admin dapat membuat, memulai, dan mengakhiri lelang; hanya satu lelang live pada satu waktu.
10. Server restart saat lelang live memulihkan timer dan state dari database.
11. `go test -race ./...` dan Playwright lulus.
12. `/metrics` menampilkan metrik yang terdaftar dan log berformat JSON.
13. Laporan load test tersedia di README dengan angka aktual.
