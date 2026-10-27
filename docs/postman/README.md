# Postman Collection: LELANG API

File:

- `live-auction.postman_collection.json`
- `live-auction.local.postman_environment.json`

## Import

1. Buka Postman.
2. Klik **Import**.
3. Masukkan kedua file JSON.
4. Pilih environment **LELANG Local**.
5. Pastikan backend berjalan di `http://localhost:8080`.
6. Jika frontend memakai origin lain, ubah `appOrigin` di environment.

## Prasyarat Backend

Jalankan dari folder `server`:

```powershell
cd D:\live-auction\server
go run ./cmd/api
```

Siapkan data demo:

```powershell
cd D:\live-auction\server
go run ./cmd/seed
```

Seed membuat admin dan bidder dengan password dari `.env`:

```text
SEED_ADMIN_EMAIL
SEED_ADMIN_PASSWORD
SEED_BIDDER_PASSWORD
```

Isi `adminEmail`, `adminPassword`, `bidderEmail`, dan `bidderPassword` di environment Postman sesuai `.env`.

## Cookie Auth

Access token dan refresh token disimpan di cookie httpOnly.

Postman menyimpan cookie secara otomatis jika cookie jar aktif. Jangan menambahkan header:

```text
Authorization: Bearer ...
```

Server membaca token dari cookie `access_token`, bukan dari header Authorization.

Karena cookie disimpan per domain, login admin dan login bidder akan saling menimpa pada host yang sama. Urutan pengujian yang disarankan:

```text
1. Login Admin
2. Create Draft
3. Start Draft
4. Login Bidder
5. Place Bid
6. Login Admin lagi
7. End Auction
```

## Header Origin

Endpoint yang mengubah state membutuhkan header:

```text
Origin: {{appOrigin}}
```

Endpoint tersebut:

- register
- login
- refresh
- logout
- create auction
- start auction
- end auction
- delete draft
- place bid

Collection sudah menambahkan header tersebut pada request yang membutuhkannya.

## Variabel Penting

| Variable | Arti |
|---|---|
| `baseUrl` | Alamat API, default `http://localhost:8080` |
| `appOrigin` | Origin frontend, default `http://localhost:5173` |
| `adminEmail` | Email admin seed |
| `adminPassword` | Password admin seed |
| `bidderEmail` | Email bidder |
| `bidderPassword` | Password bidder |
| `auctionId` | Auction live yang sedang dipakai |
| `draftAuctionId` | Draft yang baru dibuat |
| `bidId` | Bid terakhir yang berhasil |
| `startingPrice` | Harga awal auction baru |
| `minIncrement` | Kelipatan minimum bid |
| `bidAmount` | Nominal bid berikutnya |

`bidAmount` diperbarui otomatis oleh test script setelah bid berhasil:

```text
bidAmount = current_price + min_increment
```

## Alur Utama

### 1. Health dan metrics

Jalankan:

```text
Health
Metrics
```

Harapan:

```text
GET /healthz  -> 200 {"status":"ok"}
GET /metrics  -> 200 teks Prometheus
```

Jika `/healthz` mengembalikan 503, cek koneksi PostgreSQL.

### 2. Login admin

Jalankan:

```text
Auth -> Login Admin
```

Cookie admin akan disimpan Postman.

### 3. Buat dan mulai lelang

Jalankan:

```text
Admin -> Create Draft
Admin -> Start Draft
```

`Create Draft` menyimpan `id` ke `draftAuctionId`.

`Start Draft` menyimpan `id` ke `auctionId` dan mengatur `bidAmount` ke `current_price`.

Hanya satu auction yang boleh live pada satu waktu. Jika sudah ada live auction, start akan mengembalikan:

```text
409 AUCTION_ALREADY_LIVE
```

### 4. Login bidder

Jalankan:

```text
Auth -> Login Bidder
```

Cookie admin akan tertimpa cookie bidder.

### 5. Place bid

Jalankan:

```text
Bid -> Place Bid
```

Body:

```json
{
  "amount": 1000000,
  "idempotency_key": "uuid"
}
```

`idempotency_key` di collection memakai `{{$guid}}`, jadi setiap request mendapat UUID baru.

Response sukses:

```json
{
  "auction": {
    "id": "uuid",
    "current_price": 1000000,
    "highest_bidder_id": "uuid",
    "version": 1
  },
  "bid": {
    "id": "uuid",
    "auction_id": "uuid",
    "user_id": "uuid",
    "amount": 1000000
  },
  "replay": false,
  "extended": false
}
```

Status:

```text
201 bid baru
200 replay idempotency
```

Jika amount terlalu rendah:

```json
{
  "error": {
    "code": "BID_TOO_LOW",
    "message": "Bid di bawah harga minimum"
  }
}
```

### 6. Lihat riwayat bid

Jalankan:

```text
Public Auctions -> Bid History
```

Query yang didukung:

```text
GET /api/auctions/{id}/bids?limit=50
GET /api/auctions/{id}/bids?before_id=<uuid>&limit=50
```

### 7. End auction

Login admin lagi, lalu jalankan:

```text
Admin -> End Auction
```

Response berisi status `ended`, pemenang, dan harga akhir.

### 8. Results

Jalankan:

```text
Public Auctions -> Results
```

Untuk melihat arsip lelang yang sudah selesai.

## WebSocket

Postman versi baru bisa membuka WebSocket request, tetapi WebSocket tidak dimasukkan sebagai HTTP request di collection ini.

Gunakan URL:

```text
ws://localhost:8080/ws/auctions/{{auctionId}}
```

Langkah manual:

1. Pastikan auction live dan `auctionId` sudah terisi.
2. Buka WebSocket request di Postman.
3. Connect ke URL di atas.
4. Pesan pertama adalah snapshot:

```json
{
  "type": "snapshot",
  "epoch": "uuid-room",
  "seq": 0,
  "version": 1,
  "auction": {},
  "bids": [],
  "viewer_count": 1
}
```

5. Dari tab atau request lain, lakukan `Place Bid`.
6. WebSocket akan menerima event:

```text
bid
outbid
extended
presence
ended
```

Bid tetap dikirim lewat REST, bukan lewat WebSocket.

## Endpoint dan Status

| Method | Path | Auth | Status sukses |
|---|---|---|---|
| GET | `/healthz` | tidak | 200 |
| GET | `/metrics` | tidak | 200 |
| POST | `/api/auth/register` | tidak | 201 |
| POST | `/api/auth/login` | tidak | 200 |
| POST | `/api/auth/refresh` | cookie refresh | 200 |
| POST | `/api/auth/logout` | cookie refresh | 204 |
| GET | `/api/auth/me` | cookie access | 200 |
| GET | `/api/auctions/current` | tidak | 200 atau 204 |
| GET | `/api/auctions/{id}` | tidak | 200 |
| GET | `/api/auctions/{id}/bids` | tidak | 200 |
| GET | `/api/results` | tidak | 200 |
| POST | `/api/auctions/{id}/bids` | user | 201 atau 200 |
| GET | `/api/admin/auctions` | admin | 200 |
| POST | `/api/admin/auctions` | admin | 201 |
| POST | `/api/admin/auctions/{id}/start` | admin | 200 |
| POST | `/api/admin/auctions/{id}/end` | admin | 200 |
| DELETE | `/api/admin/auctions/{id}` | admin | 204 |

## Error Umum

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Data tidak valid"
  }
}
```

Kode yang sering muncul:

| Code | Arti |
|---|---|
| `UNAUTHENTICATED` | Cookie access token tidak ada |
| `TOKEN_EXPIRED` | Access token expired, lakukan refresh |
| `TOKEN_INVALID` | Token tidak valid |
| `FORBIDDEN` | Role bukan admin |
| `AUCTION_NOT_FOUND` | Auction tidak ditemukan |
| `AUCTION_NOT_LIVE` | Auction belum atau sudah tidak live |
| `AUCTION_NOT_DRAFT` | Auction bukan draft |
| `AUCTION_ALREADY_LIVE` | Sudah ada auction live lain |
| `BID_TOO_LOW` | Nominal bid di bawah minimum |
| `ALREADY_HIGHEST` | User sudah menjadi penawar tertinggi |
| `CREATOR_CANNOT_BID` | Pembuat auction mencoba bid |
| `IDEMPOTENCY_CONFLICT` | Key idempotency dipakai untuk nominal berbeda |
| `RATE_LIMITED` | Terlalu banyak request |

## Rate Limit

- Login/register: 10 request per menit dengan burst 10.
- Refresh: 30 request per menit.
- Bid: 3 request per detik dengan burst 5 per user.

Jika kena rate limit, response:

```text
429 RATE_LIMITED
```

## Catatan

- Jangan menguji dengan password atau token asli di environment yang dibagikan.
- Jangan menjalankan load test ke production.
- Jika origin frontend berubah, ubah `APP_ORIGIN` di `.env` dan `appOrigin` di Postman.
- Jika cookie tidak terkirim, pastikan request memakai host yang sama dengan saat login dan cookie jar Postman aktif.
