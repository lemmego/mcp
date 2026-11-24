---
title: GORM Provider
type: docs
prev: docs/database/raw-sql-and-ddl
next: docs/database/providers/bun
weight: 68
sidebar:
  open: true
---

The GORM provider (`github.com/lemmego/gpagorm`) is the production-ready SQL provider. It wraps [GORM](https://gorm.io/) to provide the full GPA `Repository[T]` interface with migration support.

## Setup

```go
import (
    "github.com/lemmego/gpa"
    "github.com/lemmego/gpagorm"
)

config := gpa.Config{
    Driver:   "postgres",
    Host:     "localhost",
    Port:     5432,
    Username: "user",
    Password: "password",
    Database: "mydb",
}

provider, err := gpagorm.NewProvider(config)
```

### Supported Drivers

- PostgreSQL
- MySQL
- SQLite
- SQL Server / MSSQL

### Connection Pooling

GORM connection pooling is configured through the `Config`:

```go
config := gpa.Config{
    Driver:        "mysql",
    MaxOpenConns:  25,
    MaxIdleConns:  10,
    ConnMaxLifetime: 5 * time.Minute,
    Options: map[string]interface{}{
        "gorm": map[string]interface{}{
            "log_level": "info",
            "singular_table": true,
        },
    },
}
```

## Creating Repositories

```go
// From the registry (auto-resolves the "default" GORM provider)
userRepo := gpagorm.GetRepository[User]()

// From a specific provider instance
userRepo := gpagorm.GetRepository[User](provider)

// From a named registry instance
userRepo := gpagorm.GetRepository[User]("primary")
```

## Repository Type

The GORM repository implements `gpa.MigratableRepository[T]`, which includes:

- Full `Repository[T]` CRUD
- `SQLRepository[T]` (raw SQL, DDL, eager loading)
- `MigratableRepository[T]` (auto-migration, migration status, table info)

```go
// Pattern with typed wrapper
type UserRepository struct {
    gpa.MigratableRepository[User]
}

func NewUserRepository() *UserRepository {
    repo := gpagorm.GetRepository[User]()
    return &UserRepository{repo}
}
```

## Hooks

The GORM provider supports all GPA entity hooks:

```go
type User struct {
    ID        uint
    Name      string
    Email     string
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (u *User) Validate(ctx context.Context) error {
    if u.Email == "" {
        return gpa.NewError(gpa.ErrorTypeValidation, "email required")
    }
    return nil
}

func (u *User) BeforeCreate(ctx context.Context) error {
    u.Email = strings.ToLower(u.Email)
    u.CreatedAt = time.Now()
    u.UpdatedAt = time.Now()
    return nil
}

func (u *User) AfterFind(ctx context.Context) error {
    // Post-load processing
    return nil
}
```

Hook execution order: `Validate` → `BeforeCreate` → DB Insert → `AfterCreate`

## Features

| Feature | Support |
|---------|---------|
| Transactions + Savepoints | Yes |
| Raw SQL | Yes |
| Eager Loading | Yes (Preload, FindWithRelations) |
| Auto Migration | Yes (AutoMigrate, MigrateTable) |
| Index Management | Yes (CreateIndex, DropIndex) |
| Batch Operations | Yes (CreateBatch with size 100) |
| SQL Injection Protection | Yes (regex-based field validation) |
| Error Mapping | Yes (GORM errors → GPA typed errors) |

## Next steps

- [GORM connector](/docs/database/gormconnector) — wiring this provider into a
  project's boot.
- [Generic repository](/docs/database/generic-repository) — the API you get
  once it is registered.
- [Entity hooks](/docs/database/entity-hooks) — which hooks this provider
  fires.
