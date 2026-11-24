---
title: Queries
type: docs
prev: docs/orm/
next: docs/orm/writes
weight: 51
sidebar:
  open: true
---

Reading rows back out is most of what you do with a database, so this is the
page you will return to.

Every query starts the same way — name the model, describe what you want, then
ask for it:

```go
users, err := db.Model[User]().Where(orm.Eq("active", true)).All(ctx)
```

Nothing touches the database until that last call. `Model` and `Where` only
build up a description; `All` is what sends the SQL and waits for rows. That
matters because it is what lets you assemble a query in pieces, across
several `if` statements, and run it once at the end.

## Building a query

`Model[T]` starts a query. Every builder method returns a copy, so a partially built query can be shared and branched without aliasing:

```go
base := db.Model[User]().Where(orm.Eq("active", true))

recent := base.OrderBy(orm.Desc("created_at")).Limit(10)
count, _ := base.Count(ctx)     // base is unchanged
```

| Category | Methods |
|---|---|
| Filter | `Where`, `Having`, `GroupBy`, `Distinct`, `Scope` |
| Shape | `Select`, `OrderBy`, `Limit`, `Offset`, `Table` |
| Join | `Join[U]`, `LeftJoin[U]`, `JoinTable`, `LeftJoinTable` |
| Read | `All`, `First`, `Find`, `Count`, `Exists`, `Paginate` |
| Typed read | `Pluck[V]`, `As[R]`, `Sum[V]`, `Avg[V]`, `Min[V]`, `Max[V]` |
| Relations | `With`, and `db.Relation[P, C]` |

`SELECT` names its columns explicitly rather than using `*`, so a column present in the table but absent from the struct is simply not read.

## Predicates

There are two ways to build a predicate, and they produce byte-identical SQL.

### String constructors

No code generation required:

```go
orm.Eq("active", true)        orm.Ne("role", "admin")
orm.Gt("age", 18)             orm.Gte("age", 18)
orm.Lt("age", 65)             orm.Lte("age", 65)
orm.Like("email", "%@ex.com") orm.NotLike("email", "%@spam.com")
orm.In("id", 1, 2, 3)         orm.NotIn("id", 4, 5)
orm.IsNull("deleted_at")      orm.IsNotNull("verified_at")
orm.Between("age", 18, 65)
```

Combine them with `orm.And`, `orm.Or` and `orm.Not`:

```go
db.Model[User]().Where(
    orm.Or(orm.Eq("role", "admin"), orm.Eq("role", "owner")).
        And(orm.Eq("active", true)),
).All(ctx)
```

`orm.RawPredicate` takes literal SQL when the builder cannot express something. Its `?` placeholders are renumbered per dialect, so it composes with built predicates even on PostgreSQL:

```go
.Where(orm.RawPredicate(`"score" > (SELECT AVG(score) FROM "users")`))
```

### Typed field descriptors

The same query, checked at compile time:

```go
db.Model[User]().
    Where(UserFields.Active.Eq(true)).
    Where(UserFields.Age.Gte(18)).
    OrderBy(UserFields.CreatedAt.Desc()).
    All(ctx)
```

A descriptor knows its column *and* its Go type, so comparing a `bool` column against a string will not compile. See [Field descriptors](/docs/orm/field-descriptors) for generating them.

Both forms produce an `orm.Predicate`, so there is one `Where` signature and one SQL compiler. Mix them freely.

## Reading

```go
users, err := db.Model[User]().All(ctx)              // []User
user,  err := db.Model[User]().First(ctx)            // *User, or orm.ErrNotFound
user,  err := db.Model[User]().Find(ctx, 42)         // by primary key
count, err := db.Model[User]().Count(ctx)
exists, err := db.Model[User]().Exists(ctx)
```

`Find` takes one value per key column, so a composite key is addressed as `Find(ctx, studentID, courseID)`.

## Results that change type

`Query[T]` is not locked to returning `[]T`. These are generic methods, so the result type changes mid-chain:

```go
// A single column
emails, err := db.Model[User]().Pluck[string](ctx, "email_address")

// Aggregates
total, err := db.Model[Order]().Where(orm.Eq("paid", true)).Sum[int64](ctx, "cents")
newest, err := db.Model[User]().Max[time.Time](ctx, "created_at")

// Project into a DTO, keeping the source table and predicates
type UserEmail struct {
    Email string `orm:"column:email_address"`
}
rows, err := db.Model[User]().Where(orm.Eq("active", true)).As[UserEmail]().All(ctx)

// A join, read into a row type of its own
report, err := db.Model[User]().
    Join[Post](orm.RawPredicate(`"posts"."user_id" = "users"."id"`)).
    As[UserPostRow]().
    All(ctx)
```

An aggregate over no rows is `NULL`, and reads back as the zero value rather than failing.

## Pagination

```go
page, err := db.Model[Post]().
    Where(orm.Eq("published", true)).
    OrderBy(orm.Desc("created_at")).
    Paginate(ctx, 2, 20)

page.Items       // []Post
page.Total       // rows in the whole result set
page.TotalPages
page.HasNext()
page.HasPrev()
```

`Total` counts the whole result set, ignoring any limit or offset already on the query. Aggregates drop `ORDER BY`, which is meaningless for a count and which PostgreSQL rejects outright.

## Scopes

A scope names a filter so it can be reused and composed:

```go
func Published() orm.Scope[Post] {
    return func(q *orm.Query[Post]) *orm.Query[Post] {
        return q.Where(orm.Eq("published", true))
    }
}

func InCategory(id uint64) orm.Scope[Post] {
    return func(q *orm.Query[Post]) *orm.Query[Post] {
        return q.Where(orm.Eq("category_id", id))
    }
}

posts, err := db.Model[Post]().Scope(Published(), InCategory(3)).All(ctx)
```

## Raw SQL

```go
type MonthlyReport struct {
    Month string `orm:"column:month"`
    Total int64  `orm:"column:total"`
}

rows, err := db.Raw[MonthlyReport](
    `SELECT strftime('%Y-%m', created_at) AS month, COUNT(*) AS total
     FROM posts GROUP BY month`,
).All(ctx)
```

Raw queries tolerate columns the destination does not declare — hand-written SQL routinely returns more than the struct models. Model queries do not, so a mistyped `Select` is reported rather than silently dropped.

## Next steps

- [Writes and transactions](/docs/orm/writes) — creating, updating and rolling
  back.
- [Relationships](/docs/orm/relationships) — loading the rows hanging off the
  ones you found.
- [Field descriptors](/docs/orm/field-descriptors) — replacing column strings
  with checked identifiers.
