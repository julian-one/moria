# The HTTP contract

What shire depends on. Moria speaks JSON over plain HTTP to exactly one
caller; every response body is either the resource or `{"error": "..."}`,
always with `Content-Type: application/json`.

## Sessions and the TOKEN cookie

- **Opaque server-side sessions.** A UUIDv4 `session_id` row in `sessions`,
  no claims embedded. Created on login. Lifetime 24h (`session.Duration`).
- **Expiry is enforced on read.** `session.ByID` selects with
  `expires_at > datetime('now')`, so an expired row is indistinguishable
  from a missing one. There is no purge loop; expired rows accumulate
  harmlessly and are cleared by a `DELETE /users/{id}/sessions` or by the
  `ON DELETE CASCADE` when the user goes.
- **Shire owns the browser cookie; moria only ever reads it.** Moria returns
  the session as a JSON body and **never sends `Set-Cookie`**. Shire sets
  the session id as `TOKEN` — `HttpOnly`, `SameSite=Strict`, `Path=/`,
  `expires` copied from `expires_at`.
- **`TOKEN` is the only credential moria accepts.** No `Authorization:
  Bearer`, no `?token=`. The one exception is login itself, which carries
  `Authorization: Basic base64(identifier:password)`.

`expires_at` is computed in SQL (`datetime('now', '+N seconds')`), not bound
as a Go `time.Time` — a bound value stores RFC3339, which breaks the lexical
comparison `ByID` relies on.

### Shapes

```jsonc
// Session
{ "session_id": "...", "user_id": "...", "expires_at": "...", "created_at": "..." }

// User — password_hash and salt are excluded at the SQL layer, not by tags
{ "user_id": "...", "username": "...", "email": "...",
  "role": "admin" | "user", "created_at": "...", "updated_at": "..." }

// Collections
{ "items": [ ... ], "total": 0 }
```

## Routes

Three access tiers, built as forked middleware chains (see
[internals.md](internals.md)): **base** (no auth), **protected** (valid
`TOKEN`), **admin** (valid `TOKEN` + role `admin`).

| method + pattern             | tier      | success           |
|------------------------------|-----------|-------------------|
| `GET /health`                | base      | 200 status/time   |
| `POST /login`                | base      | 200 Session       |
| `POST /logout`               | base      | 204               |
| `POST /users`                | admin     | 201 User          |
| `GET /users`                 | admin     | 200 collection    |
| `GET /users/{id}`            | protected | 200 User          |
| `PATCH /users/{id}`          | protected | 200 User          |
| `PATCH /users/{id}/password` | protected | 204               |
| `PATCH /users/{id}/role`     | admin     | 204               |
| `GET /sessions/{id}`         | protected | 200 Session       |
| `DELETE /sessions/{id}`      | admin     | 204               |
| `GET /users/{id}/sessions`   | admin     | 200 collection    |
| `DELETE /users/{id}/sessions`| admin     | 204               |

Beyond the tier, two handlers apply their own ownership check:
`PATCH /users/{id}` allows self **or** admin; `PATCH /users/{id}/password`
allows **self only** — an admin cannot set another user's password through
it.

### Login

`POST /login` with `Authorization: Basic base64(identifier:password)`.
`identifier` matches **email or username**. Returns the Session; shire turns
it into the cookie.

- 401 `Invalid basic auth credentials` — header missing or malformed
- 401 `Invalid credentials` — unknown identifier *or* wrong password, one
  indistinguishable answer (see [security.md](security.md))

### Logout

`POST /logout` reads the `TOKEN` cookie and deletes that session row. It is
deliberately **best-effort and always 204**: a missing cookie or a failed
delete still succeeds, because shire's logout action clears the browser
cookie unconditionally and a 500 here would only strand the user with a
cookie they cannot shed. Failures are logged, not surfaced.

Note it sits on the **base** chain, not protected — an already-invalid
session must still be able to log out.

### Per-request hydration

Shire's `hooks.server.ts` runs two round-trips on every authenticated
request: `GET /sessions/{id}` (existence + expiry), then
`GET /users/{user_id}`. Any failure — invalid, expired, or moria
unreachable — deletes the cookie and degrades to logged-out. No
cross-request cache; a cache would trade TTL for revocation lag.

## Status codes

| code | when                                                          |
|------|---------------------------------------------------------------|
| 400  | undecodable body, missing required field, invalid `role`       |
| 401  | no/invalid `TOKEN` on a protected route; bad login credentials  |
| 403  | admin route without the role; ownership check failed            |
| 404  | `{id}` matched no row                                          |
| 409  | username or email already taken (on `POST /users`)             |
| 500  | database or hashing failure — details logged, never returned    |

Two rough edges worth knowing before shire keys on them: a taken username
is **409 on create but 400 on update**, and `PATCH /users/{id}` returns 403
with the message `You can only update your own username` even when the
real cause was an admin-role lookup failure.

## What moria does not do

The stack-wide doc describes a self-service registration and email-token
flow. **None of it is built, and the current design does not include it:**

- No `/register`, `/verify`, `/forgot-password`, `/reset-password`.
- No HMAC email tokens, no `HMAC_SIGNING_KEY`, no Resend integration.
  There are no secrets in the deployment at all — only the seed Secret.
- **Accounts are provisioned by admins** via `POST /users`, which is why
  that route is admin-tier and there is no unauthenticated way to create a
  user.

Rate limiting also lives at Traefik, not here: moria never sees the real TCP
source (only shire's), so it cannot rate limit meaningfully.
