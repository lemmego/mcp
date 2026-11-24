---
title: Bun Connector
type: docs
prev: docs/database/gormconnector
next: docs/database/libsql
weight: 73
sidebar:
  open: true
---

If you chose Bun as your SQL layer, this is the
[provider](/docs/core-concepts/provider-plugin-system) that opens the
connection at startup from your config files and makes it available to the
rest of your application — so you add one line to `bootstrap/providers.go`
instead of writing connection code.

It connects using [uptrace/bun](https://github.com/uptrace/bun) and optionally registers a GPA provider.

## Usage

```go
func LoadProviders() []app.Provider {
    return []app.Provider{
        &bunconnector.Provider{
            UseGPA: true,
        },
    }
}
```

## Configuration

Reads from your application configuration (`internal/configs/database.go`):

```go
config.Set("sql", config.M{
    "default": "postgres",
    "connections": config.M{
        "postgres": config.M{
            "host":     config.MustEnv("DB_HOST", "127.0.0.1"),
            "port":     config.MustEnv("DB_PORT", 5432),
            "database": config.MustEnv("DB_DATABASE", "lemmego"),
            "username": config.MustEnv("DB_USERNAME", "root"),
            "password": config.MustEnv("DB_PASSWORD", ""),
        },
    },
})
```

## Without GPA

```go
&bunconnector.Provider{
    UseGPA: false,
}
```

When GPA is disabled, the raw `*bun.DB` is registered as a service:

```go
db := app.Get[*bun.DB](a)
```

## Supported Drivers

- PostgreSQL (`pgdialect`)
- MySQL (`mysqldialect`)
- SQLite (`sqlitedialect`)

## Connection Pooling

```go
config.Set("sql", config.M{
    "connections": config.M{
        "postgres": config.M{
            "host":           "127.0.0.1",
            "max_open_conns": 25,
            "max_idle_conns": 10,
            "conn_max_lifetime": "5m",
        },
    },
})
```

## Differences from GORM Connector

| Feature | GORM Connector | Bun Connector |
|---------|---------------|---------------|
| Auto-Migration | Yes | No |
| Supported Drivers | PostgreSQL, MySQL, SQLite, SQL Server | PostgreSQL, MySQL, SQLite |
| Dialects | GORM | Bun native |
| GPA Integration | Yes (gpagorm) | Yes (gpabun) |

## Next steps

- [The connection seam](/docs/database/connection-seam/) — what the connector
  hands the rest of the framework.
- [Bun provider](/docs/database/providers/bun) — the repositories this
  connection backs.
