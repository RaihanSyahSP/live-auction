# PRD: LELANG

Nama kerja: LELANG
Versi: 0.1
Tanggal: 2026-10-20
Status: Draft untuk review
Pemilik produk: belum ditetapkan
Konteks: proyek portofolio dengan scope produk yang dijaga kecil

Dokumen terkait:

- `DESIGN.md`: arah visual dan aturan desain.
- `docs/superpowers/specs/2026-10-20-live-auction-design.md`: desain teknis dan arsitektur target.
- `prototype/`: prototipe UI dan admin yang sudah berjalan secara lokal.

## 1. Ringkasan

LELANG adalah ruang lelang langsung untuk barang elektronik bekas. Satu lelang berlangsung pada satu waktu. Penonton melihat harga, riwayat bid, dan sisa waktu yang sama tanpa perlu refresh. Peserta yang login dapat memasang bid, dan admin dapat membuat draft, memulai, serta mengakhiri lelang.

Produk ini tidak berhenti di halaman katalog. Fokusnya adalah momen lelang: harga bergerak, penantang saling menaikkan bid, waktu hampir habis, dan pemenang ditentukan oleh aturan yang jelas.

Tahap saat ini adalah prototipe UI dengan state lokal di browser. Backend Go, PostgreSQL, autentikasi nyata, dan WebSocket belum dibuat.

## 2. Latar belakang dan masalah

Pembeli elektronik bekas biasanya bekerja dengan daftar barang, harga yang tidak selalu diperbarui, dan riwayat penawaran yang tersebar. Pada lelang yang berjalan singkat, keterlambatan informasi membuat pengalaman terasa tidak adil: satu orang melihat harga lama, orang lain sudah menaikkan bid.

Admin atau penjual juga membutuhkan alur yang sederhana untuk menyiapkan satu barang, menentukan harga awal dan kelipatan, menjalankan timer, lalu menutup lelang dengan pemenang yang jelas. Mereka tidak membutuhkan marketplace penuh untuk melakukan itu.

Masalah yang ingin diselesaikan:

1. Semua penonton harus melihat harga dan status lelang yang sama pada waktu yang hampir bersamaan.
2. Aturan bid harus ditegakkan di server, bukan di browser.
3. Admin harus bisa menjalankan satu sesi lelang tanpa proses manual yang berantakan.
4. Peserta harus mendapat umpan balik yang jelas saat bid diterima, ditolak, atau direbut penantang.

## 3. Tujuan produk

1. Menyajikan harga terbaru ke semua penonton secara real-time.
2. Membuat aturan bid yang dapat diprediksi: minimum, kelipatan, batas waktu, dan anti-sniping.
3. Memungkinkan admin membuat, memulai, dan mengakhiri satu lelang dengan alur yang singkat.
4. Menjaga hasil akhir tetap jelas: pemenang, harga akhir, atau status tidak terjual.
5. Menjadi bukti kemampuan teknis dalam menangani state, konkurensi, dan komunikasi real-time.

## 4. Non-tujuan

Fitur berikut tidak masuk versi pertama:

- Pembayaran, deposit, ongkir, dan penyelesaian transaksi di dalam aplikasi.
- KYC, verifikasi identitas, dan moderasi penjual.
- Chat, komentar, dan email notifikasi.
- Katalog multi-kategori dan banyak lelang live sekaligus.
- Auto-bid, reserve price, dan buy-now.
- Upload foto. Versi awal memakai URL gambar atau placeholder.
- Aplikasi mobile native.
- Multi-currency dan multi-bahasa.
- Rekomendasi barang dan personalisasi.
- Monetisasi, langganan, iklan, dan model bisnis berbayar.

## 5. Pengguna dan peran

| Peran | Kebutuhan utama | Keberhasilan |
|---|---|---|
| Guest | Melihat barang, harga, dan jalannya lelang tanpa akun | Bisa memahami kondisi lelang dan diarahkan login saat ingin bid |
| Bidder | Memasang bid, melihat posisi, dan tahu saat direbut | Bid diterima atau ditolak dengan alasan yang jelas, tanpa refresh |
| Admin | Menyiapkan dan menjalankan lelang | Bisa membuat draft, preview, start, memantau, dan end tanpa langkah membingungkan |

Untuk versi pertama, admin juga berperan sebagai penjual atau operator barang. Pemisahan peran penjual dan admin dapat ditambahkan nanti jika produk berkembang.

Matriks izin:

| Aksi | Guest | Bidder | Admin |
|---|---|---|---|
| Melihat ruang dan hasil | Ya | Ya | Ya |
| Memasang bid | Tidak | Ya | Tidak pada lelang sendiri |
| Membuat dan memulai lelang | Tidak | Tidak | Ya |
| Mengakhiri lelang | Tidak | Tidak | Ya |
| Menghapus draft | Tidak | Tidak | Ya |

## 6. Jobs to be done

- Saat melihat barang yang saya minati, saya ingin tahu harga berjalan dan siapa yang tertinggi, supaya saya bisa menentukan batas bid.
- Saat memasang bid, saya ingin langsung tahu apakah bid saya diterima, supaya saya tidak menebak posisi saya.
- Saat bid saya direbut, saya ingin diberi tahu, supaya saya bisa memutuskan apakah akan menaikkan lagi.
- Saat waktu hampir habis, saya ingin aturan perpanjangan yang jelas, supaya bid terakhir tidak ditentukan oleh refresh atau keterlambatan jaringan.
- Saat menjalankan lelang, saya ingin menyiapkan barang dan memulainya dengan cepat, supaya sesi tidak tertunda.

## 7. Prinsip produk

1. Server adalah sumber kebenaran untuk harga, waktu, dan pemenang.
2. Penonton tidak perlu akun. Bid memerlukan identitas.
3. Setiap penolakan bid menjelaskan alasannya dan angka minimum yang berlaku.
4. Satu lelang live pada satu waktu untuk versi pertama.
5. Tampilan harus tetap terbaca di ponsel dan dapat digunakan dengan keyboard.
6. Data demo tidak boleh disamarkan sebagai data nyata.
7. Lebih baik fitur sedikit tetapi utuh daripada banyak fitur dengan alur yang belum selesai.

## 8. Scope dan prioritas fitur

Prioritas memakai Must, Should, Could, dan Won't untuk versi pertama.

- Must: wajib ada agar MVP dapat dirilis.
- Should: penting, masuk MVP jika tidak mengorbankan Must.
- Could: dikerjakan setelah MVP.
- Won't: tidak direncanakan untuk versi ini.

| ID | Fitur | Prioritas | Prototype | Target rilis |
|---|---|---|---|---|
| F-01 | Register, login, logout, sesi | Must | Belum, hanya nama per tab | MVP |
| F-02 | Peran user dan admin | Must | Belum | MVP |
| F-03 | Ruang lelang publik | Must | UI ada | MVP |
| F-04 | Bid real-time dan validasi server | Must | Simulasi antar tab | MVP |
| F-05 | Aturan minimum dan kelipatan | Must | Ada di browser | MVP |
| F-06 | Anti-sniping 60 detik, maksimal 5 kali | Must | Simulasi | MVP |
| F-07 | Siklus draft, live, ended | Must | Ada | MVP |
| F-08 | Pemenang, harga akhir, tidak terjual | Must | Ada | MVP |
| F-09 | Admin create, preview, start, end, list | Must | Ada, lokal | MVP |
| F-10 | Satu lelang live pada satu waktu | Must | Ada | MVP |
| F-11 | Reconnect dan resync | Must | Belum, tanpa jaringan | MVP |
| F-12 | Loading, empty, error, ended, unsold | Must | Ada | MVP |
| F-13 | Responsive dan keyboard | Must | Ada | MVP |
| F-14 | Presence jumlah penonton | Should | Simulasi tab | MVP |
| F-15 | Notifikasi outbid | Should | Simulasi | MVP |
| F-16 | Metrics, structured log, health check | Should | Belum | Fase 2 |
| F-17 | Load test dan laporan latency | Should | Belum | Fase 2 |
| F-18 | Hapus draft | Should | Ada | MVP |
| F-19 | Audit log aksi admin | Could | Belum | Belum |
| F-20 | Edit draft sebelum live | Could | Belum | Belum |
| F-21 | Grafik harga dan statistik lelang | Could | Belum | Belum |
| F-22 | Share link dan OG image | Could | Belum | Belum |
| F-23 | Dark mode | Could | Belum | Belum |
| F-24 | Auto-bid atau proxy bidding | Could | Belum | Belum |
| F-25 | Reserve price | Could | Belum | Belum |
| F-26 | Buy-now | Could | Belum | Belum |
| F-27 | Payment, ongkir, KYC, chat | Won't | Belum | Tidak direncanakan |

## 9. Alur pengguna

### 9.1 Guest melihat lelang

1. Guest membuka halaman lelang.
2. Guest melihat barang, kondisi, harga awal atau harga saat ini, riwayat bid, jumlah penonton, dan sisa waktu.
3. Guest mencoba bid.
4. Sistem meminta login atau register.
5. Setelah login, guest kembali ke lelang yang sama.

### 9.2 Bidder memasang bid

1. Bidder membuka lelang yang sedang live.
2. Bidder mengisi nominal atau memilih nominal cepat.
3. Server memeriksa status lelang, identitas, nominal, kelipatan, dan batas waktu.
4. Jika valid, bid disimpan dan semua penonton menerima harga baru.
5. Jika bidder sebelumnya ada, bidder itu menerima notifikasi outbid.
6. Jika bid masuk di 60 detik terakhir, waktu diperpanjang.
7. Saat waktu habis, lelang ditutup dan pemenang ditentukan.

### 9.3 Admin menjalankan lelang

1. Admin login dengan akun admin.
2. Admin membuat draft: judul, subjudul, deskripsi, kondisi, kelengkapan, gambar, harga awal, kelipatan, dan durasi.
3. Admin membuka preview untuk memeriksa tampilan.
4. Admin memulai lelang. Draft menjadi live dan timer berjalan.
5. Admin memantau harga dan jumlah bid dari daftar lelang.
6. Admin dapat mengakhiri lebih awal atau menunggu timer habis.
7. Admin melihat pemenang dan harga akhir.

### 9.4 Skenario tepi

- Bid di bawah minimum ditolak dengan angka minimum.
- Bid dari bidder yang sedang tertinggi ditolak.
- Bid setelah lelang berakhir ditolak.
- Dua bid pada nominal yang sama diproses berurutan, hanya satu yang diterima.
- Admin mencoba memulai draft saat ada lelang live, ditolak.
- Server restart saat lelang live, timer dan state dipulihkan dari database.
- Client kehilangan koneksi, lalu reconnect dan menerima snapshot terbaru.
- Lelang berakhir tanpa bid, status menjadi tidak terjual.

## 10. Kebutuhan fungsional

### 10.1 Autentikasi dan peran

- Pengguna dapat register dengan email, password, dan display name.
- Pengguna dapat login dan logout.
- Sesi disimpan pada cookie httpOnly dengan masa berlaku 24 jam.
- Registrasi publik hanya menghasilkan role user.
- Admin dibuat melalui seed atau perintah operasional, bukan registrasi publik.
- Admin tidak boleh bid pada lelang yang dia buat.

Kriteria penerimaan:

- Sesi bertahan setelah refresh.
- Bid tanpa login mendapat 401.
- User biasa yang membuka endpoint admin mendapat 403.
- Admin yang mencoba bid pada lelangnya sendiri mendapat 403.

### 10.2 Ruang lelang

- Halaman lelang dapat dibuka tanpa login.
- Informasi yang ditampilkan: judul, subjudul, deskripsi, kondisi, kelengkapan, gambar, harga awal, kelipatan, harga saat ini, bidder tertinggi, riwayat bid, jumlah penonton, status, dan sisa waktu.
- Countdown mengacu pada waktu server, bukan jam perangkat.
- Riwayat bid menampilkan 50 bid terbaru, dengan opsi memuat lebih lama.
- Halaman memiliki state loading, empty, error, live, ended, dan unsold.
- Guest yang ingin bid diarahkan ke login tanpa kehilangan konteks lelang.

Kriteria penerimaan:

- Dua browser menampilkan harga dan bidder tertinggi yang sama.
- Countdown tidak berbeda antara dua perangkat yang jamnya berbeda.
- Guest dapat menonton tetapi tidak dapat mengirim bid.
- Setelah lelang berakhir, halaman menampilkan pemenang atau status tidak terjual.

### 10.3 Bid

- Bid hanya untuk pengguna terautentikasi.
- Nominal berupa integer rupiah.
- Bid pertama minimal sama dengan harga awal.
- Bid berikutnya minimal harga saat ini ditambah kelipatan.
- Bidder yang sedang memegang harga tertinggi tidak boleh bid lagi.
- Pembuat lelang tidak boleh bid.
- Server memvalidasi seluruh aturan sebelum menyimpan.
- Bid disimpan sebelum disiarkan. Jika commit gagal, tidak ada event harga baru.
- Setiap bid membawa idempotency key. Key yang sama mengembalikan hasil sebelumnya, bukan membuat bid baru.
- Rate limit 3 bid per detik dengan burst 5.
- Error menyebut kode dan angka minimum jika relevan.

Kriteria penerimaan:

- Bid di bawah minimum ditolak dan harga tidak berubah.
- Dua bid bersamaan pada nominal yang sama menghasilkan tepat satu bid tersimpan.
- Replay dengan idempotency key yang sama tidak menambah bid.
- Payload berbeda dengan key yang sama menghasilkan konflik.
- Tidak ada bid yang disiarkan sebelum transaksi database berhasil.

### 10.4 Anti-sniping

- Jika bid diterima dalam 60 detik terakhir, waktu berakhir diperpanjang 60 detik.
- Perpanjangan maksimal 5 kali per lelang.
- Setelah batas tercapai, bid tetap diterima sampai waktu berakhir, tetapi tidak memperpanjang lagi.
- Perpanjangan disiarkan ke semua penonton.

Kriteria penerimaan:

- Bid pada detik ke-59 memperpanjang lelang.
- Bid di luar jendela 60 detik tidak memperpanjang.
- Perpanjangan berhenti setelah 5 kali.
- Waktu baru terlihat sama di semua client.

### 10.5 Siklus lelang

- Status lelang: draft, live, ended.
- Hanya satu lelang live pada satu waktu.
- Admin menentukan durasi saat membuat draft, antara 1 dan 60 menit.
- Start mengubah draft menjadi live dan mengatur waktu mulai dan waktu berakhir.
- End otomatis terjadi saat waktu berakhir.
- Admin dapat mengakhiri lebih awal.
- End bersifat idempoten.
- Lelang tanpa bid menjadi tidak terjual.

Kriteria penerimaan:

- Draft tidak bisa menerima bid.
- Start kedua pada lelang yang sama ditolak.
- Start draft saat ada lelang live ditolak.
- End dua kali tidak mengubah pemenang atau harga akhir.
- Bid setelah waktu berakhir ditolak.

### 10.6 Admin

- Admin dapat membuat draft dengan field yang sudah disebut.
- Admin dapat preview draft melalui halaman lelang.
- Admin dapat start dan end lelang.
- Admin dapat melihat daftar lelang dengan status, harga, jumlah bid, dan pemenang.
- Admin dapat menghapus draft.
- Admin tidak dapat mengubah harga atau menghapus bid setelah lelang live.
- Aksi admin harus tercatat minimal dalam log aplikasi. Audit trail terpisah masuk Could.

Kriteria penerimaan:

- Draft baru muncul di daftar admin.
- Preview draft menonaktifkan bid.
- Start draft membuat lelang muncul di ruang publik.
- End menampilkan pemenang dan harga akhir.
- Draft yang dihapus tidak lagi muncul.

### 10.7 Realtime dan reconnect

- Client menerima snapshot saat terhubung.
- Setelah snapshot, client menerima event perubahan harga, outbid, presence, perpanjangan, dan akhir lelang.
- Event memiliki sequence dan epoch.
- Jika koneksi terputus, client mencoba reconnect dengan backoff.
- Saat reconnect, client menerima snapshot baru dan tidak memutar ulang event lama.
- Server mengirim heartbeat.
- Outbid dikirim ke koneksi bidder sebelumnya, berdasarkan user ID, bukan display name.

Kriteria penerimaan:

- Dua client menerima urutan event yang sama.
- Client yang reconnect mendapatkan harga dan riwayat terbaru.
- Outbid hanya diterima oleh bidder sebelumnya.
- Event dengan sequence yang meloncat memicu resync.

### 10.8 State dan pesan error

- Halaman menampilkan loading, empty, error, disconnected, ended, dan unsold.
- Pesan error memakai bahasa Indonesia dan menjelaskan tindakan berikutnya.
- Kode error utama: belum login, lelang tidak ditemukan, lelang tidak live, bid terlalu rendah, sudah tertinggi, rate limit, konflik idempotency, validasi gagal, kesalahan server.
- Tidak ada tombol atau tautan yang tidak memiliki perilaku.

Kriteria penerimaan:

- Setiap state dapat dicapai dan diuji.
- Error bid menampilkan angka minimum.
- Tombol yang tidak relevan dinonaktifkan atau disembunyikan.
- Tidak ada console error pada alur normal.

### 10.9 Observability

- Endpoint health check memeriksa proses dan database.
- Metrics mencakup koneksi aktif, bid diterima dan ditolak, latency broadcast, jumlah lelang live, dan reconnect.
- Log terstruktur memuat request ID, user ID, auction ID, bid ID, durasi, dan error code.
- Log tidak memuat password, token, atau email mentah.

Kriteria penerimaan:

- Metrics dapat dibaca dari endpoint Prometheus.
- Log JSON dapat difilter berdasarkan auction ID.
- Health check gagal jika database tidak tersedia.

## 11. Kebutuhan non-fungsional

### 11.1 Performa

Target yang harus diverifikasi dengan load test, bukan klaim hasil:

- 1.000 koneksi WebSocket bersamaan.
- 100 bid per detik selama 60 detik.
- p95 latency broadcast maksimal 250 ms pada mesin lokal 2 vCPU dan 4 GB.
- Error rate di bawah 1 persen pada skenario load test.

Jika target tidak tercapai, laporan harus memuat angka aktual dan bottleneck yang ditemukan.

### 11.2 Keandalan

- Lelang live harus dipulihkan setelah server restart.
- Timer harus tetap mengacu pada waktu berakhir yang tersimpan.
- End lelang harus idempoten.
- Bid tidak boleh hilang setelah commit.
- Snapshot harus cukup untuk menyelaraskan client setelah reconnect.

### 11.3 Keamanan

- Password di-hash dengan bcrypt atau algoritma setara.
- JWT disimpan di cookie httpOnly, SameSite Lax, dan Secure di production.
- Endpoint yang mengubah state memeriksa Origin.
- WebSocket memeriksa Origin saat handshake.
- Rate limit untuk bid dan login.
- Semua query database memakai parameter.
- Admin dibuat dari seed atau CLI, bukan registrasi publik.
- Tidak ada rahasia di log atau response error.

### 11.4 Privasi data

- Data yang disimpan: email, password hash, display name, bid, dan metadata lelang.
- Email tidak pernah ditampilkan publik.
- Display name tampil di riwayat bid.
- Tidak ada data pembayaran di versi pertama.
- Retensi data ditentukan sebelum peluncuran publik.

### 11.5 Aksesibilitas

- Kontras teks memenuhi WCAG AA.
- Semua kontrol dapat dijangkau dan dioperasikan dengan keyboard.
- Focus indicator terlihat.
- Target sentuh minimal 44 px.
- Error tidak disampaikan lewat warna saja.
- Setiap input memiliki label.
- Motion menghormati prefers-reduced-motion.

### 11.6 Responsif

- Layout berfungsi dari 320 px sampai desktop.
- Tidak ada horizontal overflow.
- Tabel atau ledger tidak memotong teks.
- Form tetap dapat digunakan saat keyboard ponsel terbuka.

### 11.7 Kompatibilitas dan lokalisasi

- Browser modern dengan dukungan WebSocket.
- Bahasa Indonesia.
- Mata uang IDR.
- Harga disimpan sebagai integer, bukan float.
- Countdown memakai waktu server. Riwayat bid ditampilkan dalam zona waktu pengguna.

## 12. Metrik keberhasilan

Belum ada data produksi. Angka di bawah adalah target awal, bukan hasil.

| Metrik | Target | Cara mengukur |
|---|---|---|
| p95 latency broadcast | Maksimal 250 ms pada load test lokal | Laporan load test |
| Bid tidak valid yang diterima | 0 | Integration test |
| Pemulihan reconnect | 100 persen skenario uji | Integration test dan Playwright |
| Error rate load test | Di bawah 1 persen | Laporan load test |
| Waktu admin membuat dan memulai lelang | Di bawah 2 menit | Uji usability |
| Lelang dengan minimal satu bid | Diukur setelah ada pengguna nyata | Analitik produk |
| Konversi penonton menjadi bidder | Diukur setelah ada pengguna nyata | Analitik produk |

Metrik produk tidak diberi angka target sampai ada data awal. Menampilkan angka karangan lebih buruk daripada tidak menampilkan apa pun.

### 12.1 Event analitik minimum

- `auction_view`
- `bid_submitted`
- `bid_accepted`
- `bid_rejected` dengan alasan
- `outbid_received`
- `auction_ended` dengan status pemenang atau tidak terjual
- `admin_draft_created`
- `admin_auction_started`
- `admin_auction_ended`

Event ini baru dikirim setelah backend ada. Prototype tidak mengirim analitik.

## 13. Rencana rilis

### Fase 0: Prototype UI

Status: selesai.

- Ruang lelang dan admin berjalan sebagai halaman statis.
- State disinkronkan antar tab lewat BroadcastChannel dan localStorage.
- Bid, anti-sniping, lifecycle, outbid, presence, dan reset disimulasikan.
- Admin dapat create, preview, start, end, delete, dan reset.
- Belum ada backend, database, autentikasi, atau WebSocket.

### Fase 1: MVP backend

Target:

- Go, PostgreSQL, REST untuk command, WebSocket untuk event.
- Auth dan peran.
- Admin lifecycle, termasuk hapus draft.
- Validasi server, idempotency, rate limit.
- Presence dan notifikasi outbid.
- Reconnect dan pemulihan restart.
- Integration test dengan Postgres asli.

Kriteria keluar:

- Semua kriteria penerimaan MVP terpenuhi.
- `go test -race ./...` lulus.
- Playwright lulus untuk alur bidder dan admin.
- Tidak ada bid tidak valid yang diterima pada pengujian.

### Fase 2: Observability dan load test

Target:

- Metrics Prometheus.
- Structured log.
- Health check.
- Load test 1.000 koneksi dan 100 bid per detik.
- Laporan hasil di README.

Kriteria keluar:

- Target performa terukur dan dilaporkan.
- Bottleneck utama terdokumentasi di laporan load test.
- Error rate di bawah 1 persen pada skenario uji.

### Fase 3: Fitur opsional

Kandidat:

- Auto-bid.
- Reserve price.
- Buy-now.
- Grafik harga.
- Share link dan OG image.
- Audit log.
- Edit draft.
- Dark mode.

Fitur di fase ini tidak boleh menunda MVP.

## 14. Risiko dan mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Dua bid bersamaan diproses tidak konsisten | Pemenang salah | Actor per room, transaksi database, row lock, integration test |
| Jam client berbeda | Countdown menyesatkan | Waktu server di snapshot, hitung offset di client |
| Bid menit terakhir membuat hasil tidak adil | Pengalaman buruk | Anti-sniping dengan batas perpanjangan |
| Client kehilangan event saat putus | Harga tertinggal | Snapshot saat reconnect, sequence dan epoch |
| Server restart saat lelang live | Timer berhenti | Muat ulang lelang live dari database saat boot |
| Scope melebar ke marketplace | Proyek tidak selesai | Non-tujuan dan prioritas fitur |
| Admin tanpa autentikasi di prototype | Penyalahgunaan | Label demo, auth wajib di MVP |
| Tidak ada pembayaran | Pemenang harus ditindaklanjuti manual | Nyatakan transaksi selesai di hasil lelang, settlement di luar aplikasi |
| MVP memakai satu instance dan satu lelang live | Belum ada scaling horizontal | Terima batas MVP, siapkan Redis pub/sub sebagai langkah berikutnya |
| Data pribadi di log | Kebocoran | Redaksi email, token, dan password |
| Legalitas untuk publik | Produk tidak bisa diluncurkan | Terms of Service dan Privacy Policy sebelum publik |

## 15. Pertanyaan terbuka

1. Nama produk final dan identitas merek.
2. Apakah admin versi pertama cukup menjadi penjual, atau perlu role penjual terpisah?
3. Apakah pemenang dihubungi lewat email di luar aplikasi, atau hasil lelang cukup ditampilkan dan diselesaikan di luar aplikasi?
4. Apakah perlu verifikasi email sebelum bid?
5. Berapa lama data lelang dan bid disimpan?
6. Apakah reserve price atau auto-bid lebih penting daripada grafik harga setelah MVP?
7. Apakah multi-lelang bersamaan dibutuhkan setelah MVP, atau satu lelang per waktu sudah cukup untuk waktu lama?
8. Apakah perlu Terms of Service dan Privacy Policy sebelum demo publik?

Keputusan yang sudah diambil:

- Satu lelang live pada satu waktu.
- Guest boleh menonton, bid harus login.
- Display name boleh sama. Outbid ditargetkan berdasarkan user ID.
- Bid tidak dapat ditarik kembali.
- Admin tidak boleh bid pada lelang yang dia buat.
- Harga dalam integer rupiah.
- Tidak ada pembayaran di dalam aplikasi pada versi pertama.

## 16. Kriteria penerimaan MVP

1. User dapat register, login, logout, dan sesi bertahan setelah refresh.
2. Guest dapat menonton lelang dan diarahkan ke login saat bid.
3. Dua browser menampilkan harga, bidder tertinggi, dan riwayat bid yang sama.
4. Bid di bawah minimum, bid setelah lelang berakhir, bid dari bidder tertinggi, dan bid dari pembuat lelang ditolak dengan kode yang benar.
5. Dua bid bersamaan pada nominal yang sama menghasilkan tepat satu bid tersimpan.
6. Bid dalam 60 detik terakhir memperpanjang lelang 60 detik, maksimal 5 kali.
7. Lelang berakhir otomatis dengan pemenang dan harga akhir, atau tidak terjual jika tidak ada bid.
8. Client yang reconnect menerima snapshot dan tidak kehilangan bid.
9. Jumlah penonton tampil dan notifikasi outbid diterima oleh bidder sebelumnya.
10. Admin dapat membuat draft, preview, start, end, melihat hasil, dan menghapus draft.
11. Hanya satu lelang live pada satu waktu.
12. Server restart saat lelang live memulihkan timer dan state dari database.
13. `go test -race ./...` dan Playwright lulus.
14. Halaman lolos pemeriksaan kontras, keyboard, target sentuh, dan layout 320 px.

## 17. Status prototipe saat ini

| Area | Status |
|---|---|
| Ruang lelang | UI dan alur lokal berfungsi |
| Bid dan validasi | Simulasi di browser |
| Anti-sniping | Simulasi di browser |
| Admin | Lokal, tanpa autentikasi |
| Sinkronisasi | BroadcastChannel dan localStorage |
| Backend, database, WebSocket | Belum dibuat |
| Auth dan role | Belum dibuat |
| Metrics dan load test | Belum dibuat |
| Verifikasi prototype | 22 cek ruang lelang dan 15 cek admin PASS, tanpa console error |
| Pembayaran dan ongkir | Di luar scope |

Prototipe ini dipakai untuk menguji urutan layar, kejelasan aturan, dan perilaku state. Prototipe bukan bukti keamanan, skalabilitas, atau kesiapan produksi.
