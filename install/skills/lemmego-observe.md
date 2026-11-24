## When to use this skill

When debugging what one request did, or when the observe dashboard shows
requests with nothing underneath them.

## The driver wrapper is what makes it work

`observe` collects queries at the **`database/sql` driver layer**, not through
an ORM hook. One wrapper therefore covers the ORM, raw `database/sql`, the
migrator, the queue's job store and the session store; an ORM-level hook would
cover one of those.

The consequence is that the application must register a wrapped driver and open
its connection through it:

    package plugins

    import (
        "database/sql"

        "github.com/lemmego/observe"
        "github.com/lib/pq"
    )

    const ObservedDriver = "postgres-observed"

    func init() {
        sql.Register(ObservedDriver, observe.Wrap(&pq.Driver{}))
    }

**An empty dashboard is almost always a missing or unused wrapper.** It reads
as "this application makes no queries" rather than as "nothing is instrumented",
which is why it is worth checking first.

Two details that bite:

- The driver value's shape differs per driver. `&pq.Driver{}` and
  `&mysql.MySQLDriver{}` are pointers; `libsql.Driver{}` is a **value**, because
  upstream registers it as `sql.Register("libsql", Driver{})`.
- `observe.Wrap` takes a `driver.Driver`. Calling it with no argument compiles
  nowhere, but a template or scaffold that forgets a branch produces exactly
  that — `not enough arguments in call to observe.Wrap`.

## The dialect still comes from DB_DRIVER

Registering `postgres-observed` changes **who opens the connection, not what is
sent over it**. `DB_DRIVER` still supplies the dialect, so the migrator and the
ORM generate the same SQL as before. Do not set `DB_DRIVER=postgres-observed` —
the connector takes the registered driver name separately.

## Reading a recording

Prefer the MCP tools over the dashboard when you are the one debugging:

- `observe-requests` — recent requests
- `observe-request` — one request and everything it caused
- `observe-slow-queries`, `observe-repeated-queries` — the N+1 finder
- `observe-errors`, `observe-tagged`

`observe-repeated-queries` is the one to reach for when a page is slow: the same
statement with different bindings is the signature of a missing `With`.

## The root is standard library only

`observe.Wrap` works outside a Lemmego application, which is deliberate — so a
plain `database/sql` program can use it. `observe.New(store, opts...)` with
`store/memory.New()` builds a recorder by hand; `rec.Close(ctx)` flushes it.
`Pause` and `Resume` exist; there is no `Enable` or `Flush`.

## Check first

`search-docs` section `observe`, `module-api` on `observe` for signatures. In a
generated project the wrapper lives in `internal/plugins/observe_driver.go` —
read it before writing a second one.
