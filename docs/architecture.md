# Architecture

Moria is the standalone authentication service for julian-one.com. It owns the
`users` and `sessions` tables (extracted from citadel during the auth cutover)
and is the only service that touches credentials.

## One listener, one binary

`moria serve` starts a single HTTP server on :8081. It is not reachable from
outside the cluster — moria has **no ingress**; shire's Node server is the
only path from the internet to the auth API.

Two route families share the listener:

- **User-facing** — registration, login/logout, password recovery,
  user/session management. Called server-side by shire via ClusterIP
  `moria:80`. Enforces session/admin auth. Rate limiting happens upstream
  at Traefik (`stark/traefik/ratelimit.yaml`).
- **`/internal/*`** — `GET /internal/sessions/{id}` (session validation) and
  `GET /internal/users?ids=...` (batch user hydration). Called by citadel.
  Unauthenticated, trusted service-to-service lookups only.

The trust model behind these choices (unauthenticated internals, XFF, no
CORS, upstream rate limiting) and the credential design (sessions, email
tokens, passwords) are in [security.md](security.md).

### Why no ingress

Until the 2026-07 cutover the public listener was exposed at
`auth.julian-one.com` and the browser called it directly. That forced CORS,
a Bearer-token copy of the session id in client-side JS (readable by any
XSS), and made the two-listener split a load-bearing security boundary
protecting the unauthenticated `/internal/*` routes. Routing everything
through shire removed all three: the cluster boundary is now the only
topology that matters and no session credential ever reaches the browser.
The ingress, certificate, and `auth.julian-one.com` DNS record were deleted
with the cutover, and the second listener was collapsed into the first once
the port split stopped being a security boundary (improvement A3).

Routes are wired in `route/init.go` (`Initialize`).

## Role in the k3s cluster

The cluster runs on a Raspberry Pi 4 ("jarvis") with k3s, Traefik ingress, and
cert-manager (Let's Encrypt). Manifests live in the sibling repo
`stark/moria/`:

- `deployment.yaml` — single replica, pinned to jarvis via `nodeSelector`,
  image `julianone/moria:latest`. Secrets `RESEND_API_KEY` and
  `HMAC_SIGNING_KEY` come from the `moria-secrets` Secret.
- `service.yaml` — ClusterIP `moria`: port 80 → 8081 (auth API, including
  `/internal/*`). There is no ingress and no certificate — nothing outside
  the cluster can reach moria.
- `configmap.yaml` — mounted as `/app/config.json` (db path, port).
- `cronjob-backup.yaml` — daily (04:10) `sqlite3 .backup` of `moria.db`,
  keeping the 7 most recent.
- `job-migrate.yaml` — one-off users/sessions migration out of `citadel.db`;
  only run during cutover with citadel and moria scaled to 0.

## Storage

SQLite at `/app/data/moria.db`, backed by a `hostPath` volume (`/data/moria`)
on jarvis. This is why the deployment is a single replica pinned to one node —
do not scale it up. Schema (`schema/model.sql`) is applied idempotently at
startup.

Connection options ride on the DSN so they apply to every pooled connection —
a bare `Exec("PRAGMA ...")` only reaches the one connection it happens to run
on (`internal/database/init.go`):

- `foreign_keys(1)` — referential integrity checks
- `journal_mode(WAL)` — concurrent reads, so the backup CronJob can run
  against a live server
- `busy_timeout(5000)` — wait under contention instead of failing with
  `SQLITE_BUSY`
- `_time_format=sqlite` — store `time.Time` in SQLite's `datetime()` format,
  so string comparisons against `datetime('now')` (session expiry, purge)
  keep working

## Who talks to moria

- **shire (Node server)** proxies all auth traffic; the browser never calls
  moria. Base URL comes from the server-only `MORIA_API_URL`
  (`http://moria.industries.svc.cluster.local`, see
  `shire/src/lib/controllers/moria.ts`). Shire owns the browser session
  cookie: it sets the opaque session id from login/registration responses as
  `TOKEN` on `julian-one.com` (HttpOnly, Secure, SameSite=Strict, 24h), and
  on each request forwards it to moria as a `Cookie` header along with the
  originating client IP as `X-Forwarded-For` for request logging
  (`shire/src/lib/controllers/context.server.ts`). There is no CORS
  handling — all calls are server-to-server.
- **citadel** never reads the users/sessions tables. It validates the `TOKEN`
  cookie and hydrates usernames through the `/internal/*` routes
  (`http://moria.industries.svc.cluster.local`) via a caching client
  (`citadel/internal/auth/client.go`, 60s TTL — revocation propagates within
  that window).

## Deploy

From the repo root: `./deploy.sh moria` — buildx builds and pushes
`julianone/moria:latest`, then rolls the deployment. The SQLite driver is
pure Go (`modernc.org/sqlite`), so the Dockerfile builds a static
`CGO_ENABLED=0` binary and ships it on distroless `static` as the `nonroot`
user; an initContainer in `stark/moria/deployment.yaml` chowns the hostPath
data dir to uid 65532 before startup.
