---
title: Transactions
type: docs
prev: docs/database/entity-hooks
next: docs/database/raw-sql-and-ddl
weight: 66
sidebar:
  open: true
---

GPA supports transactions through the `Transaction[T]` interface, which extends `Repository[T]` with commit, rollback, and savepoint support.

## Basic Transactions

```go
err := userRepo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
    // All operations within this function are atomic
    if err := tx.Create(ctx, &User{Name: "Alice"}); err != nil {
        return err  // Rolls back the transaction
    }
    if err := tx.UpdatePartial(ctx, 1, map[string]interface{}{
        "name": "Bob",
    }); err != nil {
        return err  // Rolls back the transaction
    }
    return nil  // Commits the transaction
})
```

If the function returns an error, the transaction is rolled back. If it returns `nil`, the transaction is committed.

## Nested Transactions and Savepoints

SQL providers support savepoints for nested transactions:

```go
err := userRepo.Transaction(ctx, func(tx gpa.Transaction[User]) error {
    // Outer transaction
    tx.Create(ctx, &User{Name: "Alice"})

    // Create a savepoint
    tx.SetSavepoint("before_update")

    // This can be rolled back independently
    tx.UpdatePartial(ctx, 1, map[string]interface{}{"name": "Bob"})

    // Rollback to savepoint (undoes the update but keeps the create)
    if err := tx.RollbackToSavepoint("before_update"); err != nil {
        return err
    }

    return nil  // Commits only the create
})
```

## Transaction Interface

```go
type Transaction[T any] interface {
    Repository[T]
    Commit() error
    Rollback() error
    SetSavepoint(name string) error
    RollbackToSavepoint(name string) error
}
```

## Provider Transaction Support

| Feature | GORM | Bun | MongoDB | Redis |
|---------|------|-----|---------|-------|
| Commit/Rollback | Yes | Yes | Yes | No* |
| Savepoints | Yes | No | No | No |
| Isolation Levels | Yes (ReadCommitted, RepeatableRead, Serializable) | Yes | Yes | N/A |

*Redis uses its own multi/exec model for atomic operations.

## Next steps

- [Raw SQL & DDL](/docs/database/raw-sql-and-ddl) — statements the repository
  API does not cover.
- [Entity hooks](/docs/database/entity-hooks) — what runs inside your
  transaction.
- [Writing migrations](/docs/migrations/writing-migrations) — the other place a
  transaction wraps your work.
