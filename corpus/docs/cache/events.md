---
title: Cache Events
type: docs
prev: docs/cache/tags
next: docs/queue
weight: 125
sidebar:
  open: true
---

## Listening

```go
c := cache.FromApp(a)

c.Listen(cache.EventMissed, func(e cache.Event) {
    missed := e.(cache.CacheMissed)
    slog.Info("cache miss", "key", missed.Key, "tags", missed.Tags)
})
```

| Event | Type | |
|---|---|---|
| `cache.EventHit` | `CacheHit` | Key, Tags, Value |
| `cache.EventMissed` | `CacheMissed` | Key, Tags |
| `cache.EventKeyWritten` | `KeyWritten` | Key, Tags, Value, TTL |
| `cache.EventKeyForgotten` | `KeyForgotten` | Key, Tags |
| `cache.EventFlushed` | `CacheFlushed` | Tags |

Events are concrete types, so a listener reads `e.Key` after one assertion
rather than digging through a map.

Forgetting a key that was not there raises nothing: no event means nothing
happened.

## Cost

Events cost nothing when nobody is listening. Subscriptions are summarised into
a single atomic word, so the check on a cache read is one atomic load; and the
event value is constructed *inside* that check, not before it. A cache read
with no listeners allocates exactly as much as one in a build with no events at
all.

That matters because `Get` sits on the hot path of whatever it is caching, and
a `CacheHit` carries the stored bytes — building one per read regardless of
whether anyone wanted it would be a real cost.

## Where listeners run

Synchronously, on the goroutine that touched the cache. A slow listener slows
the request that triggered it, so keep them cheap — count a metric, write a log
line — and hand anything expensive to a queue.

A listener that panics is contained and logged rather than allowed to fail the
request that happened to read the cache.

## On the application emitter

Setting `events: true` also publishes them on the application's event emitter,
for listeners registered with `a.On`:

```go
config.Set("cache", config.M{"events": true})
```

It is off by default because dispatch is synchronous either way, and because
most applications do not need it.

## What to use them for

Cache hit rate is the usual one — a counter on hits and misses tells you
whether a cache is earning its keep, which is otherwise invisible:

```go
c.Listen(cache.EventHit, func(cache.Event) { hits.Add(1) })
c.Listen(cache.EventMissed, func(cache.Event) { misses.Add(1) })
```

In development, logging every miss tends to reveal keys that never hit —
usually a key built from something that varies per request.

## Next steps

- [Event system](/docs/core-concepts/event-system) — the application emitter
  these can reach.
- [Request debugging](/docs/observe/) — cache activity attributed to one
  request.
