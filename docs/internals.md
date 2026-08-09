# Internals

Go 1.26, stdlib-first: routing, middleware, and handlers are plain
`net/http`. **No web framework, no router library, no ORM, no DI
container.** Persistence is `database/sql` with hand-written SQL.

The whole dependency list: cobra + pflag + viper (CLI/config), `google/uuid`,
`modernc.org/sqlite` (pure Go, so `CGO_ENABLED=0` and a distroless *static*
base image), `x/crypto/scrypt`, `x/sync/errgroup`, `log/slog` from stdlib.

```
main.go            cmd.Execute()
cmd/               root.go (config discovery, registration), serve.go
route/             Initialize + one file per resource
internal/
  database/        DSN, schema application
  middleware/      Chain + Logger/Authentication/Admin
  session/         session model and queries
  user/            user model, queries, scrypt
schema/            model.sql (embedded), test_user.sql
```

## cmd/ — the westhill-engine pattern

Unexported command vars registered in root's `init()`; flags declared in
each command's own `init()`; viper binds in `PreRun`; config loaded via
`cobra.OnInitialize`. `root.go` does config-file discovery, logging setup,
and command registration — nothing else.

Two orderings here are load-bearing and look wrong enough to get
"simplified":

- **Discovery lives in `cobra.OnInitialize`, not `init()`.** Only the former
  runs *after* flag parsing. Read `--config` during `init()` and the
  variable is always still empty — that is the latent bug in
  westhill-engine's version. Don't move it back.
- **`SetEnvPrefix`/`SetEnvKeyReplacer` must run before any `BindEnv`.**
  One-arg `viper.BindEnv` resolves the prefix *at bind time*, not at read
  time. They sit in `OnInitialize`, which runs before `PreRun`, where the
  binds happen.

**Every viper bind lives in the `PreRun` of the command that declares the
flags.** `serve` binds; a command with no flags binds nothing. `PreRun`
names no keys at all:

```go
_ = viper.BindPFlags(cmd.Flags())
cmd.Flags().VisitAll(func(f *pflag.Flag) { _ = viper.BindEnv(f.Name) })
```

so declaring a flag wires flag + file + env in one move. See
[architecture.md](architecture.md) for the resulting key table and the
`MORIA_PORT` naming rule.

Logging is a JSON `slog` handler on stdout, set as the default in
`OnInitialize` so every command gets it.

## Routing and handlers

`route.Initialize(route.Config{DB: db})` returns an `http.Handler` — a plain
`ServeMux` using Go 1.22+ method patterns
(`"PATCH /users/{id}/password"`) with `r.PathValue("id")`. No route
library, no regex.

**Handlers are closures**: they take dependencies as parameters and return
an `http.HandlerFunc`. Request/response structs are declared *inside* the
handler function when they are used nowhere else, which keeps the package
namespace clean:

```go
func CreateUser(db *sql.DB) http.HandlerFunc {
	type Request struct{ ... }
	return func(w http.ResponseWriter, r *http.Request) { ... }
}
```

Error handling is written out at each exit rather than funnelled through a
helper: set the content type, write the status, encode `{"error": ...}`.
Internal causes go to `slog.Error` and never into the response body.

## Middleware — the immutable Chain

`New → Append → Wrap`. `Append` returns a new `Chain` over a `slices.Clone`
of the existing middlewares, so chains **fork** instead of aliasing:

```go
base      := middleware.New(middleware.Logger())
protected := base.Append(middleware.Authentication(db))
admin     := protected.Append(middleware.Admin(db))
```

Appending to `protected` cannot retroactively change `base`. `Wrap` applies
in `slices.Backward` order, so the first middleware in the chain is the
outermost wrapper — `Logger` sees the final status of everything inside it.

- **`Logger`** wraps the `ResponseWriter` to capture the status code,
  defaulting to 200 for handlers that never call `WriteHeader`. It logs
  after the handler returns, with `remote_addr` taken from the first
  `X-Forwarded-For` entry.
- **`Authentication`** reads the `TOKEN` cookie, validates it via
  `session.ByID` (existence *and* expiry in one query), and puts the
  `*session.Session` on the request context under the typed key
  `session.ContextKey`.
- **`Admin`** must run after `Authentication` — it reads the session off the
  context and does a second query for the user's role, so every admin route
  costs one extra user lookup. `UpdateUser` is protected-tier and repeats
  that lookup itself when the target id is not the requester's, which is
  the only place the role is resolved outside the middleware.

## Database

`database.New(path)` opens the DSN, pings, and applies the embedded schema.

**SQLite options ride on the DSN**, never a bare `Exec("PRAGMA ...")` — a
pragma statement reaches only the one pooled connection that happens to
serve it, while the DSN applies to every connection the pool opens:

```
file:<path>?_pragma=foreign_keys(1)
           &_pragma=journal_mode(WAL)
           &_pragma=busy_timeout(5000)
           &_time_format=sqlite
```

- `foreign_keys(1)` — off by default in SQLite; without it the
  `ON DELETE CASCADE` from `sessions` to `users` silently does nothing.
- `journal_mode(WAL)` — readers don't block the writer, and a backup can run
  against a live server.
- `busy_timeout(5000)` — the single-writer contention valve.
- `_time_format=sqlite` — stores datetimes in SQLite's own format, which is
  string-comparable against `datetime('now')`. The session expiry check is
  a lexical comparison and breaks under RFC3339.

`schema/model.sql` is embedded with `//go:embed` and applied on every boot.
It is all `CREATE TABLE IF NOT EXISTS`, so boot is idempotent — there is no
migration tool, and **any schema change must be written to be safe to
re-run**. Two tables: `users` (with a `CHECK` constraining `role`) and
`sessions` (FK to users, cascade on delete).

The optional seed file (`--seed-path`) is read and `Exec`'d after the
schema, then logged at `Warn`. It must be `INSERT OR IGNORE` — in the
cluster it re-applies at every boot against a database that survives.

## Graceful shutdown (`cmd/serve.go`)

Stop taking new work, finish in-flight work, exit before the bulldozer.

- `signal.NotifyContext` turns SIGINT/SIGTERM into context cancellation —
  one wire shuts down everything, and any future background loop hangs off
  the same ctx.
- An explicit `http.Server` with timeouts: `ReadHeaderTimeout: 5s`
  (anti-slowloris), `ReadTimeout: 10s`, `WriteTimeout: 30s`,
  `IdleTimeout: 2m`.
- An `errgroup` supervises two workers: the listener, and a shutdown worker
  that waits on `gctx.Done()` then calls `srv.Shutdown` with a **15s grace,
  deliberately under kubernetes' 30s SIGKILL deadline**. `Shutdown` gets a
  fresh `context.Background()` timeout — reusing the already-cancelled ctx
  would abort the drain instantly.
- `http.ErrServerClosed` is the "closed on purpose" receipt and is treated
  as success; without that check every clean rollout would exit non-zero.

```
kubectl rollout restart deploy/moria
  → SIGTERM → ctx cancels → listener stops accepting
  → in-flight requests finish
  → exit 0 well inside 30s; SIGKILL never fires
```

## Checks

`make check` runs `go vet ./... && golangci-lint run && go test ./...`.

golangci-lint is v2 config; the v2 defaults already enable errcheck, govet,
staticcheck, ineffassign, and unused. On top of that: `gosec` enabled, with
`G104` excluded as an errcheck duplicate, and
`(*encoding/json.Encoder).Encode` excluded from errcheck — a handler that
fails to write its response has nothing useful left to do. Test files are
exempt from errcheck and gosec.

**There are currently no test files.** `go test ./...` passes vacuously.
The intended shape is black-box tests in `test/` (package `test`) with a
`TestMain` booting the real router via `httptest.Server` against in-memory
SQLite — `route.Initialize` already takes its dependencies as a struct,
which is what makes that possible without touching production code.
