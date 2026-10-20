# Lelang

Prototipe UI ruang lelang real-time untuk elektronik bekas. Fokus tahap ini: alur bid, harga berjalan, ledger, presence, dan state lelang. Backend Go dan PostgreSQL menyusul sesuai spec.

## Isi repo

- `prototype/`: prototipe statis HTML, CSS, JavaScript. Tanpa build step.
- `DESIGN.md`: arah visual, palet, tipografi, dan aturan desain.
- `docs/superpowers/specs/2026-10-20-live-auction-design.md`: desain produk dan teknis untuk implementasi berikutnya.

## Jalankan

Dari root repo:

```powershell
python -m http.server 4173 --directory prototype
```

Buka `http://localhost:4173`.

Alternatif tanpa Python:

```powershell
npx serve prototype -l 4173
```

## Coba alur real-time

1. Buka halaman di dua tab.
2. Isi nama berbeda di tiap tab. Nama disimpan di `sessionStorage`, jadi setiap tab bisa menjadi penawar berbeda.
3. Pasang bid di tab pertama, lalu bid lebih tinggi di tab kedua.
4. Tab pertama harus menerima harga baru dan toast "Kamu di-outbid".

Data disinkronkan lewat `BroadcastChannel` dan `localStorage`. Sinkronisasi hanya berlaku antar tab di browser dan origin yang sama. Tidak ada server.

## Yang berfungsi

- Bid dengan validasi minimum dan kelipatan.
- Harga, bidder tertinggi, countdown, dan ledger yang sinkron antar tab.
- Presence jumlah tab yang aktif.
- Notifikasi outbid.
- Anti-sniping: bid di 60 detik terakhir memperpanjang sesi 60 detik, maksimal 5 kali.
- Lelang berakhir otomatis setelah 10 menit.
- Hasil pemenang, harga akhir, dan state tidak terjual.
- Reset demo dan simulasi penantang.
- State kosong, loading, error, dan pemulihan state rusak.
- Fokus keyboard terlihat, target sentuh minimal 44px, dan layout 320px sampai desktop.

## Yang belum ada

- Backend, database, autentikasi, dan otorisasi.
- Pembayaran, ongkir, dan pembuatan lelang oleh admin.
- Upload foto. Halaman memakai ilustrasi dan label "Foto unit belum tersedia".
- Rate limit dan idempotency di server.

Semua data di halaman adalah data demo dan diberi label "Data demo".

## Verifikasi

Skrip cek browser memakai Playwright dan Chrome yang sudah terpasang. Jalankan server statis lebih dulu, lalu:

```powershell
cd prototype/tests
npm install
npm test
```

Jika memakai Edge:

```powershell
$env:PW_CHANNEL="msedge"; npm test
```

Cek mencakup dua tab, bid valid dan tidak valid, outbid, presence, reset, simulasi penantang, state ended dan unsold, pemulihan state rusak, fokus, ukuran target, label input, dan overflow di 320px, 390px, serta 768px. Screenshot tersimpan di `prototype/tests/shots/` dan tidak di-commit.

Hasil terakhir: PASS, tanpa error console atau page error.

## Berikutnya

Implementasi backend mengikuti spec: Go, PostgreSQL, REST untuk command bid, WebSocket untuk event, actor per room, auth cookie, dan admin lifecycle. Prototipe ini menjadi acuan UI dan perilaku sebelum backend ditambahkan.
