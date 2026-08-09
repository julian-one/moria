# Security model

**The cluster boundary is the only security boundary.** Every consequence
below follows from that one decision, and each is deliberate.

## 1. The trust model

- **Moria has no ingress, no IngressRoute, and no certificate.** Nothing
  outside the cluster can reach it — verified: `:8081` is refused from the
  LAN. This is not defence in depth; it is the defence.
- **The browser never talks to moria.** Shire's Node server is the sole
  caller and the sole path from the internet to the auth API. There is
  therefore no CORS handling anywhere, because there are no cross-origin
  browser calls.
- **`X-Forwarded-For` is trusted**, because only in-cluster callers can set
  it. It feeds request logging only — *nothing security-relevant keys on
  it*. Keep it that way: the moment an authorization decision reads that
  header, shire becomes able to forge it.
- **Rate limiting lives at Traefik, not here.** Moria only ever sees shire's
  pod IP as the TCP source, so it cannot rate limit meaningfully.

### Why it is this shape

The pre-burn moria was exposed at `auth.julian-one.com` and browsers called
it directly. That forced CORS, a Bearer-token copy of the session id held in
client JS (XSS-readable), and a load-bearing two-listener split. Routing
everything through shire deleted all three at once. **No session credential
reaches the browser** — `TOKEN` is `HttpOnly`, so even an XSS on shire
cannot read it.

Any future proposal to expose moria directly re-imports all three problems.

## 2. Credentials

- **`TOKEN` is the only credential moria accepts** on protected routes. The
  old `Authorization: Bearer` and `?token=` fallbacks are gone.
- **Sessions are opaque.** A UUIDv4 row id with no embedded claims — nothing
  to forge, nothing to tamper with, and revocation is a `DELETE`.
- **Expiry is enforced on read**, in the same query that fetches the
  session (`expires_at > datetime('now')`). There is no window where a
  purge loop's lateness extends a session's life.
- Login is the one exception: `Authorization: Basic` over the in-cluster
  hop.

## 3. Passwords

`internal/user/password.go`:

- **scrypt** with OWASP 2024 parameters — `N=32768, r=8, p=1, keyLen=32`.
- **32-byte cryptographically random salt per user**, stored as a BLOB
  alongside the hash. `Hash(password, nil)` generates a fresh one; passing
  the stored salt re-derives for verification.
- **`subtle.ConstantTimeCompare`** for the comparison, never `==`.
- **`password_hash` and `salt` never leave the package.** The `columns`
  const used by every read query deliberately omits them, so no handler can
  serialize them by accident — the guard is at the SQL layer, not in struct
  tags.

## 4. Enumeration resistance

- `ErrInvalidCredentials` covers **both** an unknown identifier and a wrong
  password, and both surface as the same 401 `Invalid credentials`.
- When the identifier matches no user, `Authenticate` **burns a dummy scrypt
  comparison** before returning, so response timing does not reveal whether
  the account exists. The dummy runs the same 32768-round derivation; keep
  it if you refactor that function, and keep it *after* the `ErrNoRows`
  branch rather than short-circuiting.
- Shire adds the third leg: unknown and protected routes are
  indistinguishable when logged out — both 302 to `/login`.

`POST /users` does return distinct 409s for a taken username vs. a taken
email, which is an enumeration oracle — acceptable because the route is
admin-tier.

## 5. Authorization

Three tiers as forked middleware chains: **base**, **protected** (valid
`TOKEN`), **admin** (`+ role == admin`). Both `Authentication` and `Admin`
**fail closed** — any error in the lookup produces 401/403, never a
pass-through.

Roles are constrained twice: a `CHECK (role IN ('admin', 'user'))` in the
schema and `Role.Valid()` before every write. Two handlers add an ownership
check on top of the tier:

- `PATCH /users/{id}` — self **or** admin
- `PATCH /users/{id}/password` — **self only**, so an admin cannot set
  another user's password through it

## 6. Known gaps

Real, currently unmitigated, and recorded here so they are decisions rather
than surprises. Nothing here is reachable from the internet except through
shire's authenticated routes.

1. **No ownership check on reads.** `GET /users/{id}` and
   `GET /sessions/{id}` are protected-tier only — any authenticated user who
   knows an id can read that user's record (including email) or that
   session's `user_id` and expiry. Both ids are UUIDv4, so they cannot be
   enumerated, and the admin-only `GET /users` is what would hand them out;
   the practical exposure is small but the check is genuinely missing.
2. **No request body size limit.** Handlers decode JSON straight from an
   unbounded `r.Body`. The stack-wide doc describes a 1 MiB `BodyLimit`
   middleware — it is not implemented. The `Chain` is the obvious place.
3. **Password change requires no current-password confirmation**, and does
   not invalidate the user's other sessions. A stolen session can therefore
   change the password and persist; `DELETE /users/{id}/sessions` exists but
   nothing calls it automatically.
4. **No password strength policy.** `CreateUser` and `UpdatePassword` check
   only that the field is non-empty.
5. **The seed Secret holds a real admin credential** and re-applies at every
   boot. `INSERT OR IGNORE` means a rotated password is not clobbered, but
   the seeded account cannot be removed by editing the Secret alone — delete
   the row.
6. **No tests.** There is no coverage asserting that the admin chain is
   actually wired to the admin routes, which is the kind of regression a
   refactor introduces silently.
