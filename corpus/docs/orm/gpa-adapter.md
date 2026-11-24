---
title: GPA Adapter
type: docs
prev: docs/orm/field-descriptors
next: docs/database/
weight: 55
sidebar:
  open: true
---

## Running GPA code on the ORM

[`gpaorm`](https://github.com/lemmego/gpaorm) implements the [GPA](/docs/database/) persistence contracts on top of the ORM, so an application written against `gpa.Repository[T]` runs on it without changes.

The dependency direction is one way — `gpaorm` imports the ORM, never the reverse. The ORM itself has no knowledge of GPA.

## Setting it up

Ask the connector for it:

```go
&ormconnector.Provider{UseGPA: true}
```

That registers a `gpaorm` provider as the GPA default, so the usual resolution works:

```go
func SQLRepo[T any](instanceName ...string) gpa.MigratableRepository[T] {
    provider := gpa.MustGet[*gpaorm.Provider](instanceName...)
    return provider.Repository[T]()
}

users := SQLRepo[models.User]()
```

Scaffolding with `--gpa` wires this for you.

The provider registers under the name `LemmegoORM`.

## What translates

| GPA | ORM |
|---|---|
| `Where`, `WhereIn`, `WhereLike`, `WhereNull` | predicates |
| `And`, `Or`, `Not` composites | `orm.And` / `orm.Or` / `orm.Not` |
| `OrderBy`, `Limit`, `Offset`, `Distinct` | the same |
| `Fields` | `Select` |
| `Join`, `LeftJoin` | `JoinTable` / `LeftJoinTable` |
| `GroupBy`, `Having` | the same |
| `Preload` | `With` — one query per relationship level |

Field names may be given as either struct field names or column names; both resolve.

## What it refuses

Anything the adapter cannot express returns `gpa.ErrorTypeUnsupported` rather than being quietly dropped:

- **Row locking.** A lock that is silently ignored leaves the caller believing they hold one.
- **Subqueries.**
- **`CreateTable`, `MigrateTable`, `GetTableInfo`, `GetMigrationStatus`, `Migrate`.** The ORM does not generate DDL from struct definitions; schema belongs to the [migration module](/docs/migrations/). `DropTable`, `CreateIndex` and `DropIndex` are implemented, since they need no schema inference.

## Transactions

GPA models a transaction as a per-entity repository, while the ORM's is entity-agnostic. Many `Transaction[T]` values share one `*orm.Tx`, so a unit of work still spans several models:

```go
err := users.Transaction(ctx, func(tx gpa.Transaction[User]) error {
    user := &User{Email: "ada@example.com"}
    if err := tx.Create(ctx, user); err != nil {
        return err
    }
    posts := gpaorm.TxFor[Post](tx.(*gpaorm.Transaction[User]))
    return posts.Create(ctx, &Post{UserID: user.ID})
})
```

`Commit` is a no-op — the transaction commits when the callback returns nil. `Rollback` marks it for rollback and is not reported as an error, since it is what the caller asked for.

## Errors

| ORM | GPA |
|---|---|
| `ErrNotFound` | `ErrorTypeNotFound` |
| `*ConstraintError` (unique) | `ErrorTypeDuplicate` |
| `*ConstraintError` (other) | `ErrorTypeConstraint` |
| `ErrUnsupported` | `ErrorTypeUnsupported` |
| `ErrUnsafeMutation` | `ErrorTypeInvalidArgument` |
| anything else | `ErrorTypeDatabase` |

The original error is kept as the cause, so `errors.As` still reaches `*orm.ConstraintError` and the driver error beneath it.

## Hooks

The ORM's create, update and delete hooks are signature-identical to GPA's, so an entity implementing one implements the other and the ORM runs them. The adapter adds only what GPA declares and the ORM does not: `Validate` before create and update, and `AfterFind` on results.

One behavioural difference from `gpagorm`: an after-hook error is **returned** here, rather than logged and swallowed.

## Next steps

- [GPA overview](/docs/database/) — the abstraction your code is written
  against.
- [Generic repository](/docs/database/generic-repository) — the `Repository[T]`
  the adapter satisfies.
