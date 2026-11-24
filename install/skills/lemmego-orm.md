## When to use this skill

When writing a query, a relationship or an association write with the Lemmego
ORM. The ORM is standalone and built directly on `database/sql`; `gpaorm` is
the adapter that exposes it through GPA.

## Query entry points are generic methods on the handle

Go allows methods to declare their own type parameters, so the entry points are
receiver-oriented:

    users, err := db.Model[User]().
        Where(orm.Eq("active", true)).
        With("Posts.Comments").
        OrderBy(orm.Desc("created_at")).
        Limit(20).
        All(ctx)

`db.Model[T]()`, `tx.Model[T]()`, `db.Raw[T]()`, `tx.Raw[T]()`.

Generic methods carry the cases where the *result* type changes:

    emails, err := db.Model[User]().Pluck[string](ctx, "email_address")
    total, err  := db.Model[Order]().Sum[int64](ctx, "cents")
    rows, err   := db.Model[User]().Join[Post](on).As[UserPostRow]().All(ctx)

**The consequence to know:** a generic method can never satisfy an interface.
`DB` and `Tx` are therefore concrete types with duplicated entry points and
**no shared `Querier` interface**. Do not try to write a function that accepts
"either a DB or a Tx" through an interface — it cannot exist. Pass the concrete
type, or take `*orm.Tx` and have the caller open the transaction.

Every builder method returns a copy, so a partially built query can be shared
and branched without aliasing.

## Relationships are declared in tags, naming the field

    type User struct {
        ID    int    `orm:"primaryKey;autoIncrement"`
        Posts []Post `orm:"hasMany:UserID"`
    }

`hasOne`, `hasMany`, `belongsTo`, `many2many`. `With` eager-loads at one query
per relation level using batched `WHERE fk IN (...)` lookups rather than joins;
a many-to-many costs two per level.

**`With` appends only the soft-delete predicate** — no `ORDER BY`, no `LIMIT`,
no pivot columns. If you need an ordered or limited relation, load it yourself.

## Association writes take the handle as the receiver

    db.Relation[User, Post](&user, "Posts").Add(ctx, &post)
    db.Relation[Article, Tag](&article, "Tags").Sync(ctx, &goTag, &sqlTag)

Models stay plain structs and never hold a connection, because a model holding
one cannot be safely serialised, copied between goroutines, or built in a test.

## Soft deletes and timestamps

A **nullable** `DeletedAt` (`*time.Time` or `sql.NullTime`), or any nullable
timestamp tagged `orm:"softDelete"`, makes a model soft-deletable. Reads,
counts, pagination and bulk mutations exclude trashed rows unless
`WithTrashed()` or `OnlyTrashed()`.

**A bare `time.Time` is rejected at metadata time**, deliberately: its zero
value writes a real timestamp, so `IS NULL` would match nothing and silently
hide the whole table.

`CreatedAt` and `UpdatedAt` of a time type are maintained automatically, in
UTC. `CreatedAt` is only filled when unset, so explicit and hook-assigned
values survive. UTC matters because a naive column stores no offset.

## Three things the ORM does not do

Do not spend time looking for these; write raw SQL and move on:

- **Keyset pagination.** `Paginate` is offset-based, so a cursor is
  hand-rolled raw SQL at the call site.
- **Atomic increment.** `UpdateAll` binds every value as a placeholder, so
  `col = col + 1` is inexpressible.
- **Ordered eager loading** — see `With` above.

## Naming

Table names use snake_case plus pluralisation, matching `/cli` and
`/migration`, so `Category` resolves to `categories` and agrees with a
generated migration. `TableName() string` overrides. An `orm:` tag is preferred
with `db:"..."` honoured as a fallback, so CLI-generated models map without
edits.

## Field descriptors

    go run github.com/lemmego/orm/cmd/ormgen ./internal/models

That generates typed field descriptors, so `UserFields.Active.Eq(true)`
replaces `orm.Eq("active", true)` and a renamed column is a compile error. Both
produce a `Predicate`, so there is one `Where` signature.

## Check first

`search-docs` section `orm`, and `module-api` on `orm` for signatures.
`orm/ARCHITECTURE.md` and the README are the authoritative specs.
