# Auction Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Execution mode for this project:** manual by the project owner. Do not dispatch subagents or implement every task in one pass.

**Goal:** Membangun backend Go untuk LELANG dengan auth access token dan refresh token, lelang, bid realtime, admin lifecycle, dan observability.

**Architecture:** HTTP API memakai `net/http`, PostgreSQL sebagai sumber kebenaran, transaksi bid dengan `SELECT FOR UPDATE`, room manager untuk fan-out WebSocket, dan scheduler untuk menutup lelang. Layer mengikuti controller, service, repository, model, middleware, exception, dan helper seperti repo referensi `golang-restful`, tetapi tanpa Wire, httprouter, lib/pq, atau ORM.

**Tech Stack:** Go 1.26, PostgreSQL 18, pgx/v5, coder/websocket, golang-jwt/v5, bcrypt, validator/v10, x/time/rate, prometheus/client_golang, golang-migrate, google/uuid.

**Spec:** `docs/superpowers/specs/2026-10-20-auction-backend-design.md`

## Global Constraints

- Go 1.26.5.
- PostgreSQL 18 untuk default `uuidv7()`.
- Tanpa ORM, Wire, httprouter, atau lib/pq.
- Semua primary key dan foreign key bertipe `uuid`, default `uuidv7()`. Tidak ada ID `bigint`.
- Semua harga integer rupiah, disimpan sebagai `bigint`.
- Semua timestamp UTC.
- Access token JWT HS256, 15 menit.
- Refresh token opaque 32 byte, 7 hari, disimpan sebagai hash SHA-256.
- Kedua token disimpan di cookie `httpOnly`, `SameSite=Lax`, `Secure` di production.
- Refresh token tidak boleh masuk `localStorage`.
- Satu lelang live pada satu waktu.
- Server adalah sumber kebenaran waktu, harga, dan pemenang.
- Endpoint yang mengubah state memeriksa `Origin`.
- Semua query memakai parameter pgx.
- Tidak ada secret, password, token, atau cookie di log.
- Setiap task lulus `go test -race ./...` sebelum commit.

## Review Focus

1. Dua bid bersamaan pada nominal yang sama: hanya satu yang tersimpan, yang lain mendapat `BID_TOO_LOW`, dan harga akhir konsisten.
2. Bid tepat di batas `ends_at` atau batas 60 detik anti-sniping: server memakai jam server, bid telat ditolak, perpanjangan berhenti di 5 kali.
3. Replay idempotency dengan key sama: amount sama mengembalikan bid lama, amount berbeda menghasilkan `IDEMPOTENCY_CONFLICT`, dan tidak ada baris ganda.
4. Refresh token dipakai dua kali: reuse di luar grace mencabut family, sedangkan race dalam grace menghasilkan `REFRESH_RACE` tanpa logout massal.
5. Admin start lelang kedua, end lebih awal, dan restart server: satu live tetap terjaga, end idempoten, dan timer dipulihkan.

## File Structure

```text
server/
  cmd/
    api/main.go
    seed/main.go
  internal/
    config/config.go
    app/server.go
    app/router.go
    app/database.go
    app/validator.go
    app/scheduler.go
    auth/access.go
    auth/refresh.go
    auth/cookies.go
    controller/health_controller.go
    controller/auth_controller.go
    controller/auction_controller.go
    controller/admin_controller.go
    controller/bid_controller.go
    controller/websocket_controller.go
    service/auth_service.go
    service/auction_service.go
    service/bid_service.go
    repository/tx.go
    repository/user_repository.go
    repository/auction_repository.go
    repository/bid_repository.go
    repository/refresh_token_repository.go
    model/domain/user.go
    model/domain/auction.go
    model/domain/bid.go
    model/domain/refresh_token.go
    model/web/auth_request.go
    model/web/auth_response.go
    model/web/auction_request.go
    model/web/auction_response.go
    model/web/bid_request.go
    model/web/bid_response.go
    middleware/auth.go
    middleware/admin.go
    middleware/origin.go
    middleware/rate_limit.go
    middleware/logging.go
    exception/errors.go
    exception/error_handler.go
    helper/json.go
    helper/testdb.go
    helper/uuid.go
    realtime/event.go
    realtime/client.go
    realtime/room.go
    realtime/manager.go
    auction/rules.go
  migrations/
    000001_init.up.sql
    000001_init.down.sql
  docker-compose.yml
  loadtest/main.go
  .env.example
  go.mod
  README.md
```

## Task 1: Config, HTTP server, health endpoint

**Files:**
- Create: `server/internal/config/config.go`
- Create: `server/internal/config/config_test.go`
- Create: `server/internal/app/server.go`
- Create: `server/internal/app/router.go`
- Create: `server/internal/controller/health_controller.go`
- Create: `server/internal/controller/health_controller_test.go`
- Modify: `server/cmd/api/main.go`

**Interfaces:**
- Produces: `config.Load() (config.Config, error)`
- Produces: `config.Config{Port, DatabaseURL, AppOrigin, JWTSecret string; AccessTokenTTL, RefreshTokenTTL, RefreshGracePeriod time.Duration; CookieSecure bool}`
- Produces: `app.NewRouter(health *controller.HealthController) http.Handler`
- Produces: `app.NewServer(cfg config.Config, handler http.Handler) *http.Server`
- Produces: `controller.NewHealthController() *controller.HealthController`
- Produces: `(*controller.HealthController) Health(w http.ResponseWriter, r *http.Request)`

- [ ] **Step 1: Write config tests**

Test cases:
- `TestLoadConfigRejectsMissingRequiredValues`: unset `DATABASE_URL`, `APP_ORIGIN`, or `JWT_SECRET` returns error.
- `TestLoadConfigRejectsShortJWTSecret`: secret shorter than 32 bytes returns error.
- `TestLoadConfigParsesDurations`: `ACCESS_TOKEN_TTL=15m`, `REFRESH_TOKEN_TTL=168h`, `REFRESH_GRACE_PERIOD=10s` parse correctly.
- `TestLoadConfigRejectsInvalidDuration`: `ACCESS_TOKEN_TTL=abc` returns error.

Use `t.Setenv` for each value.

- [ ] **Step 2: Run config tests to verify they fail**

Run: `go test ./internal/config -v`
Expected: FAIL because `Load` does not exist.

- [ ] **Step 3: Implement `config.Load`**

Read environment variables from the spec. Apply defaults for `PORT`, TTLs, grace period, and `COOKIE_SECURE`. Validate required values and JWT secret length. Parse durations with `time.ParseDuration`. Return a typed error listing the invalid variable.

- [ ] **Step 4: Write health endpoint test**

Test `GET /healthz` through `httptest.NewServer(app.NewRouter(...))`:
- status 200;
- `Content-Type` contains `application/json`;
- body is `{"status":"ok"}`.

- [ ] **Step 5: Run health test to verify it fails**

Run: `go test ./internal/controller ./internal/app -v`
Expected: FAIL because router and controller do not exist.

- [ ] **Step 6: Implement server, router, health controller, and main**

`NewRouter` uses `http.NewServeMux` and registers:
```go
mux.HandleFunc("GET /healthz", health.Health)
```
`NewServer` sets `ReadHeaderTimeout`, `IdleTimeout`, and `Addr` from config. `cmd/api/main.go` loads config, builds the router, logs the address, and calls `ListenAndServe` with graceful shutdown on `os.Interrupt`.

- [ ] **Step 7: Verify and commit**

```powershell
go build ./...
go test ./... -race
git add server
git commit -m "feat(server): bootstrap config and health endpoint"
```

## Task 2: PostgreSQL pool, migration, and schema constraints

**Files:**
- Create: `server/docker-compose.yml`
- Create: `server/migrations/000001_init.up.sql`
- Create: `server/migrations/000001_init.down.sql`
- Create: `server/internal/app/database.go`
- Create: `server/internal/helper/testdb.go`
- Create: `server/internal/repository/schema_test.go`

**Interfaces:**
- Produces: `app.NewDB(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error)`
- Produces: `helper.OpenTestDB(t *testing.T) *pgxpool.Pool`
- Produces: `helper.TruncateAll(t *testing.T, pool *pgxpool.Pool)`

- [ ] **Step 1: Add dependencies**

```powershell
go get github.com/jackc/pgx/v5@latest
go get github.com/golang-migrate/migrate/v4@latest
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
go get github.com/google/uuid@latest
```

- [ ] **Step 2: Create Docker Compose for PostgreSQL**

Use `postgres:18`, user `postgres`, password `postgres`, database `lelang`, port 5432, and a named volume. Add a comment showing how to create `lelang_test`.

- [ ] **Step 3: Write the failing schema tests**

Create `schema_test.go` with tests:
- `TestSchemaTablesExist`: query `information_schema.tables` for `users`, `auctions`, `bids`, `refresh_tokens`.
- `TestOnlyOneLiveAuction`: insert two auctions with status `live`, expect unique violation on the second.
- `TestEmailIsUniqueCaseInsensitive`: insert `A@Example.com` and `a@example.com`, expect unique violation.
- `TestIdempotencyIsUniquePerAuction`: insert same `(auction_id, user_id, idempotency_key)` twice, expect unique violation.
- `TestRefreshTokenHashIsUnique`: insert same `token_hash` twice, expect unique violation.

Use `helper.OpenTestDB(t)` and `helper.TruncateAll(t, pool)`. Create a test helper that inserts a valid user first, then a valid auction with every required column. Generate fixture IDs with `uuid.New()` and insert them explicitly. `OpenTestDB` must resolve the migrations directory with `runtime.Caller` so it works regardless of the test working directory.

- [ ] **Step 4: Run schema tests to verify they fail**

Start Postgres and create the test database:
```powershell
docker compose up -d postgres
docker exec -it <container> createdb -U postgres lelang_test
$env:TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/lelang_test?sslmode=disable"
go test ./internal/repository -run 'TestSchema|TestOnlyOneLive|TestEmail|TestIdempotency|TestRefresh' -v
```
Expected: FAIL because tables and helper do not exist.

- [ ] **Step 5: Write the migration**

Keep the only migration directory at `server/migrations`. Do not create or use a top-level `db/` folder.

```powershell
cd D:\live-auction\server
migrate create -ext sql -dir migrations -seq init
```

This creates `000001_init.up.sql` and `000001_init.down.sql`.

Translate the DDL in the spec exactly, including:
- all four tables: `users`, `auctions`, `bids`, `refresh_tokens`;
- all primary keys and foreign keys as `uuid` with `default uuidv7()`;
- `one_live_auction` partial unique index;
- `users_email_lower_idx`;
- `bids_idempotency_idx`;
- indexes for auction, user, and refresh token queries;
- all check constraints.

The down migration drops in reverse order: `refresh_tokens`, `bids`, `auctions`, `users`.

- [ ] **Step 6: Implement database pool and test helper**

`NewDB` uses `pgxpool.New`, sets max connections, and pings with a 5 second timeout. `OpenTestDB` skips the test if `TEST_DATABASE_URL` is empty, runs migrations once with `sync.Once`, and returns a pool. `TruncateAll` runs:
```sql
TRUNCATE refresh_tokens, bids, auctions, users CASCADE;
```

- [ ] **Step 7: Run migration and schema tests**

```powershell
migrate -path migrations -database "$env:TEST_DATABASE_URL" up
go test ./internal/repository -run 'TestSchema|TestOnlyOneLive|TestEmail|TestIdempotency|TestRefresh' -v
```
Expected: PASS.

- [ ] **Step 8: Commit**

```powershell
git add server
git commit -m "feat(server): add postgres schema and constraints"
```

## Task 3: Domain models and user/auction repositories

**Files:**
- Create: `server/internal/model/domain/user.go`
- Create: `server/internal/model/domain/auction.go`
- Create: `server/internal/model/domain/bid.go`
- Create: `server/internal/model/domain/refresh_token.go`
- Create: `server/internal/repository/tx.go`
- Create: `server/internal/repository/user_repository.go`
- Create: `server/internal/repository/auction_repository.go`
- Create: `server/internal/repository/user_repository_test.go`
- Create: `server/internal/repository/auction_repository_test.go`
- Create: `server/internal/exception/errors.go`
- Create: `server/internal/helper/uuid.go`

**Interfaces:**
- Produces: `domain.Role`, `domain.RoleUser`, `domain.RoleAdmin`
- Produces: `domain.User`, `domain.Auction`, `domain.Bid`, `domain.BidView`, `domain.RefreshToken`, `domain.Spec`
- Produces: `exception` sentinel errors for all codes in the spec.
- Produces: `repository.TxManager` with `WithinTx(ctx context.Context, fn func(pgx.Tx) error) error`
- Produces: `repository.UserRepository`:
  - `Create(ctx, tx, user) (domain.User, error)`
  - `FindByEmail(ctx, tx, email) (domain.User, error)`
  - `FindByID(ctx, tx, id uuid.UUID) (domain.User, error)`
- Produces: `repository.AuctionRepository`:
  - `Create(ctx, tx, auction) (domain.Auction, error)`
  - `FindByID(ctx, tx, id uuid.UUID) (domain.Auction, error)`
  - `FindCurrentLive(ctx, tx) (domain.Auction, error)`
  - `List(ctx, tx) ([]domain.Auction, error)`
  - `ListEnded(ctx, tx, limit int) ([]domain.Auction, error)`
  - `LockByID(ctx, tx, id uuid.UUID) (domain.Auction, error)`
  - `Update(ctx, tx, auction) (domain.Auction, error)`
  - `DeleteDraft(ctx, tx, id uuid.UUID) error`
  - `ListDue(ctx, tx, now time.Time) ([]domain.Auction, error)`

- [ ] **Step 1: Write domain models and repository interfaces**

Use the fields from the spec. All domain IDs are `uuid.UUID`. `Auction` must include `ID uuid.UUID`, `CreatedBy uuid.UUID`, `Version int64`, `HighestBidderID *uuid.UUID`, `WinnerID *uuid.UUID`, `FinalPrice *int64`, and nullable time pointers. `Bid` has `ID`, `AuctionID`, and `UserID` as `uuid.UUID`. `RefreshToken` has `ID`, `UserID`, and `FamilyID` as `uuid.UUID`, plus `ReplacedByID *uuid.UUID`. `BidView` includes `Bid` fields plus `DisplayName string`. All ID fields in web DTOs are `uuid.UUID` and serialize as strings.

Add `helper.UUIDToPg(uuid.UUID) pgtype.UUID`, `helper.PgToUUID(pgtype.UUID) uuid.UUID`, `helper.UUIDPtrToPg(*uuid.UUID) pgtype.UUID`, and `helper.PgToUUIDPtr(pgtype.UUID) *uuid.UUID`. Repositories scan UUID columns through `pgtype.UUID` and convert explicitly.

- [ ] **Step 2: Write failing user repository tests**

Use `uuid.New()` for fixture IDs. Tests:
- `TestCreateUserAndFindByEmail` lowercases email at the service boundary and returns the stored user.
- `TestCreateUserRejectsDuplicateEmail` expects a unique violation mapped to a typed error.
- `TestFindUserNotFound` expects `exception.ErrUserNotFound`.

- [ ] **Step 3: Run to verify failure**

Run: `go test ./internal/repository -run 'TestCreateUser|TestFindUser' -v`
Expected: FAIL.

- [ ] **Step 4: Implement user repository**

Use `pgx.Tx` and parameterized SQL. Scan all columns explicitly. Use `RETURNING` to get the generated UUID. Scan UUID columns through `pgtype.UUID` and convert with `helper.PgToUUID`. Return typed not-found errors from `exception`.

- [ ] **Step 5: Write failing auction repository tests**

Use `uuid.New()` for fixture IDs. Tests:
- create draft and find by UUID;
- find current live;
- list all sorted by creation date;
- list ended limited by limit;
- `DeleteDraft` only deletes draft;
- `LockByID` blocks a concurrent update;
- `ListDue` returns live auctions with `ends_at <= now`.

- [ ] **Step 6: Implement auction repository**

Implement all methods with explicit columns and `RETURNING`. Use `helper.UUIDToPg` for UUID parameters and `helper.PgToUUID` for scans. `LockByID` uses `SELECT ... FOR UPDATE`. `Update` writes mutable fields including `version` and returns the updated row. `ListDue` filters `status = 'live' AND ends_at <= $1`.

- [ ] **Step 7: Verify and commit**

```powershell
go test ./internal/repository -race -v
git add server
git commit -m "feat(server): add domain models and user auction repositories"
```

## Task 4: Bid and refresh token repositories

**Files:**
- Create: `server/internal/repository/bid_repository.go`
- Create: `server/internal/repository/refresh_token_repository.go`
- Create: `server/internal/repository/bid_repository_test.go`
- Create: `server/internal/repository/refresh_token_repository_test.go`

**Interfaces:**
- Produces: `BidRepository.Insert(ctx, tx, bid) (domain.Bid, error)`
- Produces: `BidRepository.FindByIdempotencyKey(ctx, tx, auctionID, userID uuid.UUID, key uuid.UUID) (domain.Bid, error)`
- Produces: `BidRepository.ListByAuction(ctx, tx, auctionID uuid.UUID, beforeID *uuid.UUID, limit int) ([]domain.BidView, error)`
- Produces: `RefreshTokenRepository.Create(ctx, tx, token) (domain.RefreshToken, error)`
- Produces: `RefreshTokenRepository.FindByHashForUpdate(ctx, tx, hash []byte) (domain.RefreshToken, error)`
- Produces: `RefreshTokenRepository.Rotate(ctx, tx, oldID, newID uuid.UUID, rotatedAt time.Time) error`
- Produces: `RefreshTokenRepository.RevokeFamily(ctx, tx, familyID uuid.UUID, revokedAt time.Time) error`
- Produces: `RefreshTokenRepository.DeleteExpired(ctx, tx, before time.Time) error`

- [ ] **Step 1: Confirm UUID dependency**

`github.com/google/uuid` is already added in Task 2. No new dependency is needed.

- [ ] **Step 2: Write failing bid repository tests**

Tests:
- insert bid and read it by idempotency key;
- duplicate key on same auction returns the existing row;
- same key on a different auction is allowed;
- `ListByAuction` joins users for display names, orders by `id desc`, and honors `before_id`;
- use explicit UUIDv7 values so the `id desc` ordering test is deterministic;
- invalid auction ID returns a foreign key error.

- [ ] **Step 3: Write failing refresh token repository tests**

Tests:
- create token and find by hash;
- `FindByHashForUpdate` returns rotated/revoked timestamps;
- rotate sets `rotated_at` and `replaced_by_id`;
- revoke family sets `revoked_at` on all family rows;
- delete expired removes only expired rows.

- [ ] **Step 4: Implement both repositories**

Use parameterized SQL and explicit scans. Use UUID helpers for IDs and nullable UUIDs. Use `RETURNING id` after inserting bids and refresh tokens. `FindByHashForUpdate` uses `SELECT ... FOR UPDATE`. `ListByAuction` joins `users` and returns `domain.BidView`.

- [ ] **Step 5: Verify and commit**

```powershell
go test ./internal/repository -race -v
git add server
git commit -m "feat(server): add bid and refresh token repositories"
```

## Task 5: Access token, refresh token, and cookie primitives

**Files:**
- Create: `server/internal/auth/access.go`
- Create: `server/internal/auth/refresh.go`
- Create: `server/internal/auth/cookies.go`
- Create: `server/internal/auth/access_test.go`
- Create: `server/internal/auth/refresh_test.go`
- Create: `server/internal/auth/cookies_test.go`

**Interfaces:**
- Produces: `auth.Principal{UserID uuid.UUID; Role domain.Role}`
- Produces: `auth.NewAccessTokens(secret, issuer, audience string, ttl time.Duration) *auth.AccessTokens`
- Produces: `(*auth.AccessTokens).Issue(userID uuid.UUID, role domain.Role) (string, error)`
- Produces: `(*auth.AccessTokens).Parse(raw string) (auth.Principal, error)`
- Produces: `auth.GenerateRefreshToken() (raw string, hash []byte, error)`
- Produces: `auth.HashRefreshToken(raw string) []byte`
- Produces: `auth.NewCookies(secure bool, accessTTL, refreshTTL time.Duration) auth.Cookies`
- Produces: `(auth.Cookies).Set(w http.ResponseWriter, accessToken, refreshToken string)`
- Produces: `(auth.Cookies).Clear(w http.ResponseWriter)`
- Produces: `auth.AccessCookieName = "access_token"` and `auth.RefreshCookieName = "refresh_token"`

- [ ] **Step 1: Add JWT dependency**

```powershell
go get github.com/golang-jwt/jwt/v5@latest
```

- [ ] **Step 2: Write failing token tests**

Access token tests:
- issue and parse returns the same `uuid.UUID` user ID and role;
- wrong secret fails;
- expired token fails with `exception.ErrTokenExpired`;
- malformed token fails with `exception.ErrTokenInvalid`.

Refresh token tests:
- token is 43 characters base64url, hash is 32 bytes;
- hashing the same token twice returns the same bytes;
- two generated tokens are different.

- [ ] **Step 3: Implement access and refresh token helpers**

Use `jwt.NewWithClaims(jwt.SigningMethodHS256, claims)`. Put the UUID user ID in `sub` as a string. Validate algorithm, issuer, audience, and expiry. Parse `sub` with `uuid.Parse`. Use `crypto/rand` with 32 bytes for refresh tokens. Hash with `sha256.Sum256`.

- [ ] **Step 4: Write failing cookie tests**

Use `httptest.NewRecorder` and inspect `Set-Cookie`:
- access cookie path `/`, refresh cookie path `/api/auth`;
- both `HttpOnly`;
- `SameSite=Lax`;
- `Secure` follows config;
- `MaxAge` matches TTL;
- `Clear` sets `MaxAge=-1` and empty value with the same paths.

- [ ] **Step 5: Implement cookie helper and verify**

```powershell
go test ./internal/auth -race -v
git add server
git commit -m "feat(server): add access refresh tokens and cookies"
```

## Task 6: Auth service with refresh rotation

**Files:**
- Create: `server/internal/model/web/auth_request.go`
- Create: `server/internal/model/web/auth_response.go`
- Create: `server/internal/service/auth_service.go`
- Create: `server/internal/service/auth_service_test.go`

**Interfaces:**
- Produces: `web.RegisterRequest{Email, Password, DisplayName string}`
- Produces: `web.LoginRequest{Email, Password string}`
- Produces: `web.UserResponse{ID uuid.UUID; Email, DisplayName, Role string}`
- Produces: `web.AuthTokens{User web.UserResponse; AccessToken, RefreshToken string}`
- Produces: `service.AuthService` interface for register, login, refresh, logout, and me.
- Produces: `service.NewAuthService(users repository.UserRepository, refresh repository.RefreshTokenRepository, tx repository.TxManager, tokens *auth.AccessTokens, now func() time.Time, refreshTTL, grace time.Duration) service.AuthService`
- Produces: `service.AuthService.Register(ctx, req web.RegisterRequest) (web.AuthTokens, error)`
- Produces: `service.AuthService.Login(ctx, req web.LoginRequest) (web.AuthTokens, error)`
- Produces: `service.AuthService.Refresh(ctx, rawRefresh string) (web.AuthTokens, error)`
- Produces: `service.AuthService.Logout(ctx, rawRefresh string) error`
- Produces: `service.AuthService.Me(ctx, userID uuid.UUID) (web.UserResponse, error)`

- [ ] **Step 1: Add bcrypt and validator dependencies**

```powershell
go get golang.org/x/crypto@latest
go get github.com/go-playground/validator/v10@latest
```

- [ ] **Step 2: Write failing register/login tests**

Tests:
- register hashes the password, creates a user with role `user`, creates one refresh token with a new family ID, and returns two tokens;
- register returns the UUID generated by the database;
- duplicate email returns `exception.ErrEmailTaken`;
- login with wrong password returns `exception.ErrInvalidCredentials`;
- login sets a new refresh family.

- [ ] **Step 3: Implement register and login**

Use `bcrypt.GenerateFromPassword` and `bcrypt.CompareHashAndPassword`. Normalize email to lowercase and trim. Validate with `validator`. Insert the user without an ID and use `RETURNING id` to get the UUID. Generate the refresh token, store `SHA-256`, set `expires_at = now + refreshTTL`, and issue the access token.

- [ ] **Step 4: Write failing refresh rotation tests**

Tests:
- refresh with a valid token creates a new token in the same family and marks the old token rotated;
- old access token remains parseable until expiry;
- expired refresh returns `REFRESH_INVALID`;
- revoked refresh returns `REFRESH_INVALID`;
- reuse after grace returns `REFRESH_REUSED` and revokes the family;
- reuse within grace returns `REFRESH_RACE` and does not revoke;
- two concurrent refreshes with the same token: one succeeds, the other gets `REFRESH_RACE`.

- [ ] **Step 5: Implement refresh rotation and logout**

Inside one transaction:
1. hash the raw token;
2. `FindByHashForUpdate`;
3. validate expiry and revocation;
4. handle rotated token with grace;
5. create a new token in the same family;
6. `Rotate(oldID, newID, now)`;
7. commit;
8. issue a new access token.

Logout revokes the entire family and returns nil even if the token is already invalid.

- [ ] **Step 6: Verify and commit**

```powershell
go test ./internal/service -race -v
git add server
git commit -m "feat(server): add auth service and refresh rotation"
```

## Task 7: Auth controller, middleware, and routes

**Files:**
- Create: `server/internal/controller/auth_controller.go`
- Create: `server/internal/middleware/auth.go`
- Create: `server/internal/middleware/admin.go`
- Create: `server/internal/middleware/origin.go`
- Create: `server/internal/middleware/rate_limit.go`
- Create: `server/internal/exception/error_handler.go`
- Create: `server/internal/helper/json.go`
- Modify: `server/internal/app/router.go`
- Modify: `server/cmd/api/main.go`
- Create: `server/internal/controller/auth_controller_test.go`

**Interfaces:**
- Consumes: `exception` sentinel errors from Task 3.
- Produces: `helper.ReadJSON(w, r, dst) error` and `helper.WriteJSON(w, status, payload)`.
- Produces: `middleware.Auth(tokens *auth.AccessTokens, next http.Handler) http.Handler`
- Produces: `middleware.Admin(next http.Handler) http.Handler`
- Produces: `middleware.Origin(allowedOrigin string, next http.Handler) http.Handler`
- Produces: `middleware.NewRateLimiter(limit rate.Limit, burst int) *middleware.RateLimiter`
- Produces: `(*middleware.RateLimiter).Allow(key string) bool`
- Produces: `middleware.RateLimit(limiter *middleware.RateLimiter, keyFunc func(*http.Request) string, next http.Handler) http.Handler`
- Produces: `middleware.PrincipalFrom(ctx) (auth.Principal, bool)`
- Produces: `controller.NewAuthController(authService service.AuthService, cookies auth.Cookies) *controller.AuthController`
- Produces: `(*controller.AuthController).Register/Login/Refresh/Logout/Me`

- [ ] **Step 1: Add rate limit dependency**

```powershell
go get golang.org/x/time@latest
```

- [ ] **Step 2: Write failing controller tests**

Tests:
- register returns 201 and two `Set-Cookie` headers;
- login returns 200 and user data;
- me returns 200 with a valid access cookie and 401 without it;
- refresh returns 200 with rotated cookies;
- refresh with an invalid token returns 401 `REFRESH_INVALID`;
- admin endpoint without admin role returns 403;
- state-changing request with a wrong `Origin` returns 403;
- malformed JSON returns 422 `VALIDATION_ERROR`.

Use `httptest` and a fake `AuthService` where needed.

- [ ] **Step 3: Implement error mapping and JSON helpers**

Map typed errors to the error envelope and status codes from the spec. `WriteJSON` sets `Content-Type: application/json` and encodes once. `ReadJSON` limits body size to 1 MB and disallows unknown fields.

- [ ] **Step 4: Implement middleware and auth routes**

Routes:
```text
POST /api/auth/register
POST /api/auth/login
POST /api/auth/refresh
POST /api/auth/logout
GET  /api/auth/me
```
Apply `Origin` to all state-changing routes. Allow an empty Origin for API clients and curl, but reject a non-empty Origin that does not match `APP_ORIGIN`. Apply `Auth` to `/me`. Logout reads the refresh cookie without requiring a valid access token, because the access token may already be expired. Parse the access token from the cookie, put `Principal` in context, and return 401 with `TOKEN_EXPIRED` when expired. Apply login rate limiting at 10 attempts per minute per IP and refresh rate limiting at 30 attempts per minute per IP. Keep the bid limiter at 3 per second with burst 5 for Task 9.

- [ ] **Step 5: Verify and commit**

```powershell
go test ./internal/controller ./internal/middleware -race -v
git add server
git commit -m "feat(server): add auth endpoints and middleware"
```

## Task 8: Auction rules and admin/public auction endpoints

**Files:**
- Create: `server/internal/auction/rules.go`
- Create: `server/internal/auction/rules_test.go`
- Create: `server/internal/model/web/auction_request.go`
- Create: `server/internal/model/web/auction_response.go`
- Create: `server/internal/service/auction_service.go`
- Create: `server/internal/service/auction_service_test.go`
- Create: `server/internal/controller/auction_controller.go`
- Create: `server/internal/controller/admin_controller.go`
- Create: `server/internal/controller/auction_controller_test.go`
- Modify: `server/internal/app/router.go`

**Interfaces:**
- Produces: `auction.MinimumBid(a domain.Auction) int64`
- Produces: `auction.NextEndsAt(now, endsAt time.Time, extensions int) (time.Time, bool)`
- Produces: `service.AuctionService` interface for read, admin lifecycle, and close-due use cases.
- Produces: `service.NewAuctionService(auctions repository.AuctionRepository, bids repository.BidRepository, tx repository.TxManager, now func() time.Time) service.AuctionService`
- Produces: `service.AuctionService.Current(ctx) (web.AuctionDetailResponse, error)`
- Produces: `service.AuctionService.Detail(ctx, id uuid.UUID) (web.AuctionDetailResponse, error)`
- Produces: `service.AuctionService.Bids(ctx, auctionID uuid.UUID, beforeID *uuid.UUID, limit int) ([]web.BidResponse, error)`
- Produces: `service.AuctionService.Results(ctx, limit int) ([]web.ResultResponse, error)`
- Produces: `service.AuctionService.List(ctx) ([]web.AuctionResponse, error)`
- Produces: `service.AuctionService.CreateDraft(ctx, adminID uuid.UUID, req web.CreateAuctionRequest) (web.AuctionResponse, error)`
- Produces: `service.AuctionService.Start(ctx, adminID uuid.UUID, id uuid.UUID) (web.AuctionResponse, error)`
- Produces: `service.AuctionService.End(ctx, adminID uuid.UUID, id uuid.UUID) (web.AuctionResponse, error)`
- Produces: `service.AuctionService.DeleteDraft(ctx, adminID uuid.UUID, id uuid.UUID) error`
- Produces: `service.AuctionService.CloseDue(ctx) ([]domain.Auction, error)`
- Produces: `controller.NewAuctionController(service service.AuctionService) *controller.AuctionController`
- Produces: `controller.NewAdminController(service service.AuctionService) *controller.AdminController`

- [ ] **Step 1: Write failing auction rule tests**

Tests:
- first bid minimum equals `starting_price`;
- later bid minimum equals `current_price + min_increment`;
- `NextEndsAt` extends 60 seconds inside the last 60 seconds;
- no extension outside the window;
- no extension after 5 extensions;
- boundary at exactly 60 seconds is treated consistently with the spec.

- [ ] **Step 2: Implement pure auction rules**

Keep `rules.go` free of database and HTTP dependencies. Use server time passed by the caller.

- [ ] **Step 3: Write failing auction service tests**

Use `uuid.New()` for fixture IDs. Tests:
- create draft with normalized arrays and specs;
- start draft sets live, starts_at, ends_at, version, and clears winner/final price;
- start rejects when another auction is live;
- start rejects non-draft;
- end sets winner, final price, ended_at, version, and is idempotent;
- end with no bids sets winner nil and final price nil;
- delete only draft;
- current returns live or not found;
- detail returns auction and recent bids for a UUID;
- detail with an unknown UUID returns `AUCTION_NOT_FOUND`;
- results returns ended auctions ordered by ended_at desc;
- `CloseDue` ends only live auctions with `ends_at <= now`.

- [ ] **Step 4: Implement auction service**

Use `TxManager.WithinTx` and repositories. Increment `version` on every start, end, and bid update. Use typed errors from `exception`. Map a `one_live_auction` unique violation to `AUCTION_ALREADY_LIVE`. End must return the existing ended auction when called twice, not an error.

- [ ] **Step 5: Implement controllers and routes**

Routes:
```text
GET    /api/auctions/current
GET    /api/auctions/{id}
GET    /api/auctions/{id}/bids
GET    /api/results
GET    /api/admin/auctions
POST   /api/admin/auctions
POST   /api/admin/auctions/{id}/start
POST   /api/admin/auctions/{id}/end
DELETE /api/admin/auctions/{id}
```
Apply `Auth` and `Admin` to admin routes. Return 204 for current when no live auction. Parse every `{id}` and `before_id` with `uuid.Parse`; invalid UUID returns 422 `VALIDATION_ERROR`. Pass the admin UUID from `Principal` into service calls.

- [ ] **Step 6: Verify and commit**

```powershell
go test ./internal/auction ./internal/service ./internal/controller -race -v
git add server
git commit -m "feat(server): add auction lifecycle endpoints"
```

## Task 9: Bid service and controller

**Files:**
- Create: `server/internal/model/web/bid_request.go`
- Create: `server/internal/model/web/bid_response.go`
- Create: `server/internal/service/bid_service.go`
- Create: `server/internal/service/bid_service_test.go`
- Create: `server/internal/controller/bid_controller.go`
- Create: `server/internal/controller/bid_controller_test.go`
- Modify: `server/internal/app/router.go`

**Interfaces:**
- Produces: `web.PlaceBidRequest{Amount int64; IdempotencyKey uuid.UUID}`
- Produces: `web.PlaceBidResponse{Auction web.AuctionResponse; Bid web.BidResponse; Replay bool; Extended bool; PreviousHighestUserID *uuid.UUID}`
- Produces: `service.BidService` interface.
- Produces: `service.NewBidService(auctions repository.AuctionRepository, bids repository.BidRepository, tx repository.TxManager, now func() time.Time) service.BidService`
- Produces: `(service.BidService).PlaceBid(ctx, userID, auctionID uuid.UUID, req web.PlaceBidRequest) (web.PlaceBidResponse, error)`

- [ ] **Step 1: Write failing bid service tests**

Tests:
- first bid at starting price succeeds using UUID auction and bidder IDs;
- bid below minimum returns `BID_TOO_LOW` with the minimum value;
- bidder already highest returns `ALREADY_HIGHEST`;
- creator bidding returns `CREATOR_CANNOT_BID`;
- bid after `ends_at` returns `AUCTION_NOT_LIVE`;
- bid inside the last 60 seconds extends `ends_at` and increments `extension_count`;
- sixth extension does not extend;
- replay with same key and same amount returns `Replay=true` and does not insert;
- same key with different amount returns `IDEMPOTENCY_CONFLICT`;
- two concurrent bids at the same amount: exactly one succeeds, one gets `BID_TOO_LOW`, and the database has one bid;
- version increments once per accepted bid.

- [ ] **Step 2: Implement `PlaceBid`**

Use one transaction:
1. `LockByID`;
2. validate auction, time, creator, highest bidder, amount, minimum;
3. check idempotency;
4. insert bid;
5. update auction fields and version;
6. commit.

After commit, return the updated auction and bid. Do not broadcast inside the service.

- [ ] **Step 3: Implement bid controller and route**

```text
POST /api/auctions/{id}/bids
```
Require auth. Read `amount` and `idempotency_key` from JSON. Parse `{id}` with `uuid.Parse`; invalid UUID returns 422 `VALIDATION_ERROR`. Return 201 on success and 200 on replay. Apply the bid rate limiter.

- [ ] **Step 4: Run concurrency tests**

```powershell
go test ./internal/service -run 'TestConcurrentBid|TestIdempotency|TestAntiSniping' -race -v
```
Expected: PASS. If the test exposes a race, fix the transaction, not the test.

- [ ] **Step 5: Commit**

```powershell
git add server
git commit -m "feat(server): add transactional bidding"
```

## Task 10: Realtime room manager and WebSocket

**Files:**
- Create: `server/internal/realtime/event.go`
- Create: `server/internal/realtime/client.go`
- Create: `server/internal/realtime/room.go`
- Create: `server/internal/realtime/manager.go`
- Create: `server/internal/controller/websocket_controller.go`
- Create: `server/internal/realtime/manager_test.go`
- Create: `server/internal/controller/websocket_controller_test.go`
- Modify: `server/internal/controller/bid_controller.go`
- Modify: `server/internal/controller/admin_controller.go`
- Modify: `server/internal/app/router.go`
- Modify: `server/cmd/api/main.go`

**Interfaces:**
- Produces: `realtime.Snapshot{Epoch string; Seq int64; Version int64; ServerTime time.Time; Auction domain.Auction; Bids []domain.BidView; ViewerCount int}`
- Produces: `realtime.Event{Type string; AuctionID uuid.UUID; Epoch string; Seq, Version int64; ServerTime time.Time; Payload any}`
- Produces: `realtime.Client{UserID *uuid.UUID; Send chan []byte}`
- Produces: `realtime.NewRoomManager() *RoomManager`
- Produces: `(*RoomManager).Join(auctionID uuid.UUID, userID *uuid.UUID, fallback Snapshot) (*Client, Snapshot)`
- Produces: `(*RoomManager).Leave(auctionID uuid.UUID, client *Client)`
- Produces: `(*RoomManager).PublishBid(auction domain.Auction, bid domain.BidView, previousHighestUserID *uuid.UUID, extended bool)`
- Produces: `(*RoomManager).PublishEnded(auction domain.Auction)`
- Produces: `controller.NewWebSocketController(manager *realtime.RoomManager, auctions service.AuctionService) *controller.WebSocketController`
- Produces: `(*controller.WebSocketController).ServeWS(w, r)`
- Modifies: `controller.NewBidController(service service.BidService, rooms *realtime.RoomManager) *controller.BidController`
- Modifies: `controller.NewAdminController(service service.AuctionService, rooms *realtime.RoomManager) *controller.AdminController`

- [ ] **Step 1: Add WebSocket dependency**

```powershell
go get github.com/coder/websocket@latest
```

- [ ] **Step 2: Write failing room manager tests**

Use `uuid.New()` for auction and user IDs. Tests:
- join returns a snapshot with the current version;
- publish bid increments seq once and all clients receive one event;
- event with an older version is ignored;
- outbid is sent only to the previous highest user UUID;
- presence count updates on join and leave;
- close code 4004 for an unknown or malformed auction UUID.

- [ ] **Step 3: Implement room manager**

Use a mutex per room for client maps and a single writer channel per client. Keep `lastVersion`; ignore `version <= lastVersion`. Assign `epoch` when the room is created. Use non-blocking sends and drop slow clients.

- [ ] **Step 4: Write failing WebSocket integration test**

Use `httptest.NewServer` and `coder/websocket`:
- connect two clients;
- send two bids through the service;
- both receive the same `seq` order and version;
- one client receives `outbid` when the other becomes highest;
- reconnect receives a fresh snapshot;
- no client sends commands through the socket.

- [ ] **Step 5: Implement WebSocket controller**

Parse `{id}` with `uuid.Parse`; malformed UUID gets close code 4004. Accept with origin check. Parse optional access token cookie and get the user UUID. Call `Join`, send the snapshot, start a writer goroutine, read frames only to detect close, and send pings every 20 seconds. Remove the client on exit. Wire the bid and admin controllers to call the room manager after a successful service commit.

- [ ] **Step 6: Verify and commit**

```powershell
go test ./internal/realtime ./internal/controller -race -v
git add server
git commit -m "feat(server): add realtime room and websocket"
```

## Task 11: Scheduler and restart recovery

**Files:**
- Create: `server/internal/app/scheduler.go`
- Create: `server/internal/app/scheduler_test.go`
- Modify: `server/cmd/api/main.go`
- Modify: `server/internal/service/auction_service.go`

**Interfaces:**
- Produces: `app.NewScheduler(auctions service.AuctionService, rooms *realtime.RoomManager, interval time.Duration, now func() time.Time) *app.Scheduler`
- Produces: `(*app.Scheduler).Run(ctx context.Context)`
- Produces: `(*app.Scheduler).CloseDue(ctx context.Context) error`

- [ ] **Step 1: Write failing scheduler tests**

Use `uuid.New()` for auction IDs. Tests:
- due live auction is ended and published;
- not-due auction is untouched;
- closing twice is idempotent;
- restart recovery loads a live auction and ends it if `ends_at` is in the past;
- admin end and scheduler end do not create two winners.

- [ ] **Step 2: Implement `CloseDue`**

Use `AuctionRepository.ListDue`, then for each auction run the same `End` transaction with row lock and idempotent status check. Publish `ended` once per auction that actually transitioned.

- [ ] **Step 3: Wire scheduler in main**

On startup, call `CloseDue` once before serving requests. Then start `Run(ctx)` with a 1 second interval. Stop it on graceful shutdown.

- [ ] **Step 4: Verify and commit**

```powershell
go test ./internal/app ./internal/service -race -v
git add server
git commit -m "feat(server): add lifecycle scheduler and restart recovery"
```

## Task 12: Observability, seed, load test, and runbook

**Files:**
- Create: `server/internal/middleware/logging.go`
- Create: `server/internal/app/metrics.go`
- Create: `server/internal/controller/health_controller.go` (extend DB ping)
- Create: `server/cmd/seed/main.go`
- Create: `server/loadtest/main.go`
- Modify: `server/internal/app/router.go`
- Modify: `server/cmd/api/main.go`
- Modify: `server/README.md`

**Interfaces:**
- Produces: `middleware.Logging(logger *slog.Logger, next http.Handler) http.Handler`
- Produces: `app.NewMetrics() *app.Metrics` with `Handler() http.Handler` and counters/histograms from the spec.
- Produces: `controller.NewHealthController(db interface{ Ping(context.Context) error }) *controller.HealthController`
- Produces: `seed` creates admin, bidder demo, one live auction, and one draft.

- [ ] **Step 1: Add Prometheus dependency**

```powershell
go get github.com/prometheus/client_golang@latest
```

- [ ] **Step 2: Write failing middleware and health tests**

Tests:
- `/metrics` returns 200 and contains `auction_bids_total`;
- `/healthz` returns 503 when DB ping fails and 200 when it succeeds;
- request log contains request ID and no email/cookie/token;
- seed is idempotent when run twice.

- [ ] **Step 3: Implement logging, metrics, health, and seed**

Use `slog.NewJSONHandler(os.Stdout, ...)`. Generate a request ID per request. Hash seed passwords with bcrypt. Insert seed rows without an ID and use `RETURNING id`; all IDs are UUID. Do not hardcode real credentials.

- [ ] **Step 4: Implement load test**

`loadtest/main.go` flags:
- `-base-url`
- `-auction-id` as a UUID string, parsed with `uuid.Parse`
- `-viewers`
- `-bidders`
- `-rate`
- `-duration`

It opens WebSocket connections, places bids through REST with auth cookies, and reports p50/p95/p99 broadcast latency, accepted bids, rejected bids, and errors. It must not run against production.

- [ ] **Step 5: Update runbook**

`server/README.md` must include:
- required env vars;
- docker compose Postgres;
- create test DB;
- run migrations;
- run API and seed;
- run tests with `TEST_DATABASE_URL`;
- run load test;
- note that refresh tokens are in httpOnly cookies and never localStorage.

- [ ] **Step 6: Final verification**

```powershell
go build ./...
go test ./... -race
go vet ./...
```

Record the test output and load test output. Do not claim performance numbers that were not measured.

- [ ] **Step 7: Commit**

```powershell
git add server
git commit -m "feat(server): add observability seed loadtest and runbook"
```

## Self-Review Checklist

- [ ] Spec section 4 schema is implemented by Task 2.
- [ ] Spec section 5 auth is implemented by Tasks 4, 5, 6, and 7.
- [ ] Spec section 6 REST is implemented by Tasks 7, 8, and 9.
- [ ] Spec section 7 WebSocket is implemented by Task 10.
- [ ] Spec section 8 bid flow is implemented by Task 9.
- [ ] Spec section 9 room manager and scheduler are implemented by Tasks 10 and 11.
- [ ] Spec section 10 errors are implemented by Task 7.
- [ ] Spec section 11 config is implemented by Task 1.
- [ ] Spec section 12 testing is covered by each task and Task 12.
- [ ] Spec section 13 observability is implemented by Task 12.
- [ ] No task relies on a function or type that is not defined in an earlier task.
- [ ] Every review focus item has a test in its owning task.
