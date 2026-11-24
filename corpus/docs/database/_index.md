---
title: GPA Overview
type: docs
prev: docs/orm/gpa-adapter
next: docs/database/provider-registry
weight: 60
sidebar:
  open: true
---

{{< callout type="info" >}}
**If you are building an ordinary SQL application, you can skip this
section.** [The ORM](/docs/orm/) is what `lemmego new` sets up, and it is the
shorter path. Come back when one of the reasons below applies to you.
{{< /callout >}}

Different databases are not alike. A SQL database has tables and joins; Redis
is keys and values with no query language to speak of; MongoDB stores
documents. Normally that means your code is written against whichever one you
picked, and swapping it later means rewriting the parts that touch it.

**GPA** (Go Persistence API) puts one set of operations in front of all of
them. You ask a `Repository[User]` to find, create or delete, and a
**provider** underneath translates that into GORM calls, Bun calls, Redis
commands or Mongo queries:

```go
user, err := userRepo.FindByID(ctx, 42)   // same line, whichever store
```

Reach for it when you want a backend the ORM does not cover — GORM, Bun,
MongoDB or Redis — or when one application talks to more than one store at
once, such as Postgres for your records and Redis for a leaderboard.

### Key features

- **Compile-time type safety** — Generic `Repository[T]` returns `*T` and `[]*T` directly, no `interface{}` casting
- **Unified API** — Same CRUD operations across SQL, MongoDB, Redis, and more
- **Plugin architecture** — Swap database backends without application code changes
- **No runtime reflection** — Go generics provide type safety without runtime overhead
- **Entity hooks** — Lifecycle callbacks for validation, timestamps, and business logic

### Architecture

```
Application Code
      |
      v
+----------------------------+
|  Repository[T]             |  <-- Primary API (generic CRUD)
|  Transaction[T]            |
|  QueryBuilder[T]           |
+----------------------------+
      |
      v
+----------------------------+
|  ProviderRegistry          |  <-- Singleton registry (type-safe)
+----------------------------+
      |
      v
+----------------------------+
|  Provider (interface)      |  <-- Base interface
+----------------------------+
      |
      +-- SQLProvider ---------+--> GORM, Bun
      |                        +--> SQLRepository[T], MigratableRepository[T]
      |
      +-- KeyValueProvider ---+--> Redis
      |                        +--> Basic/Batch/TTL/Increment/Pattern KV repos
      |
      +-- DocumentProvider ---+--> MongoDB
      |                        +--> DocumentRepository[T] (aggregation, geo, text)
      |
      +-- GraphProvider ------+--> Neo4j (future)
      |
      +-- WideColumnProvider -+--> Cassandra (future)
```

### Available Providers

| Provider | Package | Type | Status |
|----------|---------|------|--------|
| GORM | `github.com/lemmego/gpagorm` | SQL | Production-ready |
| Bun | `github.com/lemmego/gpabun` | SQL | In development |
| MongoDB | `github.com/lemmego/gpamongo` | Document | In development |
| Redis | `github.com/lemmego/gparedis` | Key-Value | Production-ready |

### Quick Example

```go
import (
    "github.com/lemmego/gpa"
    "github.com/lemmego/gpagorm"
)

// Create a GORM provider
provider, _ := gpagorm.NewProvider(config)

// Register with the global registry
gpa.Register[*gpagorm.Provider]("primary", provider)

// Create a type-safe repository
userRepo := gpagorm.GetRepository[User]()

// Type-safe CRUD
user, _ := userRepo.FindByID(ctx, 1)   // Returns *User
users, _ := userRepo.FindAll(ctx)       // Returns []*User
userRepo.Create(ctx, &User{Name: "Jane"})
```

## Next steps

- [Provider registry](/docs/database/provider-registry) — registering a backend
  and getting it back typed.
- [Generic repository](/docs/database/generic-repository) — the API your
  application code calls.
- [The connection seam](/docs/database/connection-seam/) — how a connection
  reaches the rest of the framework.
