---
title: Tags
type: docs
prev: docs/cache/locks
next: docs/cache/events
weight: 124
sidebar:
  open: true
---

Tags group entries so they can be invalidated together:

```go
c.Tags("posts").Put(ctx, "recent", posts, time.Hour)
c.Tags("posts", "featured").Put(ctx, "hero", hero, time.Hour)

// Publishing a post invalidates everything tagged "posts".
c.Tags("posts").Flush(ctx)
```

`Tags` returns a `*Cache`, so everything works on a tagged cache —
including `Remember`:

```go
posts, err := c.Tags("posts").RememberAs(ctx, "recent", time.Hour, build)
```

Tags name a set, so the order does not matter: `Tags("a", "b")` and
`Tags("b", "a")` are the same namespace. An entry under several tags is
invalidated by flushing **any** of them.

A tagged entry is invisible to an untagged read, and to a read under different
tags. They are separate namespaces, not a filter.

## How invalidation works

Each tag has a version. The namespace an entry is written under is derived from
the versions of all its tags. Flushing a tag mints it a new version, so every
key written under the old one becomes unreachable at once.

That is **one write per tag and no scan**, regardless of how large the cache
is. The alternative — enumerating keys to find the tagged ones — is
proportional to the whole cache, on a data structure with no index for it.

The cost is one extra read per tagged operation, to look up the current
versions. It is not cached, because caching it would stop one process seeing
another's flush.

## Entries that outlive a flush

Invalidation makes old entries unreachable; it does not necessarily delete
them. On a store that can enumerate tag members — `memory` and `redis` — they
are deleted as well. On one that cannot, they remain until their own TTL
expires.

That is why writing a non-expiring value under tags is **refused** on the file
driver:

```go
err := c.Tags("posts").Forever(ctx, "hero", hero)
// cache.ErrTagsRequireTTL
```

Such an entry would never expire and could never be reached again after a
flush, occupying space permanently with nothing able to remove it. Laravel
leaks here; refusing is better than a leak nobody can find. Give the entry a
TTL, or use a driver with a tag index.

## When not to use tags

Tags cost a round trip on every operation and add a layer of indirection. Where
a key prefix will do — `posts:recent`, `posts:featured` — and the invalidation
is one known key, a plain `Forget` is simpler and cheaper.

Tags earn their cost when the set of keys to invalidate is not known in
advance.

## Next steps

- [Cache events](/docs/cache/events) — watching a flush happen.
- [Typed access](/docs/cache/typed-access) — the API tags wrap.
