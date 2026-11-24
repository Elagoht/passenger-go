# passenger-go collage Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite passenger-go as a single collage web application whose vault key is derived from the master passphrase, with sliding sessions that warn before expiry, and a `migrate` command that imports the old database.

**Architecture:** One Go binary built on `github.com/Elagoht/collage` v1.0.1 in the scaffold layout `collage new` produces (`main.go`, `routes.go`, `pages/`, `fragments/`, `actions/`, `templates/`, `static/`, embedded via `embed.go`). Domain logic lives in `internal/*` packages with no collage dependency; pages and actions are thin adapters. A random DEK is wrapped by Argon2id-derived keys (passphrase and recovery key); the unwrapped DEK lives only in the in-memory session.

**Tech Stack:** Go ≥ 1.26.11, collage v1.0.1 (`pkg/collage`, `pkg/collagetest`), `modernc.org/sqlite`, `golang.org/x/crypto/argon2`, `golang.org/x/crypto/hkdf`, `golang.org/x/net/publicsuffix`.

**Spec:** `docs/superpowers/specs/2026-11-24-collage-rewrite-design.md` — read it before any task; it is the source of every value below.

**collage reference:** clone `https://github.com/Elagoht/collage` at tag `v1.0.1`; `README.md`, `docs/actions.md`, `docs/http.md`, `docs/testing.md`, `docs/routing.md` (guards), `docs/fragments.md`. Scaffold a reference project with `collage new ref --template demo` to copy `embed.go` / `embed_dev.go` / `main_test.go` patterns.

## Global Constraints

- Module path stays `passenger-go`. `go` directive `1.26.11`.
- Never use the Go type `any` / `interface{}` in our code (user rule). Use concrete types or generics.
- No package except `internal/vault` imports `crypto/aes`, `crypto/cipher`, `argon2`, `hkdf`. `internal/legacy` is the one exception (it must read the old format) and keeps its own private decrypt.
- Never log or render: passphrases, DEK, recovery keys (except the one-time display pages), decrypted records.
- Argon2id: `t=3, m=65536 (KiB), p=4, keyLen=32`; salts 16 bytes.
- AAD strings exactly: `passenger/v1/dek/pass`, `passenger/v1/dek/rec`, `passenger/v1/account/<id>`; HKDF info `passenger/v1/index`.
- Session cookie `passenger_session`: `HttpOnly`, `SameSite=Strict`, `Path=/`, `Secure` iff `collage.IsHTTPS(r)`.
- Env vars: `PORT` (8080), `HOST` (collage default), `DATABASE_PATH` (`./data/passenger.db`), `CSRF_KEY` (required, ≥ 32 bytes), `SESSION_IDLE` (`15m`), `SESSION_MAX` (`12h`), `TRUSTED_PROXIES` (comma list, empty). collage's `COLLAGE_CSRF_KEY` is not used; we pass `CSRF_KEY` to `SecurityConfig.CSRFKey`.
- collage page cache disabled (`CacheConfig{Enabled: false}`); every response `Cache-Control: no-store`.
- UI copy is English. Failed form → same page, status 422, values kept (passwords not echoed back), error message. Success → 303.
- DB directory created `0700`, DB file `0600`.
- Use `crypto/rand` everywhere randomness is needed (the old generator used `math/rand` — do not port that).

## Review Focus

1. **Unicode and long input in account fields** (emoji platform names, 10 KB notes, Turkish `İ` in platform): lookup uniqueness must use `strings.ToLower(strings.TrimSpace(platform))` consistently on create, update and import — test in Task 3.
2. **Editing an account to collide with another** (rename platform/identifier to an existing pair) must give a 422 "already exists" on the form, not a 500 — test in Task 9.
3. **CSV import with BOM, CRLF and quoted commas/newlines** (Firefox exports quote every field; Windows Excel adds a BOM) — test in Task 10.
4. **Two tabs / two devices**: extending in one tab must not let the other tab redirect to login early; `GET /session` must not extend — test in Task 11.
5. **`next` parameter abuse** (`//evil.com`, `/\evil.com`, `https://evil.com`, `/%2F%2Fevil.com`) must fall back to `/` — test in Task 8.

---

## File Structure

```
main.go                      collage.Main(newApp); intercepts "migrate" before Main
app.go                       newApp(env collage.Env) (*collage.App, error): config, store, session, middleware, register
routes.go                    register(app, deps) — every page/action
embed.go / embed_dev.go      files fs.FS (templates, static) — copied from scaffold
main_test.go                 newTestApp helper + end-to-end collagetest tests (split into *_test.go per area)
internal/config/config.go    Config + FromEnv
internal/vault/              kdf.go, wrap.go, record.go, recovery.go — all crypto
internal/store/              store.go (open/migrate schema), vault_row.go, accounts.go
internal/strength/           strength.go
internal/generator/          generator.go
internal/transfer/           importer.go, exporter.go, platform.go
internal/session/            manager.go, middleware.go
internal/ratelimit/          limiter.go
internal/web/                headers.go (security headers middleware), guards.go, deps.go (Deps struct), forms.go (shared form helpers, next-path sanitizer)
internal/legacy/             migrate.go, olddecrypt.go, cli.go
fragments/layouts/           app.go (guarded by RequireSession), auth.go (no guard), base.go
fragments/pages/...          one file per page content
pages/...                    one file per page
actions/...                  actions with handlers
templates/layouts/*.html, templates/pages/**/*.html
static/app.css, static/session.js, static/accounts.js, static/favicon.svg
```

Old `backend/`, `frontend/`, `cli/` are deleted in Task 1.

---

### Task 1: Skeleton and configuration

**Files:**
- Delete: `backend/`, `frontend/`, `cli/`, `go.sum`
- Create: `main.go`, `app.go`, `routes.go`, `embed.go`, `embed_dev.go`, `internal/config/config.go`, `internal/config/config_test.go`, `templates/layouts/base.html`, `static/favicon.svg` (reuse `frontend/static/img/passenger.png` → `static/passenger.png` before deleting), `.env.example`, `.gitignore` (add `data/`, `.cache/`, `bin/`)
- Modify: `go.mod`

**Interfaces:**
- Produces:
  - `config.Config{DatabasePath string; CSRFKey []byte; SessionIdle, SessionMax time.Duration; TrustedProxies []string}`
  - `config.FromEnv(getenv func(string) string) (config.Config, error)`
  - `newApp(env collage.Env) (*collage.App, error)` in `app.go` (Tasks 7+ extend it)

- [ ] **Step 1: Write failing tests** in `internal/config/config_test.go`:
  - `TestFromEnvRequiresCSRFKey`: empty `CSRF_KEY` → error mentioning `CSRF_KEY`.
  - `TestFromEnvRejectsShortCSRFKey`: 31-byte key → error.
  - `TestFromEnvDefaults`: only `CSRF_KEY` set → `DatabasePath == "./data/passenger.db"`, `SessionIdle == 15*time.Minute`, `SessionMax == 12*time.Hour`, `TrustedProxies == nil`.
  - `TestFromEnvParsesDurationsAndProxies`: `SESSION_IDLE=5m`, `SESSION_MAX=1h`, `TRUSTED_PROXIES="10.0.0.1, 192.168.0.0/16"` → parsed, trimmed.
  - `TestFromEnvRejectsBadDuration`: `SESSION_IDLE=abc` → error; `SESSION_IDLE=0s` → error.
- [ ] **Step 2:** `go test ./internal/config/` → FAIL (package missing).
- [ ] **Step 3:** Implement `FromEnv`. Remove old code, `go get github.com/Elagoht/collage@v1.0.1 modernc.org/sqlite golang.org/x/crypto golang.org/x/net`, write `main.go`/`app.go` modelled on the scaffold's `main.go` (Template FS `files`, Root `templates`, Extension `.html`; `Cache.Enabled=false`; `Security.CSRFKey=cfg.CSRFKey`; `Server.TrustedProxies`; mount `/static/` with `env.Static(files, "static")`). `newApp` returns the config error. `routes.go` registers nothing yet except a temporary `home` page rendering `base.html` — Task 8 replaces it.
- [ ] **Step 4:** `go test ./... && go vet ./... && go build ./...` → PASS.
- [ ] **Step 5:** Commit `chore: replace chi app with collage skeleton and env config`.

### Task 2: Vault cryptography

**Files:** Create `internal/vault/{kdf.go,wrap.go,record.go,recovery.go,vault_test.go}`

**Interfaces:**
- Produces:
  - `type KDFParams struct{ Alg string; T uint32; M uint32; P uint8 }` with JSON tags `alg,t,m,p`; `var DefaultKDF = KDFParams{"argon2id",3,65536,4}`
  - `type DEK [32]byte`; `func NewDEK() (DEK, error)`; `func (d *DEK) Wipe()`
  - `type Wrapped struct{ Salt []byte; Box []byte }` (Box = nonce‖ciphertext)
  - `func Wrap(secret string, dek DEK, params KDFParams, aad string) (Wrapped, error)` — new random salt each call
  - `func Unwrap(secret string, w Wrapped, params KDFParams, aad string) (DEK, error)` — returns `ErrWrongSecret` on GCM failure
  - `const AADPass = "passenger/v1/dek/pass"`, `AADRecovery = "passenger/v1/dek/rec"`
  - `type Record struct{ Platform, Identifier, Passphrase, URL, Notes string }` (JSON tags lower-case)
  - `func Seal(dek DEK, id string, r Record) ([]byte, error)`; `func Open(dek DEK, id string, blob []byte) (Record, error)` — AAD `passenger/v1/account/`+id
  - `func Lookup(dek DEK, platform, identifier string) []byte` — HMAC-SHA256 under HKDF(dek, info `passenger/v1/index`), input `strings.ToLower(strings.TrimSpace(platform)) + "\x00" + identifier`
  - `func NewRecoveryKey() (string, error)` — 32 random bytes, base32 std no padding, groups of 4 joined with `-`
  - `func NormalizeRecoveryKey(s string) string` — upper-case, strip spaces and `-`
  - `func NewAccountID() (string, error)` — 16 random bytes hex
  - `var ErrWrongSecret = errors.New("wrong passphrase or recovery key")`
- Callers pass `NormalizeRecoveryKey(k)` as the secret for recovery wraps.

- [ ] **Step 1: Write failing tests** (use `KDFParams{"argon2id",1,1024,1}` in tests for speed):
  - wrap→unwrap round trip returns the same DEK; two wraps of the same DEK differ (salt/nonce).
  - wrong secret → `errors.Is(err, ErrWrongSecret)`; correct secret but wrong AAD → `ErrWrongSecret`.
  - `Seal`/`Open` round trip; `Open` with a different id → error (AAD binding); flipping one blob byte → error.
  - `Lookup("GitHub ","a")` equals `Lookup("github","a")`; differs from `Lookup("github","A")`; differs across DEKs; `Lookup("İnstagram","x")` is stable across calls.
  - recovery key: 52 base32 chars before grouping, `NormalizeRecoveryKey("abcd-efgh ijkl") == "ABCDEFGHIJKL"`; wrap with normalized key unwraps with a lower-case spaced variant after normalizing.
- [ ] **Step 2:** `go test ./internal/vault/` → FAIL.
- [ ] **Step 3:** Implement. `Wipe` zeroes the array.
- [ ] **Step 4:** `go test ./internal/vault/` → PASS.
- [ ] **Step 5:** Commit `feat(vault): passphrase-derived key wrapping and record sealing`.

### Task 3: SQLite store

**Files:** Create `internal/store/{store.go,vault_row.go,accounts.go,store_test.go}`

**Interfaces:**
- Consumes: Task 2 types.
- Produces:
  - `func Open(path string) (*Store, error)` — creates dir 0700, file 0600, `PRAGMA foreign_keys=ON; journal_mode=WAL; busy_timeout=5000`, applies schema from spec when `user_version == 0`, sets `user_version = 1`; refuses `user_version > 1`.
  - `func (s *Store) Close() error`
  - `type VaultRow struct{ KDF vault.KDFParams; Pass, Recovery vault.Wrapped; CreatedAt time.Time }`
  - `func (s *Store) Vault(ctx) (VaultRow, error)` — `ErrNotInitialized` when absent
  - `func (s *Store) Initialized(ctx) (bool, error)`
  - `func (s *Store) CreateVault(ctx, VaultRow) error` — `ErrAlreadyInitialized` if a row exists
  - `func (s *Store) UpdatePassWrap(ctx, vault.Wrapped) error`; `func (s *Store) UpdateRecoveryWrap(ctx, vault.Wrapped) error`
  - `type Account struct{ ID string; vault.Record; CreatedAt, UpdatedAt time.Time }`
  - `func (s *Store) ListAccounts(ctx, dek vault.DEK) ([]Account, error)` — sorted by `strings.ToLower(Platform)`, then Identifier
  - `func (s *Store) GetAccount(ctx, dek, id string) (Account, error)` — `ErrNotFound`
  - `func (s *Store) CreateAccount(ctx, dek, vault.Record) (Account, error)` — `ErrDuplicate` on lookup conflict
  - `func (s *Store) UpdateAccount(ctx, dek, id string, vault.Record) error` — `ErrNotFound`, `ErrDuplicate`
  - `func (s *Store) DeleteAccount(ctx, id string) error` — `ErrNotFound`
  - `func (s *Store) InsertAccountsTx(ctx, dek, []vault.Record) error` — all-or-nothing (used by migration)
  - Errors: `ErrNotInitialized, ErrAlreadyInitialized, ErrNotFound, ErrDuplicate`
  - Store has a `now func() time.Time` field, settable via `func (s *Store) SetClock(func() time.Time)` for tests.

- [ ] **Step 1: Write failing tests** (temp dir DB):
  - `Open` creates the file with mode `0600` and its dir `0700`; reopening keeps data; `user_version` is 1.
  - `Initialized` false → `CreateVault` → true; second `CreateVault` → `ErrAlreadyInitialized`.
  - create/get/list/update/delete account round trip; list order case-insensitive by platform.
  - duplicate (`"GitHub"`,`"me"`) vs (`" github"`,`"me"`) → `ErrDuplicate` on create and on update of another row.
  - `GetAccount` with a different DEK → error, not garbage.
  - `InsertAccountsTx` with a duplicate pair in the batch → error and zero rows written.
  - no plaintext on disk: after creating an account with passphrase `"hunter2-unique-marker"`, reading the DB file bytes does not contain that string.
- [ ] **Step 2:** `go test ./internal/store/` → FAIL.
- [ ] **Step 3:** Implement with `database/sql` + `modernc.org/sqlite`; map SQLite UNIQUE violation to `ErrDuplicate`.
- [ ] **Step 4:** `go test ./internal/store/` → PASS.
- [ ] **Step 5:** Commit `feat(store): sqlite vault and encrypted account storage`.

### Task 4: Strength and generator

**Files:** Create `internal/strength/strength.go(+_test)`, `internal/generator/generator.go(+_test)`

**Interfaces:**
- Produces:
  - `func strength.Score(passphrase string) int` — same scoring as old `backend/utilities/strength/strength.go` (length buckets −2/0/1/2/3/4 + 1 per present set: lower, upper, digit, symbol). Length counts runes, not bytes.
  - `func strength.Label(score int) string` — `≤2 "weak"`, `3–5 "fair"`, `≥6 "strong"`
  - `func generator.Generate(length int) (string, error)` — clamp to [8, 4096]; old algorithm (random chars from `Chars`, then 8 distinct positions forced to two of each set), using `crypto/rand`
  - `func generator.Alternate(s string) (string, error)` — old `ManipulateMap` substitution, `crypto/rand`
  - Copy `ManipulateMap`, `Specials`, `Lowers`, `Uppers`, `Numbers`, `Chars` from old `backend/utilities/generator/generator.go` (keep their exact contents).

- [ ] **Step 1: Write failing tests:** `Score("")==-2+0`, `Score("abc")==-1`, `Score("Abcdefgh1!")==5`, `Score(21 runes of "ş")==4+1`; `Generate(3)` has length 8, `Generate(10000)` length 4096, every `Generate(16)` result contains ≥2 of each of the four sets (loop 200 times); `Alternate("password")` has the same rune count and every rune is in that letter's `ManipulateMap` list.
- [ ] **Step 2:** run → FAIL. **Step 3:** implement. **Step 4:** run → PASS.
- [ ] **Step 5:** Commit `feat: strength scoring and crypto-random generator`.

### Task 5: Sessions

**Files:** Create `internal/session/{manager.go,middleware.go,session_test.go}`

**Interfaces:**
- Consumes: `vault.DEK`.
- Produces:
  - `type Session struct{ Token string; DEK vault.DEK; Created, LastSeen time.Time }` (only read through the manager)
  - `func NewManager(idle, max time.Duration, now func() time.Time) *Manager`
  - `func (m *Manager) Create(dek vault.DEK) (*Session, error)`
  - `func (m *Manager) Touch(token string) (*Session, bool)` — valid → updates LastSeen, returns copy; expired → deletes, wipes DEK, false
  - `func (m *Manager) Peek(token string) (*Session, bool)` — like Touch without updating
  - `func (m *Manager) ExpiresAt(s *Session) time.Time` — `min(LastSeen+idle, Created+max)`
  - `func (m *Manager) Delete(token string)`
  - `func (m *Manager) Rotate(token string) (*Session, error)` — new token, same DEK, deletes every other session
  - `func (m *Manager) DeleteAll()`
  - `func (m *Manager) Sweep()`; `func (m *Manager) Start(ctx context.Context)` (minute ticker calling Sweep until ctx done)
  - `const CookieName = "passenger_session"`
  - `func (m *Manager) SetCookie(w http.ResponseWriter, r *http.Request, s *Session)`; `func ClearCookie(w http.ResponseWriter, r *http.Request)`
  - `func (m *Manager) Middleware(next http.Handler) http.Handler` — reads the cookie; for paths other than `/session` calls `Touch`, for `/session` calls `Peek`; stores result with `WithSession(ctx, *Session)`; records `HadCookie(ctx) bool` (cookie present but invalid/expired) for the guard
  - `func FromContext(ctx context.Context) (*Session, bool)`; `func HadCookie(ctx context.Context) bool`

- [ ] **Step 1: Write failing tests** with a fake clock:
  - Create → Touch at +14m ok; Touch at +14m then +28m ok (sliding); idle 15m exceeded → false and session gone.
  - Absolute: touching every 10m, at +12h01m → false.
  - `Peek` does not extend: Peek at +10m, then Touch at +16m → false.
  - `ExpiresAt` returns min of idle and absolute.
  - `Rotate` returns a new token; old token invalid; other sessions invalid; DEK equal.
  - Expired session's DEK is zeroed (keep a pointer to the internal session in the test via `Peek` before expiry, assert all-zero after).
  - Middleware: request without cookie → `FromContext` false, `HadCookie` false; with stale cookie → false, true.
  - `SetCookie` header contains `HttpOnly`, `SameSite=Strict`, `Path=/`, and `Secure` only when the request has TLS.
- [ ] **Step 2:** FAIL. **Step 3:** implement (one `sync.Mutex`, map). **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(session): in-memory sliding sessions holding the DEK`.

### Task 6: Attempt limiter

**Files:** Create `internal/ratelimit/{limiter.go,limiter_test.go}`

**Interfaces:**
- Produces:
  - `func New(now func() time.Time) *Limiter`
  - `func (l *Limiter) Allow(ip netip.Addr) (wait time.Duration, ok bool)` — false while that IP is in backoff
  - `func (l *Limiter) Fail(ip netip.Addr)` — after the 5th consecutive failure, backoff = 1s·2^(n−5), capped at 5m
  - `func (l *Limiter) Succeed(ip netip.Addr)` — resets that IP
  - `func (l *Limiter) GlobalDelay() time.Duration` — 2s when >100 failures across all IPs in the trailing hour, else 0
  - Entries older than 24h are pruned on access.

- [ ] **Step 1: Tests:** 4 failures → still allowed; 5th → `Allow` false with wait 1s; after 1s allowed; 6th failure → 2s; cap 5m after many; other IP unaffected; `Succeed` resets; 101 failures from distinct IPs within an hour → `GlobalDelay()==2s`, after an hour 0.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat(ratelimit): per-IP exponential backoff for secret attempts`.

### Task 7: Web plumbing — deps, guards, headers, layouts, styles

**Files:** Create `internal/web/{deps.go,guards.go,headers.go,forms.go,web_test.go}`, `fragments/layouts/{app.go,auth.go}`, `templates/layouts/{app.html,auth.html}`, `static/app.css` (port `frontend/static/css/{main,app,auth}.css`, drop unused rules), `static/session.js` (stub, filled in Task 11), test helper `testapp_test.go` in package `main`.

**Interfaces:**
- Consumes: Tasks 1–6.
- Produces:
  - `type web.Deps struct{ Store *store.Store; Sessions *session.Manager; Limiter *ratelimit.Limiter; KDF vault.KDFParams; Now func() time.Time; SetupCode *SetupCode; Pending *PendingSetup }` (the last two filled in Task 8; declare the types there)
  - `func (d *web.Deps) RequireSession(ctx context.Context, r *http.Request) (*collage.GuardDecision, error)` — no vault → 303 `/setup`; no session → 303 `/login?next=<escaped path+query>` plus `&expired=1` when `session.HadCookie`
  - `func (d *web.Deps) RequireGuest(...)` — vault absent → 303 `/setup`; session present → 303 `/`
  - `func (d *web.Deps) RequireUninitialized(...)` — vault present → 303 `/login`
  - `func web.SafeNext(raw string) string` — returns raw only if it starts with `/`, not `//`, not `/\`, contains no `\`, and after `url.PathUnescape` still satisfies the same; else `/`
  - `func web.SecurityHeaders(next http.Handler) http.Handler` — sets the spec's CSP exactly, `Referrer-Policy: no-referrer`, `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`
  - `func web.SessionFrom(rc *collage.RenderContext) *session.Session` — panics never; returns nil when absent
  - Layout fragments: `layouts.App(d *web.Deps) *collage.Fragment` (guarded with `RequireSession`, data: `{ExpiresAtMS int64}` for `data-session-expires` on `<body>`, nav: Accounts, New, Generate, Import, Export, Settings, Logout form), `layouts.Auth() *collage.Fragment` (centered card, no guard).
  - Test helper (package main): `func newTestApp(t *testing.T) (*collagetest.Client, *testEnv)` setting env `CSRF_KEY` (32×"k"), `DATABASE_PATH` (temp), fast KDF (`KDFParams{"argon2id",1,1024,1}` via an unexported `kdfOverride` hook in `app.go`), and a controllable clock; `testEnv` exposes `Clock *fakeClock`, `Deps *web.Deps`.
- `newApp` wires: `config.FromEnv(os.Getenv)` → `store.Open` → `session.NewManager` (+`Start` with a context cancelled on app shutdown — register via a tiny plugin with `Shutdown` or `app` shutdown hook; also `store.Close` there) → `app.Use(web.SecurityHeaders)`, `app.Use(deps.Sessions.Middleware)`.

- [ ] **Step 1: Failing tests** (`web_test.go`): `SafeNext` table — `"/accounts/1?x=1"` kept; `"//evil.com"`, `"/\\evil.com"`, `"https://evil.com"`, `"/%2F%2Fevil.com"`, `""`, `"accounts"` → `"/"`. Guard tests with an in-memory store: uninitialized → `/setup`; initialized no cookie → `/login?next=%2Faccounts%2F1`; stale cookie → contains `expired=1`. Headers test: CSP string equals the spec's.
- [ ] **Step 2–4:** FAIL → implement → PASS (`go test ./...`).
- [ ] **Step 5:** Commit `feat(web): guards, security headers, layouts and styles`.

### Task 8: Setup, login, logout, recover

**Files:** Create `internal/web/setup.go` (`SetupCode`, `PendingSetup`), `pages/auth/{setup.go,setup_confirm.go,login.go,recover.go}`, `fragments/pages/auth/*.go`, `templates/pages/auth/{setup,setup_confirm,login,recover,recovery_shown}.html`, `actions/auth/{setup.go,confirm.go,login.go,logout.go,recover.go}`, `auth_test.go` (package main). Modify `routes.go`, `app.go`.

**Interfaces:**
- Produces:
  - `type SetupCode struct{...}`; `func NewSetupCode() (*SetupCode, error)` — 8 digits from `crypto/rand`; `func (c *SetupCode) Check(s string) bool` (constant-time); `func (c *SetupCode) String() string`. `newApp` logs `"setup code: 12345678"` at startup only when not initialized (via the app's logger / `log.Printf`).
  - `type PendingSetup struct{...}`; `func (p *PendingSetup) Put(dek vault.DEK, row store.VaultRow, recovery string) (token string, err error)` (replaces any previous), `Get(token) (row, recovery, dek, ok)`, `Clear()`; cookie `passenger_setup` (same flags as session cookie), 15-minute lifetime.
  - Flows exactly as spec table: `/setup` (fields `code`, `passphrase`, `confirm`; limiter on `code` and on the confirm step's `recovery`; passphrase ≥ 12 runes; mismatch → 422) → render `setup_confirm` showing the recovery key (grouped) with field `recovery` → `POST /setup/confirm` compares `NormalizeRecoveryKey` → `CreateVault`, `Sessions.Create`, set cookie, clear pending, 303 `/`. Wrong key → 422 with the key shown again.
  - `/login` (field `passphrase`, hidden `next`): limiter check (429 page with "Try again in N seconds") → `Unwrap` → `Succeed`, session, 303 `SafeNext(next)`; wrong → `Fail`, 422 "Wrong passphrase". Apply `GlobalDelay` via `time.Sleep` before verifying. `?expired=1` shows "Your session expired. Sign in again."
  - `POST /logout` → `Sessions.Delete`, clear cookie, 303 `/login`.
  - `/recover` (fields `recovery`, `passphrase`, `confirm`): limiter; `Unwrap` recovery wrap with normalized key → new pass wrap, new recovery key + wrap, `UpdatePassWrap`+`UpdateRecoveryWrap` (one transaction: add `func (s *Store) UpdateWraps(ctx, pass, rec vault.Wrapped) error` to Task 3's store here), `DeleteAll` sessions, render `recovery_shown` with the new key and a link to `/login`.

- [ ] **Step 1: Failing tests** (`auth_test.go`, collagetest):
  - fresh app: `GET /` → 303 `/setup`; `GET /login` → 303 `/setup`.
  - setup with wrong code → 422; with short passphrase → 422; mismatch → 422; correct → page contains a recovery key; confirm with wrong key → 422; confirm with lower-case, spaced key → 303 `/`; `GET /` → 200; `GET /setup` → 303 `/login`.
  - setup without a CSRF token → 403.
  - setup with a wrong code 5 times → 6th attempt (even with the correct code) → 429.
  - login wrong → 422 "Wrong passphrase"; 5 wrong → 6th attempt (even correct) → 429; advance fake clock 1s → correct succeeds.
  - login with `next=/accounts/new` → 303 `/accounts/new`; with `next=//evil.com` → 303 `/`.
  - expired: login, advance clock 16m, `GET /` → 303 location contains `expired=1` and `next=%2F`; login page shows "Your session expired".
  - logout → `GET /` → 303 `/login?next=%2F` (no `expired`).
  - recover with the recovery key → new key shown, old key no longer recovers (422), new passphrase logs in, old passphrase does not, previously logged-in client is logged out.
  - session cookie flags: `Set-Cookie` contains `HttpOnly` and `SameSite=Strict`.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat: setup with one-time code, login, logout and recovery`.

### Task 9: Accounts, search, generator page

**Files:** `pages/accounts/{list.go,new.go,detail.go,delete.go}`, `pages/generate.go`, matching `fragments/pages/...`, `templates/pages/accounts/*.html`, `templates/pages/generate.html`, `actions/accounts/{create.go,update.go,delete.go,passphrase.go}`, `actions/generate.go`, `static/accounts.js`, `accounts_test.go`.

**Interfaces:**
- Consumes: `store` account methods, `strength.Score/Label`, `generator.Generate/Alternate`, `web.SessionFrom`.
- Produces routes: `GET /` (list; `?q=` filters case-insensitively on platform, identifier, url, notes), `GET/POST /accounts/new`, `GET/POST /accounts/{id}`, `GET/POST /accounts/{id}/delete`, `POST /accounts/{id}/passphrase` → `collage.JSON(200, passphraseResponse{...})` where the field is `Passphrase string` tagged `json:"passphrase"` (requires `X-CSRF-Token`), `GET/POST /generate` (fields `length`, or `source` for alternate).
- List card: initials badge (first letter of platform, upper-case; `?` if empty), platform, identifier, strength bar class `strength-weak|fair|strong`, copy-identifier and copy-passphrase buttons (`data-id`), external link only when URL parses with scheme `http`/`https` (else no link; prefix `https://` when scheme missing, as old `sanitizeUrl`).
- Form fields: `platform` (required), `identifier` (required), `passphrase` (required), `url`, `notes`. Detail page shows passphrase in `<input type="password">` with reveal/copy buttons; `accounts.js` handles reveal/copy/live filter using `textContent` only; the new-account form has a "Generate" button calling the generate action via fetch (`X-CSRF-Token`) or a plain link to `/generate`.

- [ ] **Step 1: Failing tests:**
  - create → 303 to `/accounts/{id}`; detail shows platform; list shows it; strength class present.
  - missing platform → 422 with the other values kept.
  - duplicate create → 422 "An account with this platform and identifier already exists".
  - edit to collide with another account → 422 same message (Review Focus 2).
  - unknown id → 404.
  - delete: GET shows confirm; POST → 303 `/`; gone from list.
  - `?q=git` matches "GitHub", not "Gmail"; `?q=` with `<script>` is rendered escaped.
  - XSS: platform `<img src=x onerror=alert(1)>` appears only escaped (`&lt;img`) in list and detail.
  - passphrase JSON: with header token → 200 body `{"passphrase":"<value>"}`; without token → 403; without session → 303.
  - generate page: `length=20` → result 20 runes; alternate `source=password` → 8 runes.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat: account management, search and generator`.

### Task 10: Import, export, settings

**Files:** `internal/transfer/{platform.go,importer.go,exporter.go,transfer_test.go}`, `pages/{import.go,export.go}`, `pages/settings/{passphrase.go,recovery.go}`, fragments/templates/actions for them, `transfer_web_test.go`.

**Interfaces:**
- Produces:
  - `func transfer.PlatformFromURL(raw string) string` — port of old `ConvertURLToPlatformName` (eTLD+1 first label, capitalized; `"Unknown"` on failure)
  - `func transfer.Import(r io.Reader) (format string, records []vault.Record, err error)` — strips UTF-8 BOM; detects by exact header: Firefox (`url,username,password,httpRealm,formActionOrigin,guid,timeCreated,timeLastUsed,timePasswordChanged`; platform = `PlatformFromURL(url)`), Chromium (`name,url,username,password,note`), passenger (`platform,identifier,passphrase,url,notes`); `encoding/csv` with `FieldsPerRecord=-1`, `LazyQuotes=true`; rows with fewer fields than the header are skipped; `ErrUnknownFormat`.
  - `func transfer.Export(w io.Writer, accounts []store.Account) error` — `encoding/csv`, header `platform,identifier,passphrase,url,notes`.
  - `/import` (multipart field `file`, `WithMaxBodyBytes(10<<20)`): each record via `CreateAccount`; result page lists success count and failed rows (platform + identifier + reason: duplicate / missing field). Never echo failed passphrases.
  - `/export`: field `passphrase`; limiter; `Unwrap` to verify → `ActionResult{Body: csv, ContentType: "text/csv; charset=utf-8", Header: Content-Disposition attachment; filename="passenger-export-YYYY-MM-DD.csv"}`.
  - `/settings/passphrase`: `current`, `passphrase`, `confirm` (≥12 runes); limiter on `current`; new pass wrap → `UpdatePassWrap`; `Sessions.Rotate` + set cookie; 303 `/settings/passphrase?saved=1`.
  - `/settings/recovery`: `current`; limiter; new recovery key + wrap → `UpdateRecoveryWrap`; render key once.

- [ ] **Step 1: Failing tests:**
  - transfer unit: Firefox CRLF fully quoted sample with BOM; Chromium LF sample; passenger sample with a comma, a quote and a newline inside notes (Review Focus 3); unknown header → `ErrUnknownFormat`; `PlatformFromURL("https://accounts.google.com/x")=="Google"`, `("not a url")=="Unknown"`.
  - export→import round trip yields identical records.
  - web: import a 3-row Chromium file with one duplicate → page says 2 imported, 1 failed (duplicate); export with wrong passphrase → 422; correct → `text/csv` body containing the header and the account; change passphrase → old passphrase fails login, new works, a second logged-in client is logged out; regenerate recovery → old key fails on `/recover`, new key works.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat: csv import/export and passphrase/recovery settings`.

### Task 11: Session expiry UX

**Files:** `actions/session/{status.go,extend.go}`, `static/session.js`, `templates/layouts/app.html` (banner markup), `session_web_test.go`.

**Interfaces:**
- Consumes: `session.Manager.Peek/Touch/ExpiresAt` and the Task 5 middleware's `/session` special case.
- Produces:
  - `GET /session` (action, `WithMethods(GET, HEAD)`, no page guard) → `200 {"expiresAt": <unix ms>}` or `401 {"expiresAt":0}`; does not extend.
  - `POST /session/extend` (CSRF via header) → Touch already happened in middleware → `200 {"expiresAt": ...}`; no session → 401.
  - `session.js`: reads `document.body.dataset.sessionExpires`; schedules a check at `expires − 60s`: fetch `/session`; if remaining > 60s reschedule; else show `#session-banner` (hidden by default in layout) with text "Your session ends in N seconds." and buttons Extend (POST `/session/extend` with `X-CSRF-Token` from the layout's hidden `{{csrfToken}}` input `_csrf`) / Sign out (submits logout form). Countdown updates each second. If any `form[data-track]` field changed, banner adds "You have unsaved changes." At expiry: `location.assign('/login?expired=1&next=' + encodeURIComponent(location.pathname + location.search))`. On `visibilitychange` to visible, re-check immediately (laptop sleep).
- [ ] **Step 1: Failing tests:** `GET /session` returns expiresAt = login time + 15m (fake clock); after advancing 10m and calling `GET /session`, expiresAt unchanged (Review Focus 4); `POST /session/extend` at +10m → expiresAt = +25m; with no token → 403; after expiry → 401; layout `<body>` has `data-session-expires`.
- [ ] **Step 2–4:** FAIL → implement → PASS. Then manually: `CSRF_KEY=... SESSION_IDLE=2m go run .`, log in, wait, see the banner at ~1 min, click Extend, see it vanish; let it expire, land on login with the message.
- [ ] **Step 5:** Commit `feat: session expiry warning, extend and redirect`.

### Task 12: Legacy migration

**Files:** `internal/legacy/{olddecrypt.go,migrate.go,cli.go,legacy_test.go}`, modify `main.go`.

**Interfaces:**
- Consumes: `store.Open`, `store.CreateVault`, `store.InsertAccountsTx`, `vault.*`.
- Produces:
  - `type Options struct{ From, EnvFile, To string; KDF vault.KDFParams; Out io.Writer }`
  - `type Result struct{ Accounts int; RecoveryKey string }`
  - `func Migrate(ctx context.Context, o Options) (Result, error)` — steps 1–8 of the spec's migration section. Old DB opened `file:<path>?mode=ro`. `oldDecrypt(secret []byte, b64 string) (string, error)` = base64-std decode, AES-GCM, 12-byte nonce prefix. Reads `AES_GCM_SECRET` by parsing the env file line by line (`KEY=VALUE`, optional quotes, ignores comments) — do not `os.Setenv`. Errors: target exists; old user not validated; any undecryptable row (message names old `accounts.id`); lookup collision (names both old ids). On any error after the target was created, remove the target and its `-wal`/`-shm`.
  - `func RunCLI(args []string, stdout, stderr io.Writer) int` — flags `--from`, `--env`, `--to` (default `./data/passenger.db`); prints `"Migrated N accounts."`, `"Your master passphrase is unchanged."`, the new recovery key on its own line, `"Your old recovery key no longer works. Store the new one safely."`, and `"Now delete the old database and the old .env (it holds AES_GCM_SECRET): <from>, <env>"`.
  - `main.go`: `if len(os.Args) > 1 && os.Args[1] == "migrate" { os.Exit(legacy.RunCLI(os.Args[2:], os.Stdout, os.Stderr)) }` before `collage.Main` (keeps migration independent of `CSRF_KEY` and of opening the new DB).
- Test fixture: `legacy_test.go` builds an old-format DB in a temp dir using the old schema (copy `QueryCreateUserTable`/`QueryCreateAccountsTable` from the deleted `backend/utilities/database/database.queries.go`; retrieve via `git show main:backend/utilities/database/database.queries.go`) and a test-only `oldEncrypt` (random nonce) and `oldEncryptDeterministic` (nonce = sha256(data‖secret)[:12]) copied from `git show main:backend/utilities/encrypt/encrypt.go`.

- [ ] **Step 1: Failing tests:**
  - 3 accounts (one with NULL `url`/`notes`, one with Turkish characters) → `Migrate` ok; `Result.Accounts==3`; open target, unwrap with old master passphrase, records equal; unwrap with `Result.RecoveryKey` works; old DB file bytes unchanged (compare sha256 before/after).
  - target exists → error, target untouched.
  - user `validated=0` → error, target absent.
  - one corrupted ciphertext → error naming its id, target absent.
  - wrong `AES_GCM_SECRET` → error, target absent.
  - `RunCLI` with missing `--from` → exit 2 with usage on stderr.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Manual check against a real old DB: build the old version from `main` (`git worktree add /tmp/old main && cd /tmp/old && go build -o old ./cli`), register + add two accounts through its UI, then `go run . migrate --from /tmp/old/database/passenger.db --env /tmp/old/.env --to /tmp/new.db`, then start the new app with `DATABASE_PATH=/tmp/new.db` and log in with the same passphrase.
- [ ] **Step 6:** Commit `feat: migrate command for the old database`.

### Task 13: Deployment files and docs

**Files:** Modify `install.sh`, `passenger-go.service`, `.env.example`, `README.md`, `.github/workflows/release.yaml`.

- `install.sh`: no template/static copy (embedded); writes `/opt/passenger-go/.env` with `>` containing `PORT`, `DATABASE_PATH=/opt/passenger-go/data/passenger.db`, `CSRF_KEY=$(openssl rand -hex 32)`; if `/opt/passenger-go/database/passenger.db` exists, print the migrate command (`/opt/passenger-go/passenger-go migrate --from /opt/passenger-go/database/passenger.db --env /opt/passenger-go/.env.old --to /opt/passenger-go/data/passenger.db`, after moving the old `.env` to `.env.old`) and the delete instruction, and exit before enabling the service. After a fresh install print where to find the setup code: `journalctl -u passenger-go | grep "setup code"`.
- `passenger-go.service`: keep; add `EnvironmentFile=/opt/passenger-go/.env`, `NoNewPrivileges=true`, `ProtectSystem=strict`, `ReadWritePaths=/opt/passenger-go/data`, `PrivateTmp=true`.
- `release.yaml`: build `go build -trimpath -o passenger-go .` (root package now), Go version ≥ 1.26.11.
- `README.md`: features, security model (one paragraph from spec), install, upgrade/migration, env table.
- [ ] **Step 1:** `bash -n install.sh` → no output. `go build ./... && go vet ./... && go test ./...` → all PASS.
- [ ] **Step 2:** Commit `docs: deployment, migration and security model`.

---

## Self-review notes

- Spec coverage: crypto (T2), schema (T3), sessions + expiry UX (T5, T11), pages table (T8–T10), security headers/CSRF/XSS (T7, T9), brute force incl. setup (T6, T8), setup code (T8), migration incl. new recovery key and warnings (T12), config (T1), deployment (T13), tests throughout.
- Deviation from spec, deliberate: `migrate` is intercepted in `main.go` instead of `app.RegisterCommand`, because `collage.Main` builds the app (needs `CSRF_KEY`, opens/creates the new DB) before dispatching commands, which would create the very target file migration must refuse to overwrite.
