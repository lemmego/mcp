---
title: Request Debugging
type: docs
prev: docs/logging
next: docs/security/
weight: 91
sidebar:
  open: true
---

`lemmego/observe` records what your application did while answering one
request — and underneath it, every query, log line, cache operation, job and
mail that request caused, in order, with timings.

Reach for it when a page is slow or wrong and your logs cannot tell you why.
A log is a flat stream with no notion of which request a line belongs to, so
"what did *this* request do" is the one question it cannot answer.

```
$ lemmego run observe:last
  request   412.80ms  GET /feed
  query       3.10ms  SELECT * FROM posts WHERE circle_id IS NULL ORDER BY id DESC LIMIT 25
  query       1.90ms  SELECT * FROM users WHERE id = $1
  query       1.80ms  SELECT * FROM users WHERE id = $1
  …
  log         0.00ms  the avatar renderer fell back to the default

N+1: 25 executions of SELECT * FROM users WHERE id = ?  (47ms total)
```

## Install

```bash
go get github.com/lemmego/observe
```

Entries live in one table, so publish the migration and run it:

```bash
lemmego run publish --tags=observe-migrations
go build ./...          # the published migration must be compiled in
lemmego run migrate up
```

{{< callout type="warning" >}}
Don't skip the rebuild. The migration was written by a binary that does not
contain it yet, so `migrate up` won't see it until you build again.
{{< /callout >}}

Upgrading from an earlier version? Publish again — you'll get an `ALTER` for
each column added since, and a republish only writes files that are missing.

## Turn it on

Register the provider after the database connector whose connection it
borrows, and before anything whose queries you want recorded:

```go
// bootstrap/providers.go
import (
    observeprovider "github.com/lemmego/observe/provider"
    "github.com/lemmego/observe/watch/observeslog"
)

func LoadProviders() []app.Provider {
    return []app.Provider{
        &logger.Provider{Options: log.Options{
            // Your log lines land under the request that produced them.
            Extra: []slog.Handler{observeslog.Handler()},
        }},
        &ormconnector.Provider{},
        &observeprovider.Provider{
            DashboardAuth: func(c app.Context) bool {
                _ = auth.Check(c)
                user, ok := auth.UserAs[*models.User](c)
                return ok && user.IsStaff
            },
        },
    }
}
```

```bash
OBSERVE_ENABLED=true
```

It records nothing until you set that, because entries hold request headers,
bound query parameters and every statement you ran.

{{< callout type="warning" >}}
Call `auth.Check(c)` inside `DashboardAuth`, as above. The dashboard mounts on
the bare mux, so none of your route middleware has run by then, and
`auth.UserAs` is a context read rather than a lookup. Leave the `Check` out
and the predicate answers false for everybody — including you.
{{< /callout >}}

## Record your queries

Queries are collected below your ORM, at the `database/sql` driver layer, so
one wrapper covers `lemmego/orm`, GORM, Bun, `sqlx`, raw `database/sql`,
migrations, `tasker` and your session store.

Register a wrapped driver and point your configuration at it:

```go
import (
    "database/sql"
    "github.com/lemmego/observe"
    "github.com/lib/pq"
)

func init() {
    sql.Register("postgres-observed", observe.Wrap(&pq.Driver{}))
}
```

```bash
DB_DRIVER=postgres-observed
```

That's the whole wiring. Two cases need a different hook: hand a GORM
dialector an already-instrumented `*sql.DB`, and wrap a `driver.Connector`
with `observe.WrapConnector` when you use `sql.OpenDB`.

Not on `database/sql` at all? Record it yourself in one line, and the entry is
treated identically everywhere else:

```go
started := time.Now()
cursor, err := collection.Find(ctx, filter)
observe.RecordQuery(ctx, "mongo", "users.find", args, time.Since(started), err)
```

## Read it three ways

**A dashboard**, at `/observe`. Server-rendered Go with no build step. Click a
request to see everything it did, or an entry to see its payload — a panic's
stack or a dump's value. It tails live over SSE, so it's current rather than
seconds stale.

**A terminal.**

```bash
lemmego run observe:last
lemmego run observe:last --request <id>
```

**An AI agent**, over MCP — the half a dashboard can't be:

```go
&mcp.Provider{Options: []mcp.Option{
    mcp.WithTools(observemcp.Tools()),
}},
```

Provider order doesn't matter here; the tools resolve the recorder when a call
arrives.

| Tool | Answers |
|---|---|
| `observe-requests` | recent requests, with timings |
| `observe-request` | one request, in full, in order |
| `observe-slow-queries` | the slowest statements |
| `observe-repeated-queries` | N+1 candidates, by request |
| `observe-errors` | error-level entries of every kind |
| `observe-tagged` | everything carrying a set of tags, across requests |

## Wire the rest of the watchers

Requests and queries come from the provider and the driver wrapper. Everything
else attaches to something your application already has — one line each, but a
watcher you don't wire records nothing and says nothing about it.
`observe:last` prints which are active.

| Watcher | Wire it |
|---|---|
| `exception` | `observehttp.Recover()` **in place of** your recoverer |
| `slog` | `observeslog.Handler()` in `log.Options.Extra` |
| `cache` | `observecache.Watch(cache, observe.Default())` |
| `job` | `observejob.Watch(observe.Default())` |
| `mail` | `Transport: observemail.Watch(transport, observe.Default())` |
| `notification` | `EventSink: observenotify.Sink()` on notify's provider |
| `httpclient` | `&http.Client{Transport: observe.WrapTransport(nil)}` |
| `authz` | `observeauthz.Watch(registry, observe.Default())` |
| `view` | `observeview.Watch(observe.Default())` |
| `event` | `observeevents.Watch(a, observe.Default(), "launch.published")` |
| `schedule` | `observeschedule.Watch(sched, observe.Default())` |
| `dump` | `observe.Dump(ctx, "label", value)` where you want it |

{{< callout type="warning" >}}
Attach watchers from a provider of your own, registered after the things they
watch. `observe.Default()` is nil until the provider has run, and
`LoadProviders` builds its whole slice first — so a watcher constructed inside
that slice gets nil and silently records nothing.
{{< /callout >}}

`observehttp.Recover()` **replaces** your recovering middleware rather than
joining it. A panic unwinds outward, so whichever recoverer sits nearest the
handler catches it and the other never fires:

```go
// bootstrap/http_middleware.go
func LoadHTTPMiddlewares() []app.HTTPMiddleware {
    return []app.HTTPMiddleware{
        observehttp.Recover(), // was middleware.Recoverer()
        middleware.RequestLogger(),
    }
}
```

You get the stack back. It answers 500 the way chi's does and still logs at
error level.

## Leaving it on in production

```bash
OBSERVE_PROFILE=production
```

One variable rather than five you'd have to remember together:

| | development | production |
|---|---|---|
| bound parameters | recorded | not recorded |
| statements | as written | normalised to shapes |
| log records | all | info and above |
| sample rate | everything | a tenth |

Anything you set explicitly in code wins over the profile.

Tune the rate on its own with `OBSERVE_SAMPLE`, between 0 and 1:

```bash
OBSERVE_SAMPLE=0.1
```

It samples **requests, not entries**. Keeping a tenth of the queries of every
request gives you a tenth of a story about all of them; keeping every query of
a tenth of requests gives you the whole story about some — and the whole story
is the one you can debug. The decision is a hash of the request id, so every
entry of one request agrees, including entries recorded minutes later by a
worker that inherited the id.

Failures are never dropped by the rate. A sampled-out request that broke is
the one you most need, so anything that failed, plus exceptions and dumps, is
kept regardless.

{{< callout type="info" >}}
`OBSERVE_PROFILE=production` stops recording bound parameters, but that isn't
a privacy control by itself: a value written into the statement *text* is part
of the statement. An ORM binds its values, so this is rare — hand-written SQL
that interpolates doesn't. Normalising statements to shapes is what removes
those, and the profile turns on both.
{{< /callout >}}

## Choosing a store

| Store | Use it when |
|---|---|
| `store/sqlstore` | almost always — every process can read it |
| `store/memory` | tests, and a single process with no database |

The memory store answers the wrong process: `lemmego run mcp:serve` boots a
second one and a worker is a third, so entries recorded by your web process
aren't in the memory of the process your agent is asking. The answer reads as
"this application makes no queries" rather than "you're looking in the wrong
place". The provider warns you when it falls back to it.

## Safety

- It's off unless `OBSERVE_ENABLED` says otherwise.
- `DashboardAuth` nil means **no access**, not no authentication — the
  dashboard answers 401 to everything until you give it a predicate.
- Set `OBSERVE_SAMPLE` before using this under real traffic.
- The live tail sits behind the same guard as everything else.
- Pausing is a `POST`, so your CSRF middleware applies to it.

## Writing your own store

`storetest` is exported, so a store of your own can prove itself against the
same contract both built-in stores pass:

```go
func TestMyStoreSatisfiesTheContract(t *testing.T) {
    storetest.Run(t, func(*testing.T) observe.Store { return NewMyStore() })
}
```

## Next steps

- [Logging](/docs/logging) — the channels and rotation behind the log watcher.
- [AI agents (MCP)](/docs/mcp) — the tools that read this recorder from an
  agent.
- [Error handling](/docs/http/error-handling) — the errors it records against a
  request.
