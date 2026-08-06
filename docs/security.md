# Security model

How moria authenticates callers and protects credentials. The topology this
relies on (no ingress, who talks to moria) is in
[architecture.md](architecture.md).

## Trust model

The cluster boundary is the only security boundary. Moria has no ingress, so
every reachable caller is an in-cluster service — shire's Node server for the
user-facing routes, citadel for `/internal/*`. Consequences, all deliberate:

- **`/internal/*` is unauthenticated.** `GET /internal/sessions/{id}` and
  `GET /internal/users?ids=...` are trusted service-to-service lookups. They
  are safe only because nothing outside the cluster can reach them; they must
  never be exposed through an ingress.
- **`X-Forwarded-For` is trusted.** In-cluster callers set it to the browser
  address Traefik derived from the real TCP source
  (`middleware.GetClientIP`). It feeds request logging only — nothing
  security-relevant keys on it.
- **No CORS.** All calls are server-to-server; browsers never talk to moria
  directly.
- **Rate limiting lives at Traefik**, not in moria
  (`stark/traefik/ratelimit.yaml`): per-IP token buckets on POSTs to the
  public `/register`, `/login`, and `/forgot-password` pages, keyed on the
  real TCP source address. Moria never sees the real address, so it cannot
  rate limit meaningfully.

## Sessions

- Opaque server-side sessions: a UUIDv4 `session_id` row in the `sessions`
  table, no claims embedded. Created on login and on registration completion.
- Lifetime is `session.SessionDuration` (24h). Shire owns the browser cookie:
  it sets the session id as `TOKEN` on `julian-one.com` (HttpOnly, Secure,
  SameSite=Strict, same 24h) and forwards it to moria on each request. Moria
  only ever reads the cookie.
- The `TOKEN` cookie is the **only** accepted credential
  (`middleware.Authentication`). Bearer/query-string fallbacks were removed
  with the cutover (improvement A1) — no session credential is readable by
  browser JS.
- Expiry is stored in UTC and enforced on read (`session.Validate` /
  `IsValid`). The hourly `session.PurgeLoop` is hygiene only — it stops the
  table accumulating dead rows, it is not the enforcement point.
- Citadel validates sessions through `/internal/sessions/{id}` via a caching
  client (60s TTL), so revocation propagates within that window.

## Email tokens (verification and password reset)

Stateless, signed with HMAC-SHA256 under `HMAC_SIGNING_KEY`
(`internal/token`). Format:

```
base64url(JSON(claims)) . base64url(HMAC-SHA256(payload, key))
```

- Claims carry a `purpose` (`verify` or `reset`) that `token.Verify`
  requires, so a verification token cannot be replayed as a reset token or
  vice versa.
- TTLs: verification 24h, reset 30 minutes.
- A reset token embeds a SHA-256 digest of the password hash it was issued
  against (`Claims.Binding`, checked by `Claims.BoundTo`). Once the password
  changes the binding no longer matches, so a used or stale reset link
  self-invalidates without any server-side token state. The payload is
  readable by the token holder — hence a digest of the hash, never the hash
  itself.
- The payload is signed, not encrypted: never put secrets in claims.

## Passwords

- scrypt with OWASP 2024 parameters (N=32768, r=8, p=1, keyLen=32) and a
  32-byte per-user random salt (`internal/user/password.go`).
- Comparison uses `subtle.ConstantTimeCompare`.
- Account-existence leaks are closed at the API level:
  - `Login` burns a dummy scrypt comparison when the identifier matches no
    user, so response timing does not reveal whether the account exists.
  - `ForgotPassword` returns the same 202 whether or not the email exists.

## Request hardening

- Every handler is wrapped in a 1 MiB body cap (`middleware.BodyLimit`), so
  oversized bodies fail at the JSON decoder instead of being read in full.
- Server timeouts (anti-slowloris `ReadHeaderTimeout`, etc.) and the SIGTERM
  drain are covered in [graceful-shutdown.md](graceful-shutdown.md).
