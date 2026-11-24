---
title: ORM Overview
type: docs
prev: docs/frontend/vite
next: docs/orm/queries
weight: 50
sidebar:
  open: true
---

Your data lives in a database, in tables of rows and columns. Your Go code
works in structs. Something has to carry values across that gap, and if you do
it yourself you write SQL as strings, scan each column into each field by
hand, and find your mistakes at runtime.

An **ORM** — object-relational mapper — does that carrying for you. You write
Go, and it writes the SQL:

```go
users, err := db.Model[User]().Where(orm.Eq("active", true)).All(ctx)
```

That becomes `SELECT * FROM users WHERE active = $1`, and the rows come back
as `[]User` with every field already filled in. Note what the `[T]` buys you:
you get `[]User`, not a bag of `any` you have to assert. Ask for a field that
does not exist and your build fails rather than your page.

The **Lemmego ORM** is the one `lemmego new` sets up by default, built
directly on Go's own `database/sql`.

It stands alone. Its only dependencies are `database/sql` and two small naming libraries, shared with the CLI and migration modules so that table names agree. It imports no other Lemmego package, and nothing about it requires the framework:

```go
sqlDB, _ := sql.Open("pgx", dsn)
db := orm.Open(sqlDB, orm.Postgres())
```

That is the whole setup.

### Supported databases

PostgreSQL, MySQL and SQLite. Every feature documented in this section is exercised against all three on each test run.

## ORM or GPA?

Both ship with the framework, and they are not rivals.

| | Lemmego ORM | [GPA](/docs/database/) |
|---|---|---|
| Scope | SQL only | SQL, MongoDB, Redis, and more |
| API | Query builder with relationships | Uniform `Repository[T]` per backend |
| Relationships | `hasOne`, `hasMany`, `belongsTo`, many-to-many | Preload by name |
| Best for | A SQL application you want to write SQL-shaped code against | Code that must run across storage engines |

If you want both, [`gpaorm`](/docs/orm/gpa-adapter) implements the GPA contracts on top of the ORM, so code written against `gpa.Repository[T]` runs on it unchanged.

## Installation

A project scaffolded with `lemmego new` already has it. To add it by hand:

```bash
go get github.com/lemmego/orm
go get github.com/lemmego/ormconnector   # wires it into the app lifecycle
```

Register the connector in `bootstrap/providers.go`:

```go
func LoadProviders() []app.Provider {
    return []app.Provider{
        &fs.Provider{},
        &session.Provider{},
        &ormconnector.Provider{},
        // ...
    }
}
```

The connector reads the same `sql` configuration the other connectors read, opens the connection, and publishes `*orm.DB` to the service container:

```go
db := app.Get[*orm.DB](a)   // or ormconnector.Get(a)
```

It builds its DSN with the migration module's own `DataSource` and derives the ORM dialect from the same driver name, so the ORM and `lemmego migrate` always resolve the same database.

## A first query

```go
type User struct {
    ID        int       `orm:"primaryKey;autoIncrement"`
    Email     string    `orm:"column:email_address"`
    Active    bool
    CreatedAt time.Time
    UpdatedAt time.Time
    Posts     []Post    `orm:"hasMany:UserID"`
}

users, err := db.Model[User]().
    Where(orm.Eq("active", true)).
    With("Posts").
    OrderBy(orm.Desc("created_at")).
    Limit(20).
    All(ctx)
```

`Model[T]` and `Raw[T]` are generic **methods**, available on both `*orm.DB` and `*orm.Tx`, so a transaction is a drop-in replacement for the connection.

## Conventions

- **Table names** come from the model name: snake_case, then pluralised. `Category` becomes `categories`. This is the same pipeline the CLI and migration modules use, so a model and a generated migration agree. Implement `TableName() string` to override.
- **Column names** default to the snake_case field name. `orm:"column:email_address"` overrides.
- **Primary keys** are declared with `orm:"primaryKey"`, and `autoIncrement` marks a generated key so it is written back to the entity after insert.
- A `db:"..."` tag is honoured as a fallback, so models generated before the ORM existed map without edits.
- Fields that cannot round-trip through a single column — nested structs, slices of structs — are skipped rather than mapped.

## Next steps

- [Queries](/docs/orm/queries) — predicates, projections, pagination, scopes
- [Relationships](/docs/orm/relationships) — eager loading and associations
- [Writes and transactions](/docs/orm/writes) — upserts, soft deletes, savepoints
- [Field descriptors](/docs/orm/field-descriptors) — compile-time checked columns
- [GPA adapter](/docs/orm/gpa-adapter) — running GPA code on the ORM
