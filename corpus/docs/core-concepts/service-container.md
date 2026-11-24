---
title: Service Container
type: docs
prev: docs/core-concepts/application-bootstrap
next: docs/core-concepts/provider-plugin-system
weight: 22
sidebar:
  open: true
---

Your handler needs to send an email. The mail transport was built at startup,
in a completely different file. How does your handler get hold of it?

You could pass it down through every function between the two, which means
every function in the middle takes an argument it does not use. Or you could
build a new one inside the handler, which means a new connection per request
and no way to swap it for a fake in a test.

The **service container** is the third option: a box you put ready-made things
into at startup and take them out of wherever you need them, keyed by their
Go type.

```go
// at startup, in a provider
app.AddService(mailer)

// later, in a handler
mailer := app.Get[*mail.Mailer](a)
```

Because the key is the type, you get a `*mail.Mailer` back — no casting, and a
typo is a compile error.

## Adding services

Register services via `app.AddService()`:

```go
type MyService struct {
    // ...
}

func (p *MyProvider) Provide(a app.App) error {
    svc := &MyService{}
    a.AddService(svc)
    return nil
}
```

## Retrieving Services

### Via the App Instance

```go
svc := a.Service(&MyService{}).(*MyService)
```

### Via the Generic Helper

```go
import "github.com/lemmego/api/app"

svc := app.Get[*MyService](a)
```

The generic `Get[T]` helper composes with type safety:

```go
session := app.Get[*session.Session](a)
filesystem := app.Get[*fs.FileSystem](a)
```

## Built-in Services

The framework registers several services through its providers:

| Service | Type | Provider |
|---------|------|----------|
| Session Manager | `*session.Session` | `session.Provider` |
| File System | `*fs.FileSystem` | `fs.Provider` |
| Database | `*orm.DB` | `ormconnector` |
| Database | `*gorm.DB` / `*bun.DB` | `gormconnector` / `bunconnector` |
| Inertia | `*inertia.Inertia` | `inertia.Provider` |
| Auth | `*auth.Auth` | `auth.Provider` |

## How It Works

The container uses reflection to store services by their pointer type:

```go
type serviceContainer struct {
    services map[reflect.Type]interface{}
    mutex    sync.RWMutex
}

func (sc *serviceContainer) Add(service interface{}) {
    t := reflect.TypeOf(service)
    sc.services[t] = service
}

func (sc *serviceContainer) Get(service interface{}) error {
    // Resolves the service by type and populates the provided pointer
}
```

## When to Use

Use the service container for:
- Framework internals (session, filesystem, database connections)
- Simple service registration where you need exactly one instance
- Services that need to be available across the entire application

For more advanced DI needs (multiple lifetimes, auto-resolution of dependencies, scoped instances), see the [Dependency Injection](/docs/core-concepts/dependency-injection/) page.

## Next steps

- [Dependency injection](/docs/core-concepts/dependency-injection) — lifetimes,
  automatic resolution and scoped containers.
- [Provider & plugin system](/docs/core-concepts/provider-plugin-system) —
  where services get registered.
