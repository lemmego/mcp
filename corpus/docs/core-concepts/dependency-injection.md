---
title: Dependency Injection
type: docs
prev: docs/core-concepts/provider-plugin-system
next: docs/core-concepts/event-system
weight: 24
sidebar:
  open: true
---

**Dependency injection** sounds grander than it is. It means: when a piece of
code needs something, hand it in from outside rather than letting it build its
own.

The reason is substitution. A handler that constructs its own mail client can
only ever send real email, so testing it sends real email. A handler that is
*given* one takes a real client in production and a fake in a test, and the
handler itself does not change.

The [service container](../service-container) covers the simple case — put a
thing in, take it out by type. This page is the fuller `di` package, which you
want when something needs building rather than just storing: when a service
has its own dependencies, or when you need a fresh one per request rather than
one shared for the life of the process.

## Service lifetimes

A **lifetime** answers: when is this built, and how long does it live?

Three lifetimes determine when a service instance is created and how long it lives:

| Lifetime | Behavior |
|----------|----------|
| **Transient** | A new instance is created every time the service is resolved |
| **Singleton** | A single instance is created and shared for the lifetime of the container |
| **Scoped** | A single instance is created per scope (e.g., per HTTP request) |

## Basic Usage

```go
import "github.com/lemmego/api/di"

container := di.New()
```

### Registration

```go
// Singleton — one instance for the application lifetime
di.RegisterSingleton[*Database](container, func() *Database {
    return NewDatabase()
})

// Transient — new instance each time
di.RegisterTransient[*UserService](container, func(db *Database) *UserService {
    return NewUserService(db)
})

// Or register with explicit lifetime
di.Register[*UserService](container, di.Scoped, func() *UserService {
    return NewUserService()
})
```

### Resolution

```go
db, err := di.Resolve[*Database](container)
userSvc := di.MustResolve[*UserService](container)
```

### Check Registration

```go
if di.Has[*UserService](container) {
    svc := di.MustResolve[*UserService](container)
}
```

## Automatic Dependency Resolution

Factory functions can declare dependencies as parameters. The container resolves them automatically:

```go
di.RegisterSingleton[*Database](container, func() *Database {
    return NewDatabase("postgres://...")
})

di.RegisterTransient[*UserRepository](container, func(db *Database) *UserRepository {
    return NewUserRepository(db)
})

di.RegisterTransient[*UserService](container, func(repo *UserRepository) *UserService {
    return NewUserService(repo)
})

// Resolving UserService automatically resolves Database and UserRepository
svc := di.MustResolve[*UserService](container)
```

## Fluent API

The fluent `For[T]` API provides a cleaner registration syntax:

```go
di.For[*Database](container).
    AsSingleton().
    Use(func() *Database {
        return NewDatabase("postgres://...")
    })

di.For[*UserService](container).
    AsTransient().
    Use(func(db *Database) *UserService {
        return NewUserService(db)
    })

// Register an existing instance
di.For[*Config](container).
    AsSingleton().
    UseInstance(&Config{Port: 8080})
```

## Scoped Containers

Scoped services are useful for per-request dependencies:

```go
// Register a scoped service
di.For[*RequestContext](container).
    AsScoped().
    Use(func() *RequestContext {
        return &RequestContext{RequestID: uuid.New().String()}
    })

// Create a scope (e.g., per HTTP request)
scope := container.CreateScope()

// Resolving within the scope returns the same instance
ctx1 := di.MustResolve[*RequestContext](scope)
ctx2 := di.MustResolve[*RequestContext](scope)
// ctx1 == ctx2 (same instance within scope)

// Different scope = different instance
scope2 := container.CreateScope()
ctx3 := di.MustResolve[*RequestContext](scope2)
// ctx3 != ctx1 (different scope)
```

## Circular Dependency Detection

The container detects circular dependencies at resolution time and returns an error instead of entering an infinite loop.

## Container Access

The container instance itself can be injected into factories:

```go
di.For[*ServiceLocator](container).
    AsSingleton().
    Use(func(c *di.Container) *ServiceLocator {
        return &ServiceLocator{container: c}
    })
```

## When to Use DI

The `di` package is suitable for complex applications that need:
- Multiple service lifetimes (transient, singleton, scoped)
- Automatic dependency resolution
- Scoped request-based services
- Fluent registration syntax

For simpler cases, the service container (`app.AddService` / `app.Get[T]`) is sufficient.

## Next steps

- [Service container](/docs/core-concepts/service-container) — the simpler API
  most application code uses.
- [Provider & plugin system](/docs/core-concepts/provider-plugin-system) —
  where bindings are declared.
