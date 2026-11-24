## When to use this skill

When dispatching background work or adding a scheduled job. `queue` is the
Lemmego provider; `tasker` is the engine underneath it. **Use `queue` in an
application** — reach for `tasker` directly only if you are writing something
that is not a Lemmego app.

## Provider order is load-bearing

`queue` borrows the application's database connection rather than opening its
own, so `&queue.Provider{}` must be listed in `LoadProviders` **after the
database connector**. Listed before it, the queue has nothing to resolve.

This is the most common way a correct queue setup does nothing, and it is
silent — the same class of mistake as a database-backed session driver listed
before the connector.

## Register the SQL driver yourself

The module does not import a `database/sql` driver, so the application does:

    import _ "github.com/lib/pq"              // postgres
    import _ "github.com/glebarez/go-sqlite"  // sqlite
    import _ "github.com/go-sql-driver/mysql" // mysql

`lemmego new` writes the one the project chose. Adding a database to an
existing project means adding this yourself. It fails at boot with
`database/sql`'s own message naming the forgotten import, on every environment,
so it is not a mistake you can ship.

It used to import all three, so every project linked the two it had never
configured — the pure-Go sqlite driver alone is twenty-nine packages of
translated libc.

## Running it

    lemmego run tasker:work --queue=default --workers=3
    lemmego run tasker:schedule      # the cron scheduler — exactly one process

**`tasker:schedule` must run as exactly one process.** Two schedulers mean
every cron job fires twice.

A dashboard is mounted automatically at `/tasker/` with a JSON API beneath it;
`Config.RoutePrefix` moves it. In a project with authentication the scaffold
gates it on an admin list read from `TASKER_ADMINS`, which is **empty by
default** — that closes the dashboard rather than opening it, so an empty list
is not a misconfiguration to fix by removing the gate.

## Configuration

`&queue.Provider{}` with no config auto-configures from the application's
config, and `.env` overrides that. Pass `Config` explicitly only when you need
something the env cannot express.

## Check first

`search-docs` section `queue`, `module-api` on `queue`. The README documents the
configuration priority order, which is where surprises live.
