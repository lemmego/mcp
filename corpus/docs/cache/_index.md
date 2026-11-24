---
title: Caching
type: docs
prev: docs/ratelimit/
next: docs/cache/drivers
weight: 120
sidebar:
  open: true
aliases:
  - /docs/caching/
---

Some answers are expensive to work out and do not change often. A dashboard
that runs six queries, a list of countries, a rendered page — recomputing them
for every visitor is work you have already done.

A **cache** is somewhere fast to keep the answer so the next request skips the
work:

```go
total, err := cache.Remember(ctx, "orders:count", time.Hour,
    func(ctx context.Context) (int, error) {
        return repo.CountOrders(ctx)   // only runs on a miss
    })
```

Two ideas carry most of the use. A **miss** is when the value is not there, so
you have to do the work; a **hit** is when it is. And a **TTL** — time to live
— is how long an entry stays before it is thrown away, which is how you stop
serving something stale forever.

The `cache` module gives you that over several storage backends behind one
API, with typed access, atomic locks, tagged invalidation and events.

```go
import (
    "github.com/lemmego/cache"
    _ "github.com/lemmego/cache/drivers"
)

users, err := cache.Remember(c.RequestContext(), "users:active", 5*time.Minute,
    func(ctx context.Context) ([]models.User, error) {
        return repos.User(c.App()).Active(ctx)
    })
```

Generated projects register it already. In an existing project, add the
provider to `bootstrap/providers.go`:

```go
import (
    "github.com/lemmego/cache"
    _ "github.com/lemmego/cache/drivers"
)

func LoadProviders() []app.Provider {
    return []app.Provider{
        &cache.Provider{},
    }
}
```

and publish the config file:

```shell
lemmego publish --tags=config
```

## Two layers

`Store` is the driver contract. It deals only in `[]byte`, so a driver never
has to know how a value is encoded.

`Cache` sits above it and owns encoding, key namespacing and events. It is what
almost all application code should use.

`Cache` is a concrete type, and there is **no interface over it**.
Several of its methods declare their own type parameters — `GetAs[T]`,
`RememberAs[T]` — which Go permits only on concrete types, because a method
with type parameters can never satisfy an interface. Introducing a `Cacher`
interface would silently remove the typed API.

## Reading and writing

```go
c := cache.FromApp(a) // or use the package-level functions below

err := c.Put(ctx, "user:7", user, time.Hour)
err := c.Forever(ctx, "settings", settings)

user, found, err := c.GetAs[models.User](ctx, "user:7")
```

`GetAs` returns a bool alongside the error, and the two mean different things.
A miss is not a failure, so `err` stays reserved for a store that broke or a
value that would not decode — and the bool is what separates a miss from a
cached `0`, `false` or `""`.

```go
ok, err := c.Has(ctx, "user:7")
ok, err := c.Missing(ctx, "user:7")

// Store only if absent, atomically.
added, err := c.Add(ctx, "job:1:claimed", true, time.Minute)

// Read and remove in one step.
token, found, err := c.PullAs[string](ctx, "one-time-token")

forgotten, err := c.Forget(ctx, "user:7")
err := c.Flush(ctx)
```

Counters have their own operations, because they are read-modify-write and
have to be atomic:

```go
attempts, err := c.Increment(ctx, "login:attempts:"+ip, 1)
remaining, err := c.Decrement(ctx, "quota:"+userID, 1)
```

An existing expiry survives an increment, so a counter with a one-minute window
still resets after a minute.

## Remember

`Remember` returns the cached value, computing and storing it on a miss:

```go
posts, err := cache.Remember(ctx, "posts:recent", 10*time.Minute,
    func(ctx context.Context) ([]models.Post, error) {
        return repos.Post(a).Recent(ctx, 20)
    })
```

Concurrent callers within one process collapse to a single computation, so a
popular key going cold does not run the callback once per request.

If the callback returns an error, **nothing is cached** and the error is
returned. A failed computation must not be remembered as though it had worked.

`RememberForever` is the same without an expiry.

Across processes, add `cache.LockFor`:

```go
report, err := cache.Remember(ctx, "report:monthly", time.Hour,
    buildReport,
    cache.LockFor(2*time.Minute),
)
```

One process rebuilds while the others wait and then read what it wrote. It is
opt-in because it costs a lock round trip, and because it is only as strong as
the store's lock — see [Locks](/docs/cache/locks).

## Package-level functions

The same operations exist as package functions, working on the cache the
provider installed:

```go
cache.Put(ctx, "key", value, time.Hour)
value, found, err := cache.Get[models.User](ctx, "key")
cache.Remember(ctx, "key", ttl, build)
cache.Forget(ctx, "key")
cache.Flush(ctx)
```

They return `cache.ErrNotInitialized` when no cache is configured, rather than
panicking, so an application that has not set one up fails the call and not the
process.

Use `cache.FromApp(a)` to resolve the cache from the container instead. It is
spelled `FromApp` rather than `Get`, which belongs to the generic function
above.

## Configuration

`internal/configs/cache.go`:

| Key | Default | |
|---|---|---|
| `driver` | `file` | `memory`, `file`, `redis`, `null` |
| `prefix` | `lemmego_cache:` | Namespaces keys, and bounds what a flush removes |
| `ttl` | `3600` | Seconds |
| `codec` | `json` | `json` or `gob` |
| `events` | `false` | Publish events on the app's event emitter |
| `lenient` | `false` | Serve without a cache rather than failing to boot |

An unreachable backend fails the boot by default, so a wrong Redis address
stops a deployment rather than quietly degrading it. `lenient` turns that into
a warning and a cache that keeps nothing — every read misses, which is slower
but still serving.

## Commands

```shell
lemmego run cache:clear       # empty the cache
lemmego run cache:forget key  # remove one key
lemmego run cache:prune       # reclaim expired entries (file driver)
```

These run in a different process from the server, which is why the default
driver is `file` rather than `memory`. `cache:clear` against a memory cache
says so instead of reporting a success it cannot deliver.

## Next steps

- [Drivers](/docs/cache/drivers) — choosing between memory, file, Redis and
  null.
- [Typed access](/docs/cache/typed-access) — getting a `*User` back instead of
  bytes.
- [Atomic locks](/docs/cache/locks) — making sure only one process does a
  thing.
