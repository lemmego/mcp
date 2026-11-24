---
title: libSQL
type: docs
prev: docs/database/bunconnector
next: docs/migrations/
weight: 74
sidebar:
  open: true
---

SQLite normally lives in a file your application opens directly. libSQL is a
fork of SQLite that runs as a **server** instead: a process called `sqld` holds
the database, and your application talks to it over HTTP. You get SQLite's SQL
and SQLite's behaviour, with a database that several machines can reach.

Reach for it when you want SQLite's simplicity but your application no longer
runs on one machine. If a single file on one disk is still fine, plain
[SQLite](/docs/configuration/) is simpler and faster.

## Create a project

```bash
lemmego new blog --non-interactive \
  --module github.com/me/blog \
  --database libsql
```

Your project gets one driver import in `bootstrap/providers.go`:

```go
_ "github.com/tursodatabase/libsql-client-go/libsql"
```

## Configure the connection

A libSQL server is addressed by URL, so your `.env` has no `DB_HOST` or
`DB_PORT`:

```bash
DB_DRIVER=libsql
DB_URL=http://127.0.0.1:8080

# The migrator finds the server here. Keep it in step with DB_URL.
DATABASE_URL=http://127.0.0.1:8080
```

You need both. Your application reads `DB_URL`; `lemmego run migrate` reads
`DATABASE_URL` (or `DATABASE_URL_UNPOOLED`, which it checks first). The two
disagree because the migrator builds a DSN from host-and-port style settings
for every other driver, and a URL is not expressible that way.

## Run a server

There is no "create database" step. Point `sqld` at a directory that does not
exist and it builds the database there:

```bash
make libsql
```

That reads the address out of `DB_URL` and keeps the data in
`storage/data/libsql`, which is already in your `.gitignore`.

You rarely need to run it yourself, though — **`lemmego dev` starts one for
you**:

```text
libSQL on http://127.0.0.1:8080, data in /home/me/blog/storage/data/libsql
```

It starts a server only when `DB_DRIVER` is `libsql`, the URL points at your
own machine, and nothing is already answering on that port. If a server is
already running there, `lemmego dev` checks that it can answer a query and
leaves it alone.

{{< callout type="warning" >}}
**Give `--db-path` an absolute path** if you run `sqld` yourself from a script,
a service file or anywhere other than your project root. The path is resolved
against the server's working directory, so a relative one creates a second,
empty database somewhere else — your application connects without complaint
and every table looks as though it was never migrated.
{{< /callout >}}

## Use a hosted server

Point `DB_URL` at it and nothing else changes:

```bash
DB_URL=https://blog-me.turso.io?authToken=...
DATABASE_URL=https://blog-me.turso.io?authToken=...
```

`lemmego dev` starts no local server when the URL is not your own machine.

## Inspect the database

The server keeps a real SQLite file, so any SQLite tool reads it:

```bash
sqlite3 "file:storage/data/libsql/dbs/default/data?mode=ro"
```

Keep `?mode=ro`. The server owns the write-ahead log, and writing to the file
behind its back corrupts the database. Point a GUI at the same path and set it
to read-only.

To run a query against the server instead, post to its HTTP endpoint:

```bash
curl http://127.0.0.1:8080/v2/pipeline \
  -H 'Content-Type: application/json' \
  -d '{"requests":[{"type":"execute","stmt":{"sql":"SELECT 1"}},{"type":"close"}]}'
```

## Behaviour and limits

**Read-only transactions are not available.** The server rejects them. Your
code does not have to care: `orm.LibSQL()` drops the read-only flag and opens
an ordinary transaction, so a read path that asks for one still works. You lose
the safety net that would have turned an unexpected write into an error.

**A transaction cannot sit idle.** The server closes a transaction's stream
after about ten seconds of inactivity, and the error that follows is empty —
`failed to execute SQL:` with nothing after it. Keep slow work that is not a
query — an HTTP call to another service, a large file read — outside your
transactions.

**Some pragmas are rejected.** `busy_timeout`, `journal_mode`, `synchronous`
and the `wal_checkpoint` family all fail, because the server needs sole control
of the write-ahead log. Nothing in your project sets them for you; the queue
notes at debug level that it could not set `busy_timeout` and carries on, which
is correct — there is no local file to wait for a lock on.

**One server is one point of failure.** `sqld` has no leader election, no
quorum and no automatic failover. If the process or its disk goes, the database
is unavailable until you bring it back. Plan for that with backups, or choose
Postgres if you need a database that survives losing a machine.

**Full-text search needs Postgres.** The [search](/docs/search/) module queries
`tsvector` columns, which SQLite and libSQL do not have.

## Next steps

- [Configuration](/docs/configuration/) — where `DB_URL` and the driver import
  come from
- [Migrations](/docs/migrations/) — writing and running them
- [The connection seam](/docs/database/connection-seam/) — how your connection
  reaches the rest of the framework
