---
title: Query System
type: docs
prev: docs/database/generic-repository
next: docs/database/entity-hooks
weight: 64
sidebar:
  open: true
---

GPA provides two ways to build queries: option-based (functional) and builder-based (fluent). Both produce the same `Query` object and support the same set of conditions.

## Conditions

Conditions are the building blocks of queries:

```go
// Basic conditions
gpa.Where("age", gpa.OpGreaterThan, 18)
gpa.Where("status", gpa.OpEqual, "active")
gpa.Where("name", gpa.OpLike, "%john%")

// Available operators
gpa.OpEqual              // =
gpa.OpNotEqual           // !=
gpa.OpGreaterThan        // >
gpa.OpGreaterThanOrEqual // >=
gpa.OpLessThan           // <
gpa.OpLessThanOrEqual    // <=
gpa.OpLike               // LIKE
gpa.OpNotLike            // NOT LIKE
gpa.OpIn                 // IN
gpa.OpNotIn              // NOT IN
gpa.OpBetween            // BETWEEN
gpa.OpIsNull             // IS NULL
gpa.OpIsNotNull          // IS NOT NULL
```

### Convenience Builders

```go
gpa.Where("age", gpa.OpGreaterThan, 18)
gpa.WhereIn("status", "active", "pending")
gpa.WhereLike("name", "%john%")
gpa.WhereNull("deleted_at")
gpa.WhereNotNull("email")
```

### Composite Conditions

```go
// AND condition
gpa.And(
    gpa.Where("age", gpa.OpGreaterThan, 18),
    gpa.Where("status", gpa.OpEqual, "active"),
)

// OR condition
gpa.Or(
    gpa.Where("role", gpa.OpEqual, "admin"),
    gpa.Where("role", gpa.OpEqual, "moderator"),
)
```

## Query Options (Functional API)

Pass options directly to repository methods:

```go
users, err := userRepo.Query(ctx,
    gpa.Where("age", gpa.OpGreaterThan, 18),
    gpa.Where("status", gpa.OpEqual, "active"),
    gpa.OrderBy("name", gpa.OrderAsc),
    gpa.Limit(10),
    gpa.Offset(20),
    gpa.Fields("id", "name", "email"),
    gpa.Preload("Profile", "Posts"),
    gpa.Distinct(),
)
```

### Sort and Pagination

```go
gpa.OrderBy("created_at", gpa.OrderDesc)
gpa.OrderBy("name", gpa.OrderAsc)
gpa.Limit(10)
gpa.Offset(20)
```

### Field Selection

```go
gpa.Fields("id", "name", "email")
// Same as:
gpa.Select("id", "name", "email")
```

### Joins

```go
// Convenience helpers take the table and a join condition
gpa.InnerJoin("orders", "orders.user_id = users.id")
gpa.LeftJoin("comments", "comments.user_id = users.id")

// gpa.Join takes the join type explicitly, plus an optional alias.
// JoinInner, JoinLeft, JoinRight and JoinFull are the available types.
gpa.Join(gpa.JoinRight, "profiles", "profiles.user_id = users.id")
gpa.Join(gpa.JoinFull, "audits", "audits.user_id = users.id", "a")
```

### Grouping and Having

```go
gpa.GroupBy("status")
gpa.Having("count(*)", gpa.OpGreaterThan, 5)
```

### Locking

```go
gpa.Lock(gpa.LockForUpdate)
gpa.Lock(gpa.LockShared)
```

### Preloading Relationships

```go
gpa.Preload("Profile")
gpa.Preload("Posts.Comments")  // Nested eager loading
```

## Query Builder (Fluent API)

The fluent `QueryBuilder` provides a chainable alternative:

```go
qb := gpa.NewQueryBuilder[User]()

qb.Where("age", gpa.OpGreaterThan, 18).
   Where("active", gpa.OpEqual, true).
   OrderBy("name", gpa.OrderAsc).
   Limit(10).
   Offset(0).
   Preload("Profile")

// Execute
users, err := qb.Execute(ctx, userRepo)
user, err := qb.ExecuteOne(ctx, userRepo)
count, err := qb.Count(ctx, userRepo)
```

### Full Builder API

```go
qb := gpa.NewQueryBuilder[T]()

qb.Where(field, operator, value)
qb.WhereCondition(condition)          // a Condition built elsewhere
qb.OrderBy(field, direction)
qb.Limit(n)
qb.Offset(n)
qb.Select(fields...)
qb.Join(gpa.JoinInner, table, condition)
qb.GroupBy(fields...)
qb.Having(field, operator, value)
qb.Distinct()
qb.Lock(gpa.LockForUpdate)
qb.Preload(relations...)

// Build the Query object without executing
query := qb.Build()
```

There is one `Join`, taking the join type as its first argument —
`gpa.JoinInner`, `gpa.JoinLeft`, `gpa.JoinRight` or `gpa.JoinFull`. The
`gpa.InnerJoin` and `gpa.LeftJoin` helpers shown earlier are `QueryOption`s for the
option-style API, not builder methods.

## Subqueries

A subquery is a `*Query`, not a SQL string: it is built with the same builder
and passed in, so the same validation and placeholder handling applies.

```go
highValue := gpa.NewQueryBuilder[Order]().
    Select("user_id").
    Where("total", gpa.OpGreaterThan, 100).
    Build()

gpa.InSubQuery("id", highValue)
gpa.ExistsSubQuery(highValue)
gpa.CorrelatedSubQuery("score", gpa.OpGreaterThan, highValue, "product_id")
```

`CorrelatedSubQuery` takes four arguments: the outer field, the operator, the
subquery, and the field on the inner query that correlates back to the outer
one.

## Next steps

- [Transactions](/docs/database/transactions) — running a group of queries
  atomically.
- [Raw SQL & DDL](/docs/database/raw-sql-and-ddl) — for the queries this cannot
  express.
- [Generic repository](/docs/database/generic-repository) — the repository
  methods these options feed.
