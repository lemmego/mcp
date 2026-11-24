---
title: Atomic Locks
type: docs
prev: docs/cache/typed-access
next: docs/cache/tags
weight: 123
sidebar:
  open: true
---

An atomic lock stops the same work running twice — a nightly import triggered
by three servers, a webhook delivered twice, a user double-clicking Submit.

```go
lock := c.Lock("import:nightly", 10*time.Minute)

ran, err := lock.Get(ctx, func(ctx context.Context) error {
    return importEverything(ctx)
})
if err != nil {
    return err
}
if !ran {
    // Someone else is already doing it.
    return nil
}
```

`Get` acquires, runs the callback, and releases — including if the callback
panics, so a crash frees the lock rather than holding it until the TTL.

## How far the exclusion reaches

**Only as far as the store.** This is the one thing to get right:

| Driver | A lock excludes |
|---|---|
| `redis` | every process talking to that Redis |
| `file` | every process on that host |
| `memory` | other goroutines in the same process |
| `null` | nothing at all |

A `memory` lock guarding work that runs on three machines guards nothing. If
the lock matters, use `redis`.

A store with no lock support returns a lock that **refuses** — `Acquire`
reports `cache.ErrUnsupported` rather than pretending to have succeeded. A
caller that ignores the error never proceeds believing it holds one.

## Waiting

```go
acquired, err := lock.Block(ctx, 5*time.Second)
if err != nil {
    return err
}
if !acquired {
    return errors.New("the importer is busy")
}
defer lock.Release(ctx)
```

`Block` polls with jittered backoff and honours context cancellation. Reaching
the deadline returns `(false, nil)` — a lock that was busy is an expected
outcome, not an error.

## Owner tokens

Every lock carries a random owner token, and only the holder can release one:

```go
lock := c.Lock("import", time.Minute)
lock.Owner() // an opaque token
```

This is not ceremony. A lock has a TTL so that a process that dies does not
hold it forever, which means a holder can lose its lock mid-work by running
past the TTL. If release were unconditional, that holder would then delete a
lock someone else had already acquired, and two workers would run at once —
the exact failure the lock exists to prevent.

On Redis the release is a single Lua compare-and-delete for the same reason:
reading the owner and then deleting are two operations, and the lock can change
hands between them.

## Handing a lock over

A lock taken in a web request can be released by the job that request queued:

```go
// In the request
lock := c.Lock("import", time.Hour)
if acquired, _ := lock.Acquire(ctx); acquired {
    queue.Dispatch(ctx, &ImportJob{LockOwner: lock.Owner()})
}

// In the job
lock := c.RestoreLock("import", job.LockOwner, time.Hour)
defer lock.Release(ctx)
```

## Extending

A holder still working as its lease runs down can push it out. On Redis:

```go
if extender, ok := lock.(interface {
    Extend(context.Context, time.Duration) (bool, error)
}); ok {
    stillHeld, err := extender.Extend(ctx, 5*time.Minute)
    if !stillHeld {
        // The lock was lost. Stop; someone else has it.
    }
}
```

A false return means the lock is already gone, which tells the caller to stop
rather than carry on believing it is protected.

## Locks and tags

A tagged cache does not provide locks. A tag namespace can be flushed out from
under a lock, which would leave the lock unreachable while still held.

## Next steps

- [Tags](/docs/cache/tags) — grouping entries for invalidation.
- [Queue](/docs/queue) — where a lock usually guards a job.
