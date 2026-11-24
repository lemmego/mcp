---
title: Column Types Reference
type: docs
prev: docs/migrations/schema-builder
next: docs/migrations/writing-migrations
weight: 82
sidebar:
  open: true
---

## Methods

Every method on `*migration.Table` that declares a column, with the type each
driver gets. The generic name in the first column is the `ColType*` constant,
which you only need when calling `AddColumn` directly.

### Integers

| Method | PostgreSQL | MySQL | SQLite |
|---|---|---|---|
| `Increments(name)` | `SERIAL` | `INT` | `INTEGER` |
| `BigIncrements(name)` | `BIGSERIAL` | `BIGINT` | `INTEGER` |
| `TinyInt(name)` | `SMALLINT` | `TINYINT` | `TINYINT` |
| `SmallInt(name)` | `SMALLINT` | `SMALLINT` | `SMALLINT` |
| `MediumInt(name)` | `INTEGER` | `MEDIUMINT` | `MEDIUMINT` |
| `Int(name)` | `INTEGER` | `INT` | `INTEGER` |
| `BigInt(name)` | `BIGINT` | `BIGINT` | `BIGINT` |

`UnsignedTinyInt`, `UnsignedSmallInt`, `UnsignedMediumInt`, `UnsignedInt` and
`UnsignedBigInt` are the same types with `Unsigned()` applied — which adds
`UNSIGNED` on MySQL and nothing on PostgreSQL or SQLite.

There is no `Integer` or `BigInteger`. The methods are `Int` and `BigInt`.

### Decimals

| Method | PostgreSQL | MySQL | SQLite |
|---|---|---|---|
| `Float(name, precision, scale)` | `FLOAT` | `FLOAT` | `FLOAT` |
| `Double(name, precision, scale)` | `DOUBLE PRECISION` | `DOUBLE` | `DOUBLE` |
| `Decimal(name, precision, scale)` | `DECIMAL` | `DECIMAL` | `DECIMAL` |

Pass `0, 0` to `Double` unless you mean a precision — see the note under
dates.

### Text

| Method | PostgreSQL | MySQL | SQLite |
|---|---|---|---|
| `Char(name, length)` | `CHAR` | `CHAR` | `CHAR` |
| `String(name, length)` | `VARCHAR` | `VARCHAR` | `VARCHAR` |
| `Text(name)` | `TEXT` | `TEXT` | `TEXT` |
| `Json(name)` | `JSONB` | `JSON` | `TEXT` |
| `UUID(name)` | `UUID` | `CHAR` | `TEXT` |
| `ULID(name)` | `CHAR` | `CHAR` | `TEXT` |

`Json` gives `JSONB` on PostgreSQL rather than `JSON`: it stores a parsed
form, so it is cheaper to read than to write and can carry a GIN index. The
cost is that key order is not preserved and duplicate keys collapse — both
properties of JSON objects rather than of this choice. SQLite has no JSON
type; it stores JSON as `TEXT` and its JSON1 functions operate on `TEXT`, so
that is the answer rather than a fallback.

**MySQL cannot index a JSON column at all**, and no key length helps.
`Validate` says so, pointing at a generated column over the extracted value.
The same check covers `TEXT` and `BLOB` columns, which MySQL refuses in a key
without a length.

### Dates and times

| Method | PostgreSQL | MySQL | SQLite |
|---|---|---|---|
| `Date(name)` | `DATE` | `DATE` | `DATE` |
| `Time(name, precision)` | `TIME` | `TIME` | `TIME` |
| `DateTime(name, precision)` | `TIMESTAMP` | `DATETIME` | `DATETIME` |
| `DateTimeTz(name, precision)` | `TIMESTAMP WITH TIME ZONE` | `DATETIME` | `DATETIME` |
| `Timestamp(name, precision)` | `TIMESTAMP` | `TIMESTAMP` | `TIMESTAMP` |
| `TimestampTz(name, precision)` | `TIMESTAMP WITH TIME ZONE` | `TIMESTAMP` | `TIMESTAMP` |

**Pass `0` for the precision on the `Tz` variants.** It is appended after the
whole type name, so `TimestampTz("created_at", 6)` emits
`TIMESTAMP WITH TIME ZONE(6)` and PostgreSQL answers
`syntax error at or near "("`. With `0` you get the type and no suffix, which
is what was wanted. `DateTime` and `Timestamp` take a precision safely,
because theirs is one word.

### Other

| Method | PostgreSQL | MySQL | SQLite |
|---|---|---|---|
| `Boolean(name)` | `BOOLEAN` | `BOOLEAN` | `BOOLEAN` |
| `Binary(name)` | `BYTEA` | `BINARY` | `BINARY` |
| `Enum(name, values...)` | `TEXT` + `CHECK` | `ENUM(...)` | `TEXT` |

**`Enum` enforces nothing on SQLite.** MySQL gets a native `ENUM` and
PostgreSQL a `TEXT` with a `CHECK (col IN (...))` appended, but SQLite gets
plain `TEXT` and no constraint at all — so the values are checked on two
drivers of three. Use a `String` and a `Check` when you need all three:

```go
t.String("status", 16).Default("draft")
t.Check("posts_status_valid", "status IN ('draft','published','removed')")
```

## Timestamps and soft deletes

There are no `Timestamps()` or `SoftDeletes()` shortcuts. Declare them:

```go
t.TimestampTz("created_at", 0).Default(migration.Expr("now()"))
t.TimestampTz("updated_at", 0).Default(migration.Expr("now()"))
t.TimestampTz("deleted_at", 0).Nullable()
```

A soft-delete column must be nullable, and the model field must be
`*time.Time` or `sql.NullTime` — the ORM rejects a bare `time.Time` at
metadata time, because its zero value writes a real timestamp and
`deleted_at IS NULL` would then match nothing and hide the whole table. See
[Soft deletes](/docs/orm/writes#soft-deletes).

## Next steps

- [Writing migrations](/docs/migrations/writing-migrations) — using these
  columns in a migration file.
- [Schema builder](/docs/migrations/schema-builder) — the keys and constraints
  around them.
