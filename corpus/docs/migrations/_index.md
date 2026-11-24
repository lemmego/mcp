---
title: Migrations
type: docs
prev: docs/database/libsql
next: docs/migrations/schema-builder
weight: 80
sidebar:
  open: true
---

Your database has a shape — which tables exist, which columns they have. That
shape has to change as you build, and it has to change the same way on your
laptop, on your colleague's, and on the server. Doing it by hand means somebody
eventually runs the wrong statement, or forgets one, and finds out when the
application breaks against a database that is subtly different.

A **migration** is that change written down as a file: one function to apply
it, one to undo it. You commit those files alongside your code, so the shape
of your database has a history exactly like the rest of your project, and any
machine can be brought up to date by running them in order.

```go
func Up(tx *sql.Tx) error   // add the table
func Down(tx *sql.Tx) error // remove it again
```

`lemmego/migration` runs them for you against SQLite, MySQL or PostgreSQL, and
gives you a **schema builder** so a column you declare once is rendered
correctly for whichever of the three you are on. Where a database can do
something the other two cannot, `tx.Exec` lets you write the SQL directly —
that is expected, not a workaround.

```go
func mig_20250706000001_create_users_table_up(tx *sql.Tx) error {
    schema := migration.Create("users", func(t *migration.Table) {
        t.BigIncrements("id").Primary()
        t.String("email", 255)
        t.String("handle", 30)
        t.TimestampTz("created_at", 0).Default(migration.Expr("now()"))

        t.Check("users_handle_format", `handle ~ '^[a-z0-9][a-z0-9_]{1,29}$'`)
        t.Index("email").Unique().Name("users_email_key")
    }).Build()

    _, err := tx.Exec(schema)
    return err
}

func mig_20250706000001_create_users_table_down(tx *sql.Tx) error {
    _, err := tx.Exec(migration.Drop("users").Build())
    return err
}
```

## What it does

- **Bidirectional** — an up and a down per migration, so a deploy is
  reversible.
- **Dialect-aware** — one declaration, three renderings. Unsigned integers,
  timestamps with zones, inline versus standalone indexes, and every
  `ALTER TABLE` drop action differ between the drivers, and the builder knows
  how.
- **Refuses rather than guesses** — a partial index on MySQL, a tolerant drop
  on MySQL, a column change on SQLite: `Validate` reports each instead of
  emitting SQL the server will reject, or worse, SQL it will accept with the
  part you wanted silently missing.
- **Tracked** — applied migrations are recorded in `schema_migrations`,
  grouped into a batch per run.

## Getting started

Generate a migration:

```shell
lemmego g migration create_users_table
```

Apply it:

```shell
lemmego run migrate up
```

Then read [Schema Builder](/docs/migrations/schema-builder) for what you can
declare, [Column Types](/docs/migrations/column-types) for what each method
gives you per driver, [Writing Migrations](/docs/migrations/writing-migrations)
for the shape of a file and the one-transaction rule, and
[Running Migrations](/docs/migrations/running-migrations) for the CLI.

## Common pitfalls

Each is covered again in context. These three are worth knowing up front:

- **`TimestampTz(name, 0)`** — any other precision emits
  `TIMESTAMP WITH TIME ZONE(6)`, which PostgreSQL rejects.
- **`Enum` enforces nothing on SQLite** — MySQL gets a native `ENUM` and
  PostgreSQL a `TEXT` with a `CHECK`, but SQLite gets plain `TEXT` and no
  constraint. Use `String` plus `Check` if you need all three.
- **`ForeignID` is not a foreign key column** — it declares an
  auto-incrementing one. Use `UnsignedBigInt` plus `Foreign`.

## Next steps

- [Schema builder](/docs/migrations/schema-builder) — the fluent API for
  tables, columns and keys.
- [Writing migrations](/docs/migrations/writing-migrations) — the shape of a
  migration file, up and down.
- [Running migrations](/docs/migrations/running-migrations) — the commands, and
  what `--step` counts.
