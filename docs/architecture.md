# Architecture

Moria is the cluster-internal auth API for julian-one.com: one Go binary,
one listener, one SQLite file. It has **no ingress and no certificate** —
the only thing that can reach it is shire's Node server, over cluster DNS.

For the stack-wide view (router, k3s, Traefik, shire) see
[`../../docs/architecture.md`](../../docs/architecture.md). This directory
covers moria only:

| doc                            | scope                                        |
|--------------------------------|----------------------------------------------|
| architecture.md (this)         | topology, deployment, config surface         |
| [contract.md](contract.md)     | the HTTP contract shire depends on           |
| [internals.md](internals.md)   | code layout, middleware, SQLite, shutdown    |
| [security.md](security.md)     | trust model, passwords, roles                |

```
shire (Node/SvelteKit, :3000)
   │ http://moria.industries.svc.cluster.local   ClusterIP moria:80
   ▼
moria (:8081) ──▶ /app/data/moria.db  (hostPath /data/moria, jarvis only)
```

## 1. Position in the cluster

- Namespace `industries`. ClusterIP `moria:80 → 8081`. No IngressRoute, no
  cert, no NodePort — verified unreachable from the LAN.
- **The browser never talks to moria.** Shire's server is the sole caller,
  so there is no CORS handling anywhere and no session credential ever
  reaches browser JS. See [security.md](security.md) for why this is the
  whole security model.
- Plain HTTP inside the cluster; TLS terminates at Traefik.

## 2. Deployment (`stark/moria/`)

`strategy: Recreate`, not RollingUpdate — SQLite takes exactly one writer
and every pod generation mounts the same hostPath, so overlapping
generations would put two writers on one file.

- **`nodeSelector: kubernetes.io/hostname: jarvis`** — the hostPath is
  node-local, so the database exists only on jarvis. Never unpin it and
  **never scale up**: a pod landing on friday comes up *healthy* against an
  empty database rather than failing, which is the worst failure shape
  available.
- **initContainer `data-permissions`** chowns `/app/data` to 65532 and
  chmods 700. `DirectoryOrCreate` makes the hostPath root-owned and the
  container runs as distroless nonroot, which cannot otherwise write it.
- **readinessProbe on `/health`. No liveness probe** — liveness pointed at
  the same endpoint as readiness turns transient slowness into restart
  storms, and a crashed process restarts via `restartPolicy` regardless.
- Env is only `MORIA_DATABASE_PATH=/app/data/moria.db` and
  `MORIA_SEED_PATH=/app/seed/seeding.sql`, the latter from the `moria-seed`
  Secret mounted read-only at `/app/seed`. The seed re-applies on every
  boot against a database that now survives; that is safe **only** because
  the seed is `INSERT OR IGNORE`, so existing rows — including changed
  passwords — are left alone. Keep any future seed file `INSERT OR IGNORE`.

### Image

Multi-stage: `golang:1.26-alpine` build → `gcr.io/distroless/static-debian12:nonroot`.
`CGO_ENABLED=0` (pure-Go `modernc.org/sqlite`, so no libc in the runtime
image), `-trimpath`, `ENTRYPOINT /usr/local/bin/moria` with `CMD ["serve"]`.

Build and roll with `./deploy.sh moria` from the repo root: buildx arm64 →
Docker Hub `julianone/moria:latest` → rollout. The `:latest` tag defaults
the pull policy to `Always`, so a rollout restart re-pulls.

## 3. Config surface — one name, three places

Three keys, all declared as flags on `serve`:

| flag              | env                    | `.moria.json`     | default    |
|-------------------|------------------------|-------------------|------------|
| `--listen-port`   | `MORIA_LISTEN_PORT`    | `"listen-port"`   | `8081`     |
| `--database-path` | `MORIA_DATABASE_PATH`  | `"database-path"` | `moria.db` |
| `--seed-path`     | `MORIA_SEED_PATH`      | `"seed-path"`     | `""`       |

Precedence: viper defaults → `.moria.json` → env → flags.

Declaring a flag wires all three surfaces at once — there is no second list
to keep in sync. `PreRun` names no keys: it calls `viper.BindPFlags` then
`VisitAll` with one-arg `viper.BindEnv(f.Name)`. See
[internals.md](internals.md) for why that ordering is load-bearing.

### The naming rule that must not be broken

k8s **service links** are on (the default): every Service in `industries`
injects docker-links vars into every pod — `MORIA_PORT=tcp://10.43.x.x:80`,
`MORIA_SERVICE_HOST`, and friends. Nothing consumes them; discovery is
cluster DNS.

They are inert **only because** moria's keys bind to names the injector can
never produce. The flag is `--listen-port` and not `--port` precisely
because `MORIA_PORT` arrives in every pod as `tcp://10.43.x.x:80` and once
crash-looped the pod with `too many colons in address`.

There is **no `AutomaticEnv`** — the env surface is a closed list, so a
stray `MORIA_*` var cannot become a config key on its own. That closes the
general hole but not this one: the danger is a *declared* flag whose name
collides. Name new flags and secrets clear of `MORIA_PORT*` and
`MORIA_SERVICE_*`.

## 4. Local development

From the repo root:

- `make moria` — moria alone, seeded from `schema/test_user.sql`
- `make dev` — moria + shire, Ctrl-C stops both
- `make check` — `go vet ./... && golangci-lint run && go test ./...`
- `make validate` — curls `/health`, then greps shire's SSR home page for
  the badge (requires `make dev` running)

`.moria.json` is gitignored; it sets the same three keys for a bare
`go run . serve`. The seeded dev account is `test` / `test@example.com`
with role `admin` (`schema/test_user.sql`).
