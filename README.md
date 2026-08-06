# Moria

## _"speak, friend, and enter"_

Auth API for julian-one.com: users + sessions, scrypt-hashed passwords, cookie `TOKEN`, roles `admin`/`user`.
Session tokens are 32 random bytes, returned once by `POST /login` as `token`; only their SHA-256 is stored as the
session's `session_id`, so the secret never appears in the database. PostgreSQL-backed — `schema/model.sql` applies
idempotently at startup. Timestamps are `timestamptz`: the app writes `time.Now().UTC()` and column defaults use
`now()`.

## Run

Config precedence: flags > `MORIA_*` env > `.moria.json` in cwd.

| Flag             | Env                  | Default |
| ---------------- | -------------------- | ------- |
| `--listen-port`  | `MORIA_LISTEN_PORT`  | `8081`  |
| `--database-url` | `MORIA_DATABASE_URL` | —       |

Local dev reads the gitignored `.moria.json`, which points at `moria_dev` on friday (`192.168.20.223:5432`, role `moria`, password in 1Password item `moria-postgres`):

```sh
go run . serve
```

A fresh database can't mint its first admin through the API (`POST /users` is admin-gated):

```sh
MORIA_PASSWORD=... go run . create-user --username julian --email ... --role admin
```

## Test

Tests create throwaway `moria_test_*` databases on the target server (the role has `CREATEDB`) and drop them on cleanup:

```sh
MORIA_TEST_DATABASE_URL="postgres://moria:$(op item get moria-postgres --fields password --reveal)@192.168.20.223:5432/moria_dev" go test ./...
```

## Deploy

Runs in-cluster only (`http://moria` from the `web` namespace; `kubectl port-forward` for direct access); manifests
live in the stark repo at `moria/moria.yaml`, and the prod DSN reaches the pod via Secret `moria-database`.

```sh
TAG=$(date -u +%Y%m%d-%H%M%S)
docker build --platform linux/arm64 -t julianone/moria:$TAG --push .
```

Then set the new tag in stark's `moria/moria.yaml` and `kubectl --context stark apply -f moria/`.
