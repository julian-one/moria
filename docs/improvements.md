# Improvements

Tracked findings from the 2026-07 Go best-practices review, plus follow-ups
from the cluster-internal cutover. Ordered by priority, not effort. Check
items off as they land.

## Cluster-internal cutover (landed 2026-07)

Moria (and citadel) no longer have a public ingress: shire's Node server is
the only internet-facing service and proxies all auth traffic to moria over
cluster DNS (`MORIA_API_URL`), forwarding the `TOKEN` cookie plus the client
IP as `X-Forwarded-For` for request logging (rate limiting has since moved
to Traefik, `stark/traefik/ratelimit.yaml`). CORS handling
was deleted, `stark/moria/{ingress,certificate}.yaml` are gone, and shire
strips `session_id` from page data, so no session credential ever reaches
browser JS. Details in [architecture.md](architecture.md).

Follow-ups this unlocked:

- [x] **A1. Delete dead auth fallbacks** — the `Authorization: Bearer` and
  `?token=` paths in `internal/middleware/authentication.go` were unreachable
  by design (every caller sends the cookie). Removed; the TOKEN cookie is now
  the only accepted credential. Supersedes item 4.
- [x] **A2. Vestigial Set-Cookie** — shire sets the browser cookie itself
  from the login/registration response body, so moria's
  `session.SetSessionCookie`/`ClearSessionCookie` headers were ignored by
  every caller. Removed, along with the hardcoded cookie `Secure`/`SameSite`
  constants (resolving item 8); `session/cookie.go` now holds only
  `SessionDuration` and `CookieName`.
- [x] **A3. Collapse the two listeners** — with no ingress the
  public/internal split was no longer a security boundary. `/internal/*`
  moved onto :8081 and the second `http.Server` in `cmd/serve.go` was
  deleted. Citadel's `moria.base_url` now points at port 80; a temporary
  8082 → 8081 compat mapping in the k8s Service covered the rollout and was
  removed once citadel rolled.

## Correctness and security

- [x] **1. Token purpose confusion** — tokens now carry a `purpose` claim
  (`verify`/`reset`) that `token.Verify` requires; reset TTL is 30 min
  (verification stays 24h); reset tokens embed a digest of the current
  password hash (`Claims.BoundTo`) so they self-invalidate once the password
  changes. (`internal/token/token.go`, `route/auth.go`)
- [x] **2. SQLite options must ride on the DSN** — `PRAGMA foreign_keys` via
  `db.Exec` only applied to one pooled connection. Now set per-connection via
  DSN: `_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000`.
  (`internal/database/init.go`)
- [x] **3. Graceful shutdown + server timeouts** — bare `http.ListenAndServe`
  replaced with `http.Server` (timeouts set), `signal.NotifyContext`,
  `errgroup`, and `Shutdown` with a 15s grace on SIGTERM. Walkthrough in
  [graceful-shutdown.md](graceful-shutdown.md). (`cmd/serve.go`)
- [x] **4. Session token accepted via query string** — removed with A1: the
  `?token=` fallback (and Bearer) are gone from authentication middleware.
  (`internal/middleware/authentication.go`)
- [x] **5a. Request body limits** — `middleware.BodyLimit` in the base chain
  wraps every handler with a 1 MiB `http.MaxBytesReader` cap.
  (`internal/middleware/bodylimit.go`, `route/init.go`)
- [x] **5b. Login timing side channel** — `Login` burns a dummy scrypt
  comparison when the user is not found. (`route/auth.go`)
- [x] **5c. UTC consistency** — `session.Create` now stores expiry from
  `time.Now().UTC()`, matching the `datetime('now')` comparison in
  `session.Validate`. (`internal/session/create.go`)
- [x] **5d. Purge expired sessions** — hourly `session.PurgeLoop` goroutine,
  supervised by the serve errgroup, exits on ctx cancellation.
  (`internal/session/purge.go`, `cmd/serve.go`)

## Go idioms

- [x] **8. Hardcoded env values → config** — resolved by deletion: the CORS
  origins half went away with the cutover, and the cookie `Secure`/`SameSite`
  constants were deleted with A2.
- [x] **9. Delete dead code** — `OptionalAuthentication` removed from
  `internal/middleware/authentication.go`.

## Tooling and build

- [x] **10. Dockerfile** — switched to `modernc.org/sqlite` (pure Go) for a
  `CGO_ENABLED=0` static binary; multi-stage build shipping on distroless
  `static` as `nonroot`. The deployment gained a chown initContainer and a
  non-root securityContext (`stark/moria/deployment.yaml`).
- [x] **11. Lint + CI** — `.golangci.yml` (v2 defaults + gosec) is clean,
  GitHub Actions workflow runs test/lint/govulncheck
  (`.github/workflows/ci.yml`); version derives from build info instead of
  the hardcoded `"0.1.0"` (`cmd/root.go`).
