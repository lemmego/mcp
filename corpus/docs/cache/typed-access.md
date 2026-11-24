---
title: Typed Access
type: docs
prev: docs/cache/drivers
next: docs/cache/locks
weight: 122
sidebar:
  open: true
---

## Encoding

A driver stores bytes. A codec turns values into them:

| Codec | |
|---|---|
| `json` (default) | Readable in `redis-cli`, portable across languages, cannot be tripped up by an unregistered type |
| `gob` | Preserves Go types exactly, including unexported fields via `GobEncoder`; readable only by Go, and a type reached through an interface field must be `gob.Register`ed |

JSON's limitation is type fidelity — a value decoded into `any` arrives as
`float64` rather than `int`. Decoding into a concrete type, which the generic
API does, avoids it entirely.

```go
config.Set("cache", config.M{"codec": "gob"})
```

## The generic methods

```go
user, found, err := c.GetAs[models.User](ctx, "user:7")
users, err := c.GetManyAs[models.User](ctx, []string{"user:7", "user:8"})
token, found, err := c.PullAs[string](ctx, "one-time")
posts, err := c.RememberAs(ctx, "recent", time.Hour, build)
```

The `...As` suffix follows `session.GetAs` and `config.LookupAs`: a typed
variant of a method that also exists untyped. The package-level functions take
the plain names — `cache.Get[T]`, `cache.Remember[T]` — matching `app.Get[T]`.

### The `found` return

```go
value, found, err := c.GetAs[int](ctx, "count")
```

Check `found`, not the value. A cached `0` and an absent key are both the zero
value of `int`, and the same applies to `false` and `""`.

`err` means the store broke or the value would not decode. A miss is not an
error, so `if err != nil { return err }` will not propagate one.

At the driver layer the convention is reversed: `Store.Get` returns
`cache.ErrMiss`, which composes with `errors.Is` through any decorators a
store is wrapped in.

## Stampede protection

When a popular key expires, every request that misses it will try to rebuild
it at once. `Remember` collapses those within a process to a single call:

```go
// Fifty concurrent requests, one call to build.
value, err := cache.Remember(ctx, "hot", time.Minute, build)
```

Across processes, that is not enough: four servers each run their own copy of
that machinery, so the callback runs four times. `LockFor` adds a lock:

```go
value, err := cache.Remember(ctx, "hot", time.Minute, build,
    cache.LockFor(30*time.Second),
    cache.WaitFor(30*time.Second),
)
```

One process wins, rebuilds, and writes. The others wait for it and then read
what it wrote, falling back to building it themselves only if the winner failed
to write anything.

It is opt-in because it costs a lock round trip on every miss, and because a
lock is only as strong as the store behind it — over a memory store it still
spans one process.

## TTLs

```go
c.Put(ctx, "key", value, time.Hour)   // expires
c.Forever(ctx, "key", value)          // does not
```

`Put` **rejects a non-positive TTL** with `cache.ErrInvalidTTL`. A zero
`time.Duration` is what an uninitialised struct field or a config key that did
not parse holds, and Laravel's rule — where a non-positive TTL deletes the key
— turns that mistake into silent data loss. `Forever` is how you ask for a
non-expiring entry deliberately.

## Next steps

- [Tags](/docs/cache/tags) — invalidating a group of entries at once.
- [Atomic locks](/docs/cache/locks) — the other half of stampede protection.
