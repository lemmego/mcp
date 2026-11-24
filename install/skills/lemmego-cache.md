## When to use this skill

When caching a value, taking a lock, or invalidating by tag.

## Two layers, and only one of them is an interface

`Store` is the driver contract and deals **only in `[]byte`**. `Cache` sits
above it and owns encoding, events and the generic API.

**`Cache` is concrete and must stay that way.** Its generic methods cannot
exist on an interface, so a function that wants "any cache" has to take
`*cache.Cache`, not an interface you define. Do not introduce one — it will not
compile against the typed accessors, which are the reason to use this module.

## Typed access

    user, found, err := c.GetAs[User](ctx, "user:7")

`GetAs[T]` and `RememberAs[T]` are generic methods on `Cache`. Drivers store
`[]byte` and a codec does the rest, so the type parameter is where the shape
lives.

Three return values, and `found` is not an error: a miss is an ordinary
outcome, and conflating it with a failure is how a cache outage turns into a
404.

## Four stores, registered the way database/sql drivers are

`memory`, `file`, `redis`, `null`. `null` is a real choice, not a placeholder —
it makes "caching disabled" a configuration rather than a branch in the
application.

## Locks are atomic, with owner tokens

    lock := c.Lock("import", time.Minute)

Process-, host- or cluster-wide depending on the store — a memory store cannot
give you a cluster lock, so the guarantee follows the driver. The **owner
token** is what makes release safe: without it, a slow holder whose lock has
already expired can release somebody else's.

## Tagged invalidation is version-namespaced

A flush is **one write per tag and no scan**, which is why it is safe to use on
Redis in production. Reach for a tag rather than iterating keys.

## Events are free when nobody listens

Hit, miss, written, forgotten, flushed. Useful for a cache-ratio metric without
instrumenting call sites.

## What is deliberately not built

Do not look for these: multi-level caching, Memcached, or automatic
invalidation on data mutation. The last one is the interesting omission — it
needs the cache to know your write paths, which is the coupling this module
exists without.

## Check first

`search-docs` section `cache`, `module-api` on `cache`.
