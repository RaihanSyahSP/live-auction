# Migrations

Skema target ada di:

`docs/superpowers/specs/2026-10-20-auction-backend-design.md`

Buat file migration dengan pola:

```text
000001_init.up.sql
000001_init.down.sql
```

Urutan tabel:

1. `users`
2. `auctions`
3. `bids`
4. index dan constraint

Tool yang dipakai di plan:

```powershell
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
migrate -path migrations -database "$env:DATABASE_URL" up
migrate -path migrations -database "$env:DATABASE_URL" down 1
```

Jangan menaruh password asli di SQL migration atau di repo.
