---
title: Provider & Plugin System
type: docs
prev: docs/core-concepts/service-container
next: docs/core-concepts/dependency-injection
weight: 23
sidebar:
  open: true
---

Adding a feature to your application usually means several small things at
once: open a connection, put something in the container, maybe add a route and
a CLI command. A **provider** is where all of that for one feature lives.

It is a struct with a `Provide` method. At startup the framework hands it your
application, and it registers whatever it contributes:

```go
func LoadProviders() []app.Provider {
    return []app.Provider{
        &ormconnector.Provider{},
        &session.Provider{},
        &mail.Provider{...},
    }
}
```

That list in `bootstrap/providers.go` is, in effect, what your application is
made of. Adding a module from these docs means adding a line to it; removing
the line removes the feature, with nothing left behind in framework code.

Your own code can be a provider too, and that is the usual way to set up
something at startup that your handlers need later.

## The provider interface

Every provider implements this interface:

```go
type Provider interface {
    Provide(a App) error
}
```

The `Provide` method receives the full `App` instance and can access configuration, register services, and set up middleware:

```go
type MyProvider struct{}

func (p *MyProvider) Provide(a app.App) error {
    cfg := a.Config()
    svc := NewMyService(cfg)
    a.AddService(svc)
    return nil
}
```

## Specialized Provider Interfaces

Providers can optionally implement more specific interfaces:

### RouteProvider

Registers routes during bootstrapping:

```go
type MyProvider struct{}

func (p *MyProvider) AddRoutes() app.RouteCallback {
    return func(a app.App) {
        r := a.Router()
        r.Get("/my-route", myHandler)
    }
}
```

### CommandProvider

Adds CLI commands:

```go
func (p *MyProvider) AddCommands() []app.Command {
    return []app.Command{
        func(a app.App) *cobra.Command {
            return &cobra.Command{
                Use:   "my:command",
                Short: "Does something",
                Run: func(cmd *cobra.Command, args []string) {
                    // implementation
                },
            }
        },
    }
}
```

### MiddlewareProvider

Registers app-level middleware:

```go
func (p *MyProvider) AddMiddlewares() []app.Handler {
    return []app.Handler{
        myCustomMiddleware,
    }
}
```

### ErrMapProvider

Registers error-to-handler mappings:

```go
func (p *MyProvider) AddErrMap() app.ErrMap {
    return app.ErrMap{
        app.ErrNotFound: notFoundHandler,
    }
}
```

### PublishableProvider

Registers assets for publishing:

```go
func (p *MyProvider) AddPublishables() []*app.Publishable {
    return []*app.Publishable{...}
}
```

### ShutdownProvider

Executes cleanup on application shutdown:

```go
func (p *MyProvider) Shutdown(ctx context.Context) error {
    return myDBConnection.Close()
}
```

## Registration

Providers are registered in `bootstrap/providers.go`:

```go
func LoadProviders() []app.Provider {
    return []app.Provider{
        &fs.Provider{},
        &session.Provider{},
        &inertia.Provider{},
        &ormconnector.Provider{},
        &auth.Provider{Opts: &auth.Opts{...}},
        &MyProvider{},
    }
}
```

## Built-in Providers

| Provider | Package | Purpose |
|----------|---------|---------|
| `fs.Provider` | `github.com/lemmego/api/providers/fs` | File system abstraction |
| `session.Provider` | `github.com/lemmego/api/providers/session` | Session management |
| `inertia.Provider` | `github.com/lemmego/inertia` | Inertia.js integration |
| `ormconnector.Provider` | `github.com/lemmego/ormconnector` | Built-in ORM database connection |
| `gormconnector.Provider` | `github.com/lemmego/gormconnector` | GORM database connection |
| `bunconnector.Provider` | `github.com/lemmego/bunconnector` | Bun ORM database connection |
| `auth.Provider` | `github.com/lemmego/auth` | Authentication |

## Next steps

- [Dependency injection](/docs/core-concepts/dependency-injection) — binding
  what your provider registers.
- [The connection seam](/docs/database/connection-seam/) — how a database
  provider reaches the rest of the framework.
- [Event system](/docs/core-concepts/event-system) — the lifecycle events a
  provider can listen for.
