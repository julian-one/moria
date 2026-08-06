# Login returns a `token` field, not a session record

`POST /login` used to answer with a `Session` whose `session_id` field was overwritten with the raw secret, so the
same field name meant "secret token" in the login response and "stored digest" everywhere else — two call sites in
shire disagreed about which one they held, and the type system could not tell them apart. We decided login answers
`{token, expires_at}` and a `session_id` always means the digest, accepting the wire-format break because shire is
the only client and both repos deploy together.
