---
title: Writing Migrations
type: docs
prev: docs/migrations/column-types
next: docs/migrations/running-migrations
weight: 83
sidebar:
  open: true
---

## Structure

A migration is a Go file in `internal/migrations/` with three parts: an
`init` that registers it, an up function, and a down function.

```go
package migrations

import (
    "database/sql"

    "github.com/lemmego/migration"
)

func init() {
    migration.GetMigrator().AddMigration(&migration.Migration{
        Version: "20250706000001",
        Up:      mig_20250706000001_create_users_table_up,
        Down:    mig_20250706000001_create_users_table_down,
    })
}

func mig_20250706000001_create_users_table_up(tx *sql.Tx) error {
    schema := migration.Create("users", func(t *migration.Table) {
        t.BigIncrements("id").Primary()
        t.String("email", 255)
        t.String("name", 255)
        t.TimestampTz("created_at", 0).Default(migration.Expr("now()"))
        t.TimestampTz("updated_at", 0).Default(migration.Expr("now()"))

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

The version is a `YYYYMMDDHHMMSS` string, and the CLI generates it:

```shell
lemmego g migration create_users_table
```

`Drop` returns a `*Schema`, so it needs `.Build()` before it reaches `Exec`.
`return migration.Drop("users")` does not compile, and neither does
`tx.Exec(migration.Drop("users"))`.

## The whole batch is one transaction

`migrate up` opens **one** transaction, runs every pending migration in it,
and commits once — with a hardcoded 30-second budget for all of them
together. That has three consequences worth knowing before you write
anything unusual:

- **A failure anywhere rolls back everything** in that run, not just the
  migration that failed. There are no savepoints between them.
- **`CREATE INDEX CONCURRENTLY` is impossible**, because it cannot run inside
  a transaction.
- **Thirty seconds is the budget**, so a migration that rewrites a large
  table needs to be split or run outside the tool.

PostgreSQL's DDL is transactional, so `CREATE FUNCTION`, `CREATE TRIGGER` and
`CREATE EXTENSION` are all fine inside it.

## Raw SQL

`tx.Exec` is the sanctioned escape hatch, not a workaround — the signature is
`func(tx *sql.Tx) error` precisely so that anything the builder cannot say is
still sayable. Mix the two freely:

```go
func mig_up(tx *sql.Tx) error {
    schema := migration.Create("posts", func(t *migration.Table) {
        t.BigIncrements("id").Primary()
        t.UnsignedBigInt("user_id")
        t.Text("body").Default("")
        t.TimestampTz("deleted_at", 0).Nullable()

        t.Foreign("user_id").References("id").On("users").OnDelete("CASCADE")
        t.Check("posts_body_present", "length(body) > 0")
        t.Index("user_id", "deleted_at").Name("posts_author_idx")
    }).Build()

    if _, err := tx.Exec(schema); err != nil {
        return err
    }

    // PostgreSQL-only, so there is nothing for the builder to render per
    // dialect: a generated column and the index over it.
    for _, statement := range []string{
        `ALTER TABLE posts ADD COLUMN search_vector tsvector
           GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED`,
        `CREATE INDEX posts_search_idx ON posts USING GIN (search_vector)`,
    } {
        if _, err := tx.Exec(statement); err != nil {
            return err
        }
    }
    return nil
}
```

What belongs in each half is in
[What still needs raw SQL](/docs/migrations/schema-builder#what-still-needs-raw-sql).
The short version: if all three drivers can do it, declare it, because the
declaration renders differently on each and hand-written SQL does not.

## Writing a down that works

A down migration returns the schema to what the *previous* version of your
code creates. Two things make that harder than it looks.

**Drop in reverse dependency order.** `migration.Drop` emits a bare
`DROP TABLE IF EXISTS` with no `CASCADE`, so a table still referenced by a
foreign key refuses to go:

```go
func mig_down(tx *sql.Tx) error {
    for _, table := range []string{"post_comments", "posts"} {
        if _, err := tx.Exec(migration.Drop(table).Build()); err != nil {
            return err
        }
    }
    return nil
}
```

**Use `IfExists` when you are undoing a change to a table you did not
create.** A rollback should not fail because the constraint it removes is
already gone:

```go
migration.Alter("users", func(t *migration.Table) {
    t.DropCheck("users_handle_not_reserved").IfExists()
}).Build()
```

MySQL has no `IF EXISTS` on an `ALTER TABLE` drop action, so `Validate`
refuses it there — see
[Altering a table](/docs/migrations/schema-builder#altering-a-table).

## Files outlive the code they were written against

A migration that has run is history. Changing what it emits changes what a
*fresh* database gets and nothing else, so an existing database and a new one
drift apart — which is how a `pg_dump` comparison stops being able to tell
you anything.

If you do rewrite old migrations — to adopt a builder feature that did not
exist when they were written, say — the check that makes it safe is a schema
diff rather than a review:

```shell
createdb before && createdb after
# migrate `before` on the old code, `after` on the new
pg_dump --schema-only --no-owner before > before.sql
pg_dump --schema-only --no-owner after  > after.sql
diff before.sql after.sql
```

An empty diff means the rewrite is a restatement. A non-empty one names
exactly what you changed — usually an index or constraint name, which is the
thing worth not changing.

## Next steps

- [Running migrations](/docs/migrations/running-migrations) — applying and
  rolling back what you wrote.
- [Column types reference](/docs/migrations/column-types) — the column methods
  available to you.
- [Deployment](/docs/deployment/) — running migrations on the way to
  production.
