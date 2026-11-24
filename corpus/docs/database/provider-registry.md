---
title: Provider Registry
type: docs
prev: docs/database/
next: docs/database/connection-seam
weight: 61
sidebar:
  open: true
---

A **provider** is the piece that knows how to talk to one particular database
— there is a GORM provider, a Bun provider, a Redis provider. You set one up
at startup, and from then on your code asks for it by name instead of passing
a connection through every function.

The **registry** is where they are kept. Think of it as a labelled shelf: you
put the Postgres provider on it under the name `"primary"` and the Redis one
under `"cache"`, and anywhere in your application you can take either back
down.

What makes it worth having is that you get the right *type* back, not a
`Provider` you have to assert:

```go
gpa.Register[*gpagorm.Provider]("primary", gormDB)

primary := gpa.MustGet[*gpagorm.Provider]("primary")   // already the right type
```

## Architecture

Providers are stored in a two-level map:

```
providers map[string]map[string]Provider
           ^                ^
           |                +-- instance name (e.g., "primary", "cache")
           +-- provider type name (e.g., "gorm", "redis")
```

## Registration

### Register a Provider

```go
import (
    "github.com/lemmego/gpa"
    "github.com/lemmego/gpagorm"
    "github.com/lemmego/gparedis"
)

// Register a GORM provider as "primary"
gpa.Register[*gpagorm.Provider]("primary", gormProvider)

// Register with registry defaults
gpa.RegisterDefault[*gpagorm.Provider](gormProvider)

// Register a Redis provider as "cache"
gpa.Register[*gparedis.Provider]("cache", redisClient)
```

### From an Application Provider

```go
type DatabaseProvider struct{}

func (p *DatabaseProvider) Provide(a app.App) error {
    cfg := a.Config()
    provider, err := gpagorm.NewProvider(gpa.Config{
        Host:     cfg.String("database.host"),
        Port:     cfg.Int("database.port"),
        Database: cfg.String("database.name"),
    })
    if err != nil {
        return err
    }
    gpa.Register[*gpagorm.Provider]("default", provider)
    return nil
}
```

## Retrieval

### Type-Safe Retrieval

```go
// With compile-time type inference
primary := gpa.MustGet[*gpagorm.Provider]("primary")
cache := gpa.MustGet[*gparedis.Provider]("cache")

// With error handling
provider, err := gpa.Get[*gpagorm.Provider]("primary")
if err != nil {
    // handle error
}
```

### Default Instance

If no instance name is provided, the registry looks up `"default"`:

```go
provider := gpa.MustGet[*gpagorm.Provider]()
```

### Multiple Instances

```go
// Get all instances of a type
gormProviders, err := gpa.GetByType[*gpagorm.Provider]()
```

## Lifecycle Management

Registration and lookup are package-level generic functions, but the
lifecycle operations are methods on the registry, reached through
`gpa.Registry()`:

```go
r := gpa.Registry()

// List registered types
types := r.ListTypes()

// List instances for a type
instances, err := r.ListInstances("gorm")

// Remove and close a provider
err := r.Remove("gorm", "primary")

// Remove and close all providers
err := r.RemoveAll()

// Health check
health := r.HealthCheck()
// Returns map[string]map[string]error — [type][instance]error
```

`ListInstances`, `Remove` and `RemoveAll` all report an error; only
`ListTypes` and `HealthCheck` cannot fail.

## Type Inference

The registry maps Go `reflect.Type` to provider type names. When `Register[T]` is called:

1. The provider's `ProviderInfo().Name` is used as the type key
2. `reflect.TypeOf(provider)` is mapped to this name
3. On `Get[T]()`, the type is looked up in reverse to find the correct provider

This means resolution is fully type-safe at compile time — no string matching required.

## Provider Interface

All providers implement:

```go
type Provider interface {
    Configure(config Config) error
    Health() error
    Close() error
    SupportedFeatures() []Feature
    ProviderInfo() ProviderInfo
}
```

And optionally one of:
- `SQLProvider` — for SQL databases (GORM, Bun)
- `KeyValueProvider` — for key-value stores (Redis)
- `DocumentProvider` — for document databases (MongoDB)
- `GraphProvider` — for graph databases
- `WideColumnProvider` — for wide-column stores

### Type Assertion Helpers

```go
if sqlProvider, ok := gpa.AsSQLProvider(provider); ok {
    db := sqlProvider.DB()
}

if gpa.IsSQLProvider(provider) {
    // provider is a SQL provider
}
```

## Next steps

- [Generic repository](/docs/database/generic-repository) — what you do with a
  provider once you have one.
- [The connection seam](/docs/database/connection-seam/) — registering during
  boot, in the right order.
- [GORM provider](/docs/database/providers/gorm) — a worked registration end to
  end.
