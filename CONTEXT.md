# Moria

The auth context for julian-one.com: it owns users, their credentials, and the sessions that prove who a caller is.

## Language

**Token**:
The secret a client holds: 32 random bytes minted at login, shown once, never stored.
_Avoid_: session_id, cookie value, bearer

**Session ID**:
The SHA-256 digest of a Token: the stored, non-secret identifier of a Session.
_Avoid_: hash, token

**Session**:
The time-boxed grant a Token unlocks.

**Mint**:
To create a Session: generates its Token and stores only the Session ID.
_Avoid_: create session

**Requester**:
The authenticated caller of a protected route: the user and session resolved once from a Token.
_Avoid_: principal, current user

**Identifier**:
The username or email a user signs in with.
_Avoid_: login, handle

**Role**:
A user's authority level: admin or user. Parsed at every entry point, so an invalid role is unrepresentable inside
the context.
