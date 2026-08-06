# Go conventions

Go 1.26. Stdlib-first: routing, middleware, and handlers are plain `net/http` —
no web framework.

## Libraries

- `spf13/cobra` + `spf13/viper` — CLI and config
- `jmoiron/sqlx` + `modernc.org/sqlite` — database (pure Go, `CGO_ENABLED=0`
  static builds)
- `Masterminds/squirrel` — SQL query building
- `log/slog` — structured logging (stdlib)
- `golang.org/x/crypto/scrypt` — password hashing
- `golang.org/x/sync/errgroup` — server lifecycle supervision
- `resend/resend-go` — transactional email (HTML templates in `internal/email/templates/`)
- `stretchr/testify` — test assertions

Do not add a router, ORM, or DI framework.

## Layout and wiring

- `cmd/` — cobra commands. Config precedence: viper defaults → `config.json` →
  env (`MORIA_` prefix; secrets like `HMAC_SIGNING_KEY` bound explicitly in
  `root.go`).
- `route/` — HTTP handlers. Handlers are closures taking their dependencies as
  parameters and returning `http.HandlerFunc`
  (`func Login(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc`). No
  globals, no context-smuggled dependencies.
- `internal/<domain>/` — small single-purpose packages (`user`, `session`,
  `token`, `email`, `database`, `middleware`, `logger`), one file per
  operation (`create.go`, `get.go`, `list.go`, ...).

## Idioms to follow

- **Routing**: Go 1.22+ `ServeMux` patterns — method and path variables in the
  pattern (`"PATCH /users/{id}/password"`, `r.PathValue("id")`).
- **Middleware**: the immutable `middleware.Chain` (`New` → `Append` → `Wrap`).
  `Append` clones (`slices.Clone`) so chains fork safely: base → protected
  (Authentication) → admin. Rate limiting is not middleware — it lives at
  Traefik (`stark/traefik/ratelimit.yaml`).
- **Errors**: wrap with `fmt.Errorf("...: %w", err)` and a lowercase,
  context-carrying message. Use `errors.AsType[T]` for typed checks.
- **Logging**: `slog` with key-value pairs (`"error", err`). JSON handler in
  production; handlers log server errors, not expected auth failures.
- **SQL**: build queries with squirrel via the shared `database.QB`
  (`?` placeholders), execute with sqlx. Column names are `snake_case` and
  match JSON tags — the frontend relies on this.
- **Background goroutines** accept a `context.Context` and exit on
  `ctx.Done()`. Guard shared maps with `sync.RWMutex`; use double-checked
  locking on the write path.
- **Server lifecycle**: `signal.NotifyContext` for SIGTERM, explicit
  `http.Server` with timeouts, `errgroup` to supervise the server and
  shutdown workers, `Shutdown` with a grace window under kubernetes' 30s
  SIGKILL deadline.
  Walkthrough: [graceful-shutdown.md](graceful-shutdown.md).
- **Security**: scrypt with OWASP params (N=32768, r=8, p=1) plus per-user
  salt and `subtle.ConstantTimeCompare`; email verification / password-reset
  tokens are stateless HMAC-SHA256 (`internal/token`), sessions are opaque
  random ids stored server-side. Never roll new crypto formats.

## Testing

Black-box tests live in `test/` (package `test`), not alongside source.
`TestMain` boots the real router with `httptest.Server` against in-memory
SQLite (`file::memory:?_pragma=foreign_keys(1)`), applies `schema/model.sql`,
and seeds via `test/seed.go`. Assertions use testify; logs are discarded
unless `-v`.

```sh
go test ./...          # run tests
go vet ./...           # before committing
golangci-lint run      # lint (config in .golangci.yml, also runs in CI)
govulncheck ./...      # vulnerability scan (also runs in CI)
go run . serve         # local server on :8081
```
