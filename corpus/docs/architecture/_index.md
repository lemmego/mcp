---
title: Architecture
type: docs
prev: docs/prompting
next: docs/appendix/
weight: 160
sidebar:
  open: true
---

This page is a map of how a request moves through a Lemmego application and
which layer owns which decision. It is reference material: if you are building
something, the pages it links to are where the instructions are.

## Request path

```
HTTP Request
    |
    v
+------------------+
| HTTP Middleware  |  Recovery, Logging, Method Override (net/http level)
+------------------+
    |
    v
+------------------+
| Router           |  Chi-based routing, URL matching
+------------------+
    |
    v
+------------------+
| App Middleware   |  CSRF, Auth, custom app middleware (Handler level)
+------------------+
    |
    v
+------------------+
| Route Handler    |  Business logic, renders response
+------------------+
    |
    v
+------------------+
| Service Layer    |  Business logic services
+------------------+
    |
    v
+------------------+
| Repository/GPA   |  Data access (type-safe generic repositories)
+------------------+
    |
    v
+------------------+
| Database/Storage |  PostgreSQL, MySQL, SQLite, Redis, S3, etc.
+------------------+
```

## Patterns you will meet

### Builder Pattern (Bootstrap)

The application is configured through a chain of `With*` methods:

```go
app.Configure().
    WithConfig(cfg).
    WithRoutes(routes).
    WithProviders(providers).
    Run()
```

This provides a clear, immutable configuration pipeline where each step adds a concern.

### Provider/Plugin Pattern

Services are registered via `Provider` implementations. Each provider has access to the full `App` instance during bootstrapping:

```go
type Provider interface {
    Provide(a App) error
}
```

This enables:
- Loose coupling between components
- Pluggable backend support (GPA providers)
- Framework extensibility without modifying core code

### Middleware Pipeline

Two middleware layers serve different purposes:
- **HTTP Middleware** wraps the entire router (recovery, logging)
- **App Middleware** operates at the handler level (CSRF, auth)

### Service Container

Simple type-based service registry with generic retrieval:

```go
a.AddService(myService)
svc := app.Get[*MyService](a)
```

### Repository Pattern

Data access is abstracted through generic repositories:

```go
type Repository[T any] interface {
    Create(ctx, entity *T) error
    FindByID(ctx, id) (*T, error)
    // ...
}
```

### Convention over Configuration

The framework follows sensible defaults:
- Route files in `internal/routes/`
- Templates in `templates/` with `.page.gohtml` / `.layout.gohtml` / `.partial.gohtml` naming
- Configuration auto-loaded from `internal/configs/` via `init()`
- Auto-detection of frontend tooling (Templ, Vite, etc.)

### Error Handling via ErrMap

Errors are mapped to handlers through a central `ErrMap`:

```go
type ErrMap = map[error]Handler
```

This provides centralized, consistent error responses across the application.

## Data Flow

### Request Lifecycle

1. Client sends HTTP request
2. HTTP middleware processes (recovery, logging)
3. Router matches the URL to a route
4. Route-level middleware runs (CSRF, auth)
5. Input decoding middleware parses the request body
6. Handler executes business logic
7. Handler renders response (JSON, HTML, redirect, etc.)
8. Response middleware runs (logging, headers)

### Dependency Flow

```
Config → Providers → Services → Repositories → GPA Providers → Database
                ↓
           Routes + Middleware
                ↓
          HTTP Server
```

## Next steps

- [Core concepts](/docs/core-concepts/) — the bootstrap, container and provider
  system up close.
- [The HTTP layer](/docs/http/) — the request path in detail.
- [GPA overview](/docs/database/) — the persistence abstraction underneath.
