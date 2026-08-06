# Moria

> "speak, friend, and enter"

Standalone authentication service (Go) for julian-one.com. Owns users and
sessions; serves registration, login, password recovery, and cluster-internal
session validation. Runs in the k3s cluster with **no public ingress** —
shire's Node server proxies all auth traffic to it.

## Structure

- `cmd/` — cobra CLI (`serve`), viper config
- `route/` — HTTP handlers (user-facing + cluster-internal `/internal/*`)
- `internal/` — domain packages (user, session, token, email, database, middleware)
- `schema/model.sql` — SQLite schema, applied at startup
- `test/` — black-box HTTP tests against in-memory SQLite

## Commands

```sh
go run . serve   # serves :8081
go test ./...
../deploy.sh moria   # build, push, roll the k8s deployment
```

## Docs

- [docs/architecture.md](docs/architecture.md) — the HTTP listener, k8s
  manifests (`stark/moria/`), storage, and how shire/citadel consume moria.
  Read before touching routes, ports, config, or anything deploy-related.
- [docs/security.md](docs/security.md) — trust model (unauthenticated
  `/internal/*`, XFF, no CORS, Traefik rate limiting) and credential design
  (sessions, email tokens, passwords). Read before touching auth, sessions,
  tokens, or middleware.
- [docs/golang.md](docs/golang.md) — libraries, layout, and idioms (handler
  closures, middleware chain, squirrel/sqlx, error wrapping, testing). Read
  before writing Go in this repo.
- [docs/improvements.md](docs/improvements.md) — prioritized backlog from the
  Go best-practices review. Check items off as they land.
