---
title: Raw SQL & DDL
type: docs
prev: docs/database/transactions
next: docs/database/providers/gorm
weight: 67
sidebar:
  open: true
---

SQL providers (GORM, Bun) support raw SQL execution and DDL operations directly through the repository.

## Raw Queries

### Query with Raw SQL

```go
users, err := userRepo.RawQuery(ctx, "SELECT * FROM users WHERE age > ?", []interface{}{18})
```

### Execute Raw SQL

```go
result, err := userRepo.RawExec(ctx, "UPDATE users SET status = ? WHERE id = ?", []interface{}{"active", 1})
affected := result.RowsAffected()
lastID := result.LastInsertId()
```

### Find by SQL (SQL Repository)

```go
users, err := userRepo.FindBySQL(ctx, "SELECT * FROM users WHERE age > ? AND active = ?", []interface{}{18, true})

// Exec SQL (returns result)
res, err := userRepo.ExecSQL(ctx, "DELETE FROM users WHERE status = ?", "inactive")
```

## DDL Operations

### Create Table

```go
err := userRepo.CreateTable(ctx)
```

This auto-creates the table based on the entity's struct definition, mapping Go types to SQL column types.

### Drop Table

```go
err := userRepo.DropTable(ctx)
```

### Index Management

```go
// Create index
err := userRepo.CreateIndex(ctx, []string{"email", "status"}, true) // unique index

// Drop index
err := userRepo.DropIndex(ctx, "idx_users_email_status")
```

### Migration

```go
// Auto-migrate (adds/modifies columns to match entity)
err := userRepo.MigrateTable(ctx)

// Check migration status
status, err := userRepo.GetMigrationStatus(ctx)

// Get table schema info
info, err := userRepo.GetTableInfo(ctx)
```

## Relationship Eager Loading

```go
// Find with related models
users, err := userRepo.FindWithRelations(ctx, []string{"Profile", "Posts"})

// Find by ID with relations
user, err := userRepo.FindByIDWithRelations(ctx, 1, []string{"Profile", "Posts.Comments"})
```

## Provider Comparison

| Feature | GORM | Bun | MongoDB |
|---------|------|-----|---------|
| RawQuery | Yes | Yes | Returns error |
| RawExec | Yes | Yes | Returns error |
| CreateTable | Yes (auto) | Yes | CreateCollection |
| DropTable | Yes | Yes | DropCollection |
| CreateIndex | Yes | Yes | Yes |
| AutoMigrate | Yes | No | N/A |
| Eager Loading | Yes | Yes | N/A | 

## Next steps

- [Schema builder](/docs/migrations/schema-builder) — the same DDL, versioned
  and reversible.
- [Query system](/docs/database/query-system) — the typed API to prefer where
  it reaches.
