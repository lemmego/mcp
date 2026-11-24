---
title: Schema Builder
type: docs
prev: docs/migrations/
next: docs/migrations/column-types
weight: 81
sidebar:
  open: true
---

## What the builder gives you

A migration is `func(tx *sql.Tx) error`, so `tx.Exec` with hand-written SQL
always works. The reason to declare a table instead is that the same
declaration comes out differently on each driver, and hand-written SQL is
correct for exactly one of them.

One table, built for all three:

```go
migration.CreateFor(migration.DriverPostgres, "circle_members", func(t *migration.Table) {
    t.UnsignedBigInt("circle_id")
    t.UnsignedBigInt("user_id")
    t.String("role", 12).Default("member")
    t.TimestampTz("joined_at", 0).Default(migration.Expr("CURRENT_TIMESTAMP"))
    t.PrimaryKey("circle_id", "user_id")
    t.Index("user_id", "joined_at").Name("circle_members_user_idx")
})
```

| | PostgreSQL | MySQL | SQLite |
|---|---|---|---|
| `UnsignedBigInt` | `BIGINT` | `BIGINT UNSIGNED` | `BIGINT` |
| `TimestampTz` | `TIMESTAMP WITH TIME ZONE` | `TIMESTAMP` | `TIMESTAMP` |
| `Index` | a separate `CREATE INDEX` | an inline `INDEX` clause | a separate `CREATE INDEX` |

MySQL takes its index inside the table and its unsigned integers in the type.
Neither is something SQL written for PostgreSQL would get right.

## Create, alter, drop

```go
schema := migration.Create("users", func(t *migration.Table) {
    t.BigIncrements("id").Primary()
    t.String("email", 255)
    t.TimestampTz("created_at", 0).Default(migration.Expr("now()"))
}).Build()

_, err := tx.Exec(schema)
```

`Build` returns the whole statement, newline-joined when there is more than
one — a table plus the `CREATE INDEX` lines that follow it. `Statements`
returns them separately, which a driver that will not take several at once
needs:

```go
for _, statement := range migration.Create("users", func(t *migration.Table) {
    t.BigIncrements("id").Primary()
    t.Index("id").Name("users_id_idx")
}).Statements() {
    if _, err := tx.Exec(statement); err != nil {
        return err
    }
}
```

`Build` panics on a schema that cannot be rendered. `Validate` returns the
same error if you would rather check:

```go
if err := migration.Create("users", func(t *migration.Table) {
    t.BigIncrements("id").Primary()
}).Validate(); err != nil {
    return err
}
```

Altering and dropping:

```go
alter := migration.Alter("users", func(t *migration.Table) {
    t.Int("view_count").Default(0)
    t.String("nickname", 80).Nullable().Change()
    t.DropColumn("legacy_body")
    t.RenameColumn("body", "body_source")
}).Build()

drop := migration.Drop("users").Build()
```

`Drop` returns a `*Schema` like the others — `Build` turns it into
`DROP TABLE IF EXISTS users`. It is not an error value, and
`return migration.Drop("users")` does not compile.

`CreateFor`, `AlterFor` and `DropFor` take a dialect instead of reading
`DB_DRIVER`, which is how you render DDL for a database you are not connected
to — and how these examples are tested.

## Columns

```go
t.BigIncrements("id").Primary()
t.String("email", 255)
t.Text("bio")
t.Int("view_count").Default(0)
t.Boolean("is_published").Default(false)
t.Json("payload").Nullable()
t.TimestampTz("created_at", 0)
```

The full list is in [Column Types](/docs/migrations/column-types). Three are
worth knowing before you reach it, because the obvious call is the wrong one:

**`TimestampTz(name, 0)` — always zero.** The precision is appended after the
whole type name, so `TimestampTz("created_at", 6)` emits
`TIMESTAMP WITH TIME ZONE(6)`, which PostgreSQL rejects with a syntax error.
Pass `0` and you get `TIMESTAMP WITH TIME ZONE`, which is what you wanted.
The same applies to `Double(name, 0, 0)`.

**`Enum` emits no constraint on SQLite.** MySQL gets a native `ENUM` and
PostgreSQL a `TEXT` with a `CHECK` appended, but SQLite gets plain `TEXT` and
nothing else — so the values you listed are enforced on two drivers of three.
Use a `String` and a `Check` when you need all three:

```go
t.String("status", 16).Default("draft")
t.Check("posts_status_valid", "status IN ('draft','published','removed')")
```

**`ForeignID` is not a foreign key column.** It adds the column as an
auto-increment type, so on PostgreSQL `t.ForeignID("user_id").Constrained()`
produces `user_id BIGSERIAL NOT NULL` — a foreign key with its own sequence,
where a row inserted without one silently gets the next number and points at
whichever user that happens to be. Declare the column and the key separately:

```go
t.UnsignedBigInt("user_id")
t.Foreign("user_id").References("id").On("users").OnDelete("CASCADE")
```

### Modifiers

```go
t.String("email", 255).Unique()
t.String("name", 255).NotNull()
t.String("phone", 255).Nullable()
t.BigIncrements("id").Primary()
t.Int("views").Unsigned()
t.String("nickname", 80).Nullable().Change()   // in an Alter
```

`Unsigned()` on PostgreSQL emits nothing, and never has: suffixes are decided
before the column is marked. MySQL honours it, because its `UNSIGNED` is read
when the type is rendered.

### Defaults

`Default` takes a value and renders it as the right SQL literal for its Go
type — strings quoted and escaped, numbers and booleans bare, `nil` as
`DEFAULT NULL`:

```go
t.String("status", 255).Default("active")      // DEFAULT 'active'
t.String("note", 255).Default("it's fine")     // DEFAULT 'it''s fine'
t.Int("age").Default(18)                       // DEFAULT 18
t.Boolean("admin").Default(false)              // DEFAULT FALSE on PostgreSQL, 0 elsewhere
t.String("nickname", 255).Nullable().Default(nil)
```

For an *expression* rather than a value, wrap it in `migration.Expr`, which is
emitted verbatim:

```go
t.Int("attempts").Default(migration.Expr("(1 + 1)"))
t.TimestampTz("created_at", 0).Default(migration.Expr("now()"))
```

Insert-time timestamps have their own value, because MySQL requires the
default's fractional-second precision to match the column's —
`CurrentTimestamp` handles that, where a hand-written
`Expr("CURRENT_TIMESTAMP")` is rejected on a `DATETIME(6)` column:

```go
t.DateTime("created_at", 6).Default(migration.CurrentTimestamp)
```

A value of a type with no SQL literal form is reported by `Validate`, and
panics in `Build`, naming the table and column rather than being dropped in
silence.

## Keys and constraints

### Primary keys

There is no `ID()` shortcut:

```go
t.BigIncrements("id").Primary()
t.PrimaryKey("circle_id", "user_id")   // composite
```

On MySQL an auto-increment column must be a key; if you do not mark one, the
builder marks it rather than emitting DDL the server would refuse.

### Checks

`CHECK` is standard SQL — PostgreSQL, MySQL 8.0.16+ and SQLite all enforce
one. Name it, because the name is what an error reports and what a later
migration drops:

```go
t.Check("posts_kind_valid", "kind IN ('status','text','link')")
t.Check("posts_counts_sane", "reaction_count >= 0 AND comment_count >= 0")
```

### Indexes

```go
t.Index("user_id")
t.Index("user_id", "created_at").Name("posts_author_idx")
t.Index("user_id", "created_at").Desc("created_at").Name("posts_recent_idx")
t.Index("email").Unique().Name("users_email_key")
t.Index("email").Unique().Where("deleted_at IS NULL").Name("users_live_email_key")
```

`Name` matters more than it looks. The derived name is right for a new table
and wrong for one that exists: every index in a migration that has already run
has a name in somebody's database, and a builder that cannot reproduce it
forces the whole statement to be hand-written.

`Desc` names a column already in the index. Writing the direction into the
column list — `Index("created_at DESC")` — happens to work today only because
the builder joins its arguments verbatim, and stops working the moment it
quotes an identifier. A column named in `Desc` that the index does not carry
is an error from `Validate`.

`Where` makes it partial. **MySQL has no partial index in any form**, so
`Validate` refuses one there rather than dropping the predicate — an index
silently covering every row instead of a tenth gives correct results and a
query plan nobody can account for.

### Unique: index or constraint

These are different objects, and on PostgreSQL the difference is visible:

```go
t.Index("email").Unique().Name("users_email_key")   // CREATE UNIQUE INDEX
t.UniqueKey("email").Name("users_email_key")        // CONSTRAINT ... UNIQUE
```

`pg_dump` renders the constraint as `ALTER TABLE … ADD CONSTRAINT` and the
index as `CREATE UNIQUE INDEX`, so a schema being *matched* rather than merely
created has to produce the right one. MySQL implements the constraint with an
index, which is what makes the distinction easy to lose.

Reach for `UniqueKey` when the uniqueness is part of what the table is, and
for `Index(...).Unique()` when it is a lookup index that happens to be unique
— or one you will want a `Where` on.

### Foreign keys

```go
t.Foreign("user_id").References("id").On("users").OnDelete("CASCADE")

t.Foreign("user_id", "skill_id").
    Name("endorsements_claimed_skill").
    References("user_id", "skill_id").
    On("user_skills").
    OnDelete("CASCADE")
```

`References` takes a column list, so a composite key is one constraint over
the pair. Two single-column keys are not the same thing: neither of them
refuses a row naming a user and a skill that are never paired, which is the
invariant a pivot exists to hold. `Validate` refuses a key referencing a
different number of columns than it carries.

`Name` is opt-in. Without it the key is anonymous and the database derives a
name of its own. Set it when you need a stable name to reference later — to
drop the constraint by name in a later migration, or to match a name that
already exists.

## Altering a table

```go
migration.Alter("posts", func(t *migration.Table) {
    t.Int("view_count").Default(0)
    t.String("slug", 160).Nullable().Change()
    t.DropColumn("legacy_body")
    t.RenameColumn("body", "body_source")

    t.DropIndex("posts_author_idx")
    t.DropUniqueKey("posts_slug_unique")
    t.DropForeignKey("posts_author_fk")
    t.DropCheck("posts_kind_valid").IfExists()
    t.DropPrimaryKey()
})
```

The three dialects disagree more here than anywhere else:

| | PostgreSQL | MySQL | SQLite |
|---|---|---|---|
| drop a primary key | `DROP CONSTRAINT <table>_pkey` | `DROP PRIMARY KEY` | refused |
| drop a unique key | `DROP CONSTRAINT <name>` | `DROP INDEX <name>` | refused |
| drop a foreign key | `DROP CONSTRAINT <name>` | `DROP FOREIGN KEY <name>` | refused |
| drop an index | `DROP INDEX <name>`, its own statement | `ALTER TABLE … DROP INDEX` | `DROP INDEX <name>` |
| change a column | `ALTER COLUMN … TYPE`, plus `SET`/`DROP NOT NULL` | `MODIFY COLUMN …` | refused |
| several actions | one statement, comma-separated | one statement, comma-separated | **one statement each** |

`DropPrimaryKey()` takes no argument — PostgreSQL needs the name and derives
`<table>_pkey`, MySQL does not use one.

`IfExists` is for a down migration, which should not fail because the thing it
removes is already gone. **MySQL has no `IF EXISTS`** on any of these — not on
an `ALTER TABLE` drop action and not on `DROP INDEX` — so `Validate` refuses
it there rather than quietly dropping the tolerance.

SQLite's `ALTER TABLE` does `RENAME TO`, `RENAME COLUMN`, `ADD COLUMN` and
`DROP COLUMN`, and nothing else. Changing a column or a constraint there means
rebuilding the table, which is a migration you write yourself — so the builder
refuses rather than emitting something SQLite will reject.

## What still needs raw SQL

`tx.Exec` is the sanctioned escape hatch. These have no portable form and are
not pretending to:

- **Expression and operator-class indexes**, `tsvector`, `GIN`/`GiST` —
  PostgreSQL only.
- **Functions and triggers.** `plpgsql` is PostgreSQL, MySQL's trigger syntax
  differs, and SQLite has no stored functions at all.
- **Generated columns**, whose expression is dialect-specific.
- **`COMMENT ON`.** PostgreSQL and MySQL spell it differently; SQLite has none.
- **Altering a column or a constraint on SQLite**, as above.

```go
// A generated column and the GIN index over it: PostgreSQL, and only
// PostgreSQL, so there is nothing for the builder to render per dialect.
for _, statement := range []string{
    `ALTER TABLE posts ADD COLUMN search_vector tsvector
       GENERATED ALWAYS AS (to_tsvector('english', coalesce(body, ''))) STORED`,
    `CREATE INDEX posts_search_idx ON posts USING GIN (search_vector)`,
} {
    if _, err := tx.Exec(statement); err != nil {
        return err
    }
}
```

## Next steps

- [Column types reference](/docs/migrations/column-types) — every column method
  in one table.
- [Writing migrations](/docs/migrations/writing-migrations) — putting the
  builder into a versioned file.
- [Raw SQL & DDL](/docs/database/raw-sql-and-ddl) — for what the builder cannot
  express.
