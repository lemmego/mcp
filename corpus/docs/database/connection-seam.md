---
title: The connection seam
type: docs
prev: docs/database/provider-registry
next: docs/database/generic-repository
weight: 62
sidebar:
  open: true
---

## When to use it

Reach for `api/db` when you are writing a package that needs its own tables —
a queue, an OAuth2 server, a session store — and cannot know which SQL layer
the application chose.

The service container keys on a concrete type, so resolving "the database"
would mean naming `*orm.DB`, `*gorm.DB` or `*bun.DB` at compile time. The GPA
registry keys on a vendor string, is off by default, and under GORM and Bun
makes the provider and the native handle mutually exclusive, so depending on
it breaks a default project.

The seam exists so such a package does not fall back to re-deriving a DSN from
configuration and opening a second connection. Two independent readings of one
config tree drift, and the result is a package quietly running against a
different database from the rest of the application.

## The seam

```go
import "github.com/lemmego/api/db"

conn, ok := db.Resolve(a)
if !ok {
    // This application has no database. That is a normal state for a
    // project scaffolded with --database none, not a failure.
}

pool := conn.SQLDB()       // *sql.DB, whichever ORM opened it
dialect := conn.Dialect()  // db.SQLite, db.MySQL, db.Postgres, db.SQLServer
name := conn.Name()        // the sql.connections key it was built from
```

Every SQL connector registers one during bootstrap — `ormconnector`,
`gormconnector` and `bunconnector` alike, and in both GPA and non-GPA mode.

## What it does not cover

Quoting, upserts, `RETURNING` and locking hints are not here. Those live in
`orm.Dialect` and in the queue's own dialect layer, which serve different
needs; merging them would turn a connection seam into a SQL-generation
library.

What a package writing one `INSERT` actually needs is the pool, the dialect,
and parameter placeholders:

```go
query := "INSERT INTO widgets (id, name) VALUES (" +
    db.Placeholder(dialect, 1) + ", " + db.Placeholder(dialect, 2) + ")"
```

PostgreSQL numbers its parameters; the others do not.

## Order matters

Providers run one at a time in the order `LoadProviders` lists them, and each
can only resolve what the ones before it registered. **The database connector
must come before anything that stores something**:

```go
return []app.Provider{
    &fs.Provider{},
    &session.Provider{},
    &ormconnector.Provider{},   // first among the providers that own resources
    &cache.Provider{},
    &queue.Provider{},
    &oauth2.Provider{},
}
```

A scaffolded project gets this right. A hand-edited one might not, and the
symptom is quiet: the subsystem falls back to opening its own connection, or
refuses to start, rather than failing loudly at the point of the mistake.

## Two connectors

The first connector to register wins, and a later one is ignored rather than
causing a panic. An application deliberately wiring two — one ORM for its own
models, another for a legacy schema — boots with a deterministic answer: the
first listed in `LoadProviders` is the application's connection.

## A borrowed connection

A `Connection` is borrowed, never owned. The connector that registered it
closes the pool during shutdown, so **do not call `Close` on the value
`SQLDB()` returns** — it would take the rest of the application down with it.

## No database at all

`Resolve` reporting `false` is an ordinary state, not an error. A package that
cannot work without a database should say so with a message naming the fix:

```go
conn, ok := db.Resolve(a)
if !ok {
    return errors.New("widgets: this application has no database; " +
        "configure a connection in internal/configs/database.go and add a " +
        "connector to bootstrap/providers.go")
}
```

A provider returning an error is turned into a panic by the bootstrapper,
which prints a goroutine dump — so the actionable part belongs first, where
someone will actually read it.

## Next steps

- [Running migrations](/docs/migrations/running-migrations) — which database
  the migrator connects to.
- [Session](/docs/session) — the driver that takes its dialect from your
  connection.
- [Queue](/docs/queue) — the other subsystem that borrows the connection.
