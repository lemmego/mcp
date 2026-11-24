---
title: Drivers
type: docs
prev: docs/cache/
next: docs/cache/typed-access
weight: 121
sidebar:
  open: true
---

## Choosing one

| | memory | file | redis | null |
|---|---|---|---|---|
| Survives a restart | no | yes | yes | — |
| Shared between processes | no | same host | yes | — |
| Shared between machines | no | no | **yes** | — |
| Locks span | this process | this host | **the cluster** | nothing |
| Tag flush removes entries | yes | on expiry | yes | — |
| Can store `Forever` under tags | yes | refused | yes | — |
| Honours context cancellation | no | no | yes | no |

Only the drivers you import are compiled in:

```go
// All four.
import _ "github.com/lemmego/cache/drivers"

// Or just what you need, to keep a Redis client out of the binary.
import _ "github.com/lemmego/cache/store/memory"
import _ "github.com/lemmego/cache/store/file"
```

Drivers register themselves the way `database/sql` drivers do. Selecting a
driver that has not been imported reports the ones that were.

## file

The default, and the right one for most single-server deployments.

Entries live under `storage/framework/cache`, sharded two levels deep so one
directory never holds the whole cache. Writes go to a temporary file and are
renamed into place, so a reader sees either the old entry or the new one, never
half of one. A truncated or corrupt entry reads as a **miss** rather than an
error — losing a cache entry must never fail a request.

It is the default because the CLI and the server are separate processes.
`lemmego run cache:clear` can reach a file cache; it cannot reach a memory one.

Expired entries are reclaimed by `cache:prune` rather than by a background
sweeper, because every process sharing the directory would otherwise walk the
tree on its own timer.

Its locks rest on exclusive file creation, which is atomic on a local
filesystem and **is not on NFS**.

## memory

The fastest, and private to one process. A second process — a queue worker, a
CLI command, a second server — has its own, entirely separate.

Expired entries are swept by a background goroutine as well as on read, because
lazy expiry alone leaks: an entry written once and never read again would hold
its memory until the process exits.

Values are copied in and out, so a caller mutating a slice it stored or
received cannot corrupt the cache.

Laravel calls this driver `array`; that name is accepted as an alias.

## redis

The only driver that works across machines, and the only one whose locks do.

`Flush` scans for the configured prefix and deletes what it finds. It
does **not** call `FLUSHDB`: a Lemmego application usually shares
one Redis database between its cache, its sessions and its queue, and emptying
the database would log every user out and drop queued work. Set
`allow_flush_db` only when the database holds nothing else.

This is why `prefix` matters more here than anywhere else — it is what bounds a
flush.

Counters use `INCRBY`, which preserves an existing key's expiry. A key it
creates has no expiry, matching Redis itself.

## null

Keeps nothing. Reads always miss, so the cache-miss path runs on every request
— useful in tests, and what `lenient` falls back to when the configured backend
cannot be reached.

Its locks always "succeed" and therefore exclude nothing. That matches
Laravel's behaviour, and it is worth knowing before relying on a lock in a test
that uses this driver.

## Writing a driver

Implement `cache.Store`, and register it:

```go
func init() {
    cache.Register("mystore", func(ctx context.Context, cfg *cache.Config) (cache.Store, error) {
        return New(cfg.Prefix)
    })
}
```

The conformance suite the built-in drivers are held to is exported, so a driver
outside this repository can prove itself against the same contract:

```go
func TestConformance(t *testing.T) {
    storetest.Run(t, storetest.Capabilities{
        Locks:           true,
        AtomicIncrement: true,
        Persists:        true,
    }, newStore)
}
```

`Capabilities` says what the store can do, so the suite skips what does not
apply rather than failing it. The factory also returns a clock the suite
advances, so expiry is tested without sleeping.

Optional capabilities are separate interfaces a store may implement —
`LockProvider`, `TagIndex`, `Pruner`. A store that does not implement one makes
callers get `cache.ErrUnsupported`, which is always better than a silent no-op.

## Next steps

- [Typed access](/docs/cache/typed-access) — the encoding layer above every
  driver.
- [Atomic locks](/docs/cache/locks) — how far exclusion reaches on each driver.
- [Redis provider](/docs/database/providers/redis) — using Redis as a store in
  its own right.
