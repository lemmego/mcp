---
title: GORM Connector
type: docs
prev: docs/database/providers/redis
next: docs/database/bunconnector
weight: 72
sidebar:
  open: true
---

Choosing GORM as your SQL layer leaves you one job: opening the connection at
startup, with the settings from your config files, and making it available to
the rest of your application. The `gormconnector` package is the
[provider](/docs/core-concepts/provider-plugin-system) that does exactly that,
so you add a line to `bootstrap/providers.go` rather than writing connection
code yourself.

It connects using GORM and optionally registers a GPA provider. It's used in the generated project's bootstrap.

## Usage in bootstrap/providers.go

```go
func LoadProviders() []app.Provider {
    return []app.Provider{
        &gormconnector.Provider{
            UseGPA: true,
        },
    }
}
```

## Configuration

The connector reads from your application configuration (set in `internal/configs/database.go`):

```go
// config.Set("sql", config.M{
//     "default": "sqlite",
//     "connections": config.M{
//         "sqlite": config.M{
//             "database": "storage/database.sqlite",
//         },
//     },
// })
```

### WithGPAConfig

```go
&gormconnector.Provider{
    UseGPA: true,
    WithGPAConfig: func(cfg *gpa.Config) {
        cfg.MaxOpenConns = 50
    },
}
```

### WithGORMConfig

```go
&gormconnector.Provider{
    WithGORMConfig: func(cfg *gorm.Config) {
        cfg.SkipDefaultTransaction = true
        cfg.PrepareStmt = true
    },
}
```

## Without GPA

```go
&gormconnector.Provider{
    UseGPA: false, // default
}
```

When GPA is disabled, `gormconnector` registers the raw `*gorm.DB` as a service:

```go
db := app.Get[*gorm.DB](a)
```

## Supported Drivers

- PostgreSQL / Postgres
- MySQL
- SQLite / SQLite3 (pure-Go via `github.com/glebarez/go-sqlite`)
- SQL Server / MSSQL

## DSN Building

The connector builds database-specific DSNs:

- **PostgreSQL**: `host=... port=... user=... password=... dbname=...`
- **MySQL**: `user:password@tcp(host:port)/dbname?charset=utf8mb4`
- **SQLite**: `file:storage/database.sqlite`
- **SQL Server**: `sqlserver://user:password@host:port?database=dbname`

## Repository Pattern

When GPA is enabled, the generated project includes a `repos/` package that wraps repository creation:

```go
// internal/repos/repo.go
func SQLRepo[T any](instanceName ...string) gpa.MigratableRepository[T] {
    return gpagorm.GetRepository[T](instanceName...)
}

// internal/repos/user_repo.go
type UserRepository struct {
    gpa.MigratableRepository[models.User]
}

func User(instanceName ...string) *UserRepository {
    repo := SQLRepo[models.User](instanceName...)
    return &UserRepository{repo}
}
```

Usage in handlers:

```go
user := repos.User().MustFindByID(c.RequestContext(), 1)
users, _ := repos.User().FindAll(c.RequestContext())
repos.User().Create(c.RequestContext(), &models.User{Name: "Jane"})
```

## Next steps

- [The connection seam](/docs/database/connection-seam/) — what the connector
  hands the rest of the framework.
- [Running migrations](/docs/migrations/running-migrations) — pointing the
  migrator at the same database.
- [GORM provider](/docs/database/providers/gorm) — the repositories this
  connection backs.
