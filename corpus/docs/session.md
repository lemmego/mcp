---
title: Session
type: docs
prev: docs/security/encryption
next: docs/authz/
weight: 105
sidebar:
  open: true
---

Sessions keep per-visitor state between requests — who is signed in, a flash
message, a half-filled form. The `session` package provides them over four
storage backends and supplies the store that
[CSRF protection](/docs/security/csrf) keeps its token in.

Session data is loaded when a request starts and committed when the response
completes. There is nothing to save by hand.

## Register the provider

```go
// bootstrap/providers.go
func LoadProviders() []app.Provider {
    return []app.Provider{
        &ormconnector.Provider{},   // registers db.Connection
        &session.Provider{},        // must come after it for the database driver
    }
}
```

Order only matters for the `database` driver, which borrows the application's
connection. The other three can go anywhere.

## Configure it

Settings live in `internal/configs/session.go`:

```go
config.Set("session", config.M{
    "driver":    config.MustEnv("SESSION_DRIVER", "file"),
    "lifetime":  config.MustEnv("SESSION_LIFETIME", 120),   // minutes
    "cookie":    config.MustEnv("SESSION_COOKIE", "lemmego_session"),
    "domain":    config.MustEnv("SESSION_DOMAIN", ""),
    "path":      "/",
    "secure":    config.MustEnv("SESSION_SECURE", false),
    "http_only": true,
    "same_site": "lax",
})
```

Set `SESSION_SECURE=true` in production so the cookie is only sent over HTTPS.

## Read and write session data

From a request:

```go
// Read
val := c.Session("cart_id")
str := c.SessionString("cart_id")

// Write
c.PutSession("cart_id", id)

// Read and remove — flash messages
val := c.PopSession("notice")
str := c.PopSessionString("notice")
```

Outside a request, or for operations on the session itself, reach the manager:

```go
sess := app.Get[*session.Session](a)

err := sess.RenewToken(ctx)
err := sess.Destroy(ctx)
userID := sess.GetString(ctx, "user_id")
sess.Put(ctx, "last_activity", time.Now())
```

It embeds `scs.SessionManager`, so its whole API is available.

## Choosing a driver

| Driver | Survives a restart | Shared across instances | Needs |
|---|---|---|---|
| `memory` | no | no | nothing |
| `file` | yes | no | a writable, persistent directory |
| `redis` | yes | yes | a Redis server |
| `database` | yes | yes | your SQL database and a table |

`memory` signs everybody out whenever the process restarts, which on a
container platform means every deploy. Use it for local development and
tests.

On a platform with no persistent volume and no Redis, `database` is the only
durable option that costs nothing extra to run.

### memory

```go
config.Set("session", config.M{"driver": "memory"})
```

### file

```go
config.Set("session", config.M{"driver": "file"})
```

Sessions are written to `storage/session/`, with expired files cleaned up
periodically. The directory must outlive the process — on a container
platform that means a mounted volume.

### redis

```go
config.Set("session", config.M{"driver": "redis"})
config.Set("keyvalue", config.M{
    "connections": config.M{"redis": config.M{
        "host": config.MustEnv("REDIS_HOST", "127.0.0.1"),
        "port": 6379,
    }},
})
```

### database

```go
config.Set("session", config.M{"driver": "database"})
```

Sessions live in a `sessions` table in your own database. The driver takes its
dialect from the connection your SQL connector registered, so the same setting
works on **PostgreSQL, MySQL and SQLite**.

Two things are required.

**1. Register `&session.Provider{}` after your database connector.** It
resolves the connection through `db.Resolve`, and providers run in
`LoadProviders` order, so a session provider listed first finds nothing. It
refuses to boot in that case and names the fix, rather than falling back to
another store and signing everybody out.

**2. Create the table yourself.** Each dialect reads `expiry` differently, so
there is no portable definition:

{{< tabs >}}
{{< tab name="PostgreSQL" >}}
```sql
CREATE TABLE sessions (
    token  TEXT        NOT NULL PRIMARY KEY,
    data   BYTEA       NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions (expiry);
```
{{< /tab >}}
{{< tab name="MySQL" >}}
```sql
CREATE TABLE sessions (
    token  CHAR(43) COLLATE utf8mb4_bin NOT NULL PRIMARY KEY,
    data   BLOB         NOT NULL,
    expiry TIMESTAMP(6) NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions (expiry);
```
{{< /tab >}}
{{< tab name="SQLite" >}}
```sql
CREATE TABLE sessions (
    token  TEXT NOT NULL PRIMARY KEY,
    data   BLOB NOT NULL,
    expiry REAL NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions (expiry);
```
{{< /tab >}}
{{< /tabs >}}

`lemmego new --session database` writes this migration for the database you
chose. Write it as raw `tx.Exec` rather than through the
[schema builder](/docs/migrations/schema-builder): the builder would add an
`id` and timestamp columns, producing a table the store cannot query.

Keep the index. Without it, every sweep of expired rows scans the table.

{{< callout type="info" >}}
The `database` driver needs `api` v0.6.5 or later. See
[Upgrade notes](/docs/appendix/upgrading#minimum-versions).
{{< /callout >}}

## Sessions and CSRF

The CSRF middleware stores its token in the session under `_token`. The token
is generated on the session's first request and reused for the session's
lifetime; [CSRF Protection](/docs/security/csrf) explains why it is not
rotated per request.

This ties the two together in ways worth knowing:

- **Destroying or regenerating a session invalidates its CSRF token.** The
  next request is issued a fresh one.
- **Signing in does not break the current page.** `auth.Login` rotates the
  session id but carries the data across, so the token survives and the
  request immediately after a sign-in is not rejected as expired.
- **Signing out does.** `auth.Logout` destroys the session, so the next page
  load issues a new token.

## Session fixation

`auth.Login` rotates the session id before writing the user into the session,
and `auth.Logout` destroys the session rather than only clearing the user.
Both are handled for you.

If you manage sessions yourself, rotate at your own sign-in boundary:

```go
sess := app.Get[*session.Session](a)

if err := sess.RenewToken(ctx); err != nil {   // before storing the identity
    return err
}
sess.Put(ctx, "user_id", user.ID)

// ...and on the way out
err := sess.Destroy(ctx)
```

Without the rotation, an attacker who can plant a session cookie in someone's
browser before they sign in still holds a valid id afterwards, authenticated
as them.

{{< callout type="warning" >}}
An application configured with `DisableSession: true` authenticates with JWT
and has no session to fixate — but it also has nothing to revoke.
`auth.Logout` clears the cookie, and a copy of the token taken beforehand
stays valid until it expires. Server-side logout and the fixation defence are
the same mechanism, and they are absent together.
{{< /callout >}}

## When a session cannot be read

A stored session sometimes cannot be decoded — the file is corrupt, or the
record holds a `gob`-encoded type the running binary no longer registers,
which is the ordinary case after a deploy that removed one.

The framework treats such a session as **absent**: it logs a warning, drops
the cookie and serves the request anonymously, so the visitor is signed out
and can sign back in. No error page is shown.

## Next steps

- [CSRF Protection](/docs/security/csrf) — the token that lives in the session.
- [Authentication](/docs/security/authentication) — what writes a user into it.
- [Deployment](/docs/deployment/#sessions-on-a-container-platform) — choosing a
  driver for your platform.
