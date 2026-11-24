---
title: Bun Provider
type: docs
prev: docs/database/providers/gorm
next: docs/database/providers/mongodb
weight: 69
sidebar:
  open: true
---

The Bun provider (`github.com/lemmego/gpabun`) wraps [uptrace/bun](https://github.com/uptrace/bun) to provide GPA-compatible SQL access. It's a lighter alternative to GORM with a focus on SQL-first development.

## Setup

```go
import (
    "github.com/lemmego/gpa"
    "github.com/lemmego/gpabun"
)

config := gpa.Config{
    Driver:   "postgres",
    Host:     "localhost",
    Port:     5432,
    Database: "mydb",
    Username: "user",
    Password: "password",
}

provider, err := gpabun.NewProvider(config)
```

### Supported Drivers

- PostgreSQL (via `pgdialect`)
- MySQL (via `mysqldialect`)
- SQLite (via `sqlitedialect`)

## Creating Repositories

```go
// From a specific provider
repo := gpabun.NewRepository[User](provider)

// From the registry
repo := gpabun.GetRepository[User]()
```

## Repository Type

The Bun repository implements `gpa.SQLRepository[T]`:

- Full `Repository[T]` CRUD
- Raw SQL (`FindBySQL`, `ExecSQL`)
- Eager loading (`FindWithRelations`, `FindByIDWithRelations`)
- DDL (`CreateTable`, `DropTable`, `CreateIndex`, `DropIndex`)

## Differences from GORM

| Feature | GORM Provider | Bun Provider |
|---------|---------------|--------------|
| Migration | Yes (AutoMigrate) | No (stub) |
| Savepoints | Yes | No |
| Validation Hook | Yes | No |
| Features | Transactions, Indexing, Agg, RawSQL | Transactions, Indexing, FullText, Joins, SubQueries |

## Hooks

The Bun provider supports Before/After Create/Update/Delete/Find hooks but does not support `ValidationHook`:

```go
func (u *User) BeforeCreate(ctx context.Context) error {
    u.CreatedAt = time.Now()
    return nil
}
```

## Use Cases

- Projects that prefer Bun's SQL-first approach over GORM's ORM-heavy pattern
- Applications needing full-text search or advanced subquery support
- Lightweight deployments where Bun's smaller footprint is beneficial

## Next steps

- [Bun connector](/docs/database/bunconnector) — wiring this provider into a
  project's boot.
- [GORM provider](/docs/database/providers/gorm) — the more complete SQL
  provider today.
