---
title: Rate Limiting
type: docs
prev: docs/tenancy/
next: docs/cache/
weight: 109
sidebar:
  open: true
---

`lemmego/ratelimit` caps how often a caller can hit a route. Use it on
anything a stranger can reach — sign-in, registration, password reset — and on
signed-in actions that are cheap to repeat, such as voting or commenting.

It is a fixed-window counter stored in the application's cache.

## Install

```shell
go get github.com/lemmego/ratelimit
```

It needs [`lemmego/cache`](/docs/cache/) registered and nothing else. There is
no provider to add: the counters live in the cache and the middleware is a
plain function.

## Limit a route

```go
import (
    "time"

    "github.com/lemmego/ratelimit"
)

r.Post("/login", ratelimit.Middleware(ratelimit.PerIP(5, time.Minute)), handlers.Login)
```

Five requests per minute per IP. A sixth gets a `429` and never reaches the
handler.

## Limit something that is not a route

`Allow` takes a key you choose and returns whether to proceed, plus how long
until the caller may retry:

```go
ok, retryAfter, err := ratelimit.Allow(c.RequestContext(), "vote:"+userID, 30, time.Minute)
if err != nil {
    // The cache could not be reached. See Behaviour when the cache is down.
}
if !ok {
    c.SetHeader("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
    return c.Error(http.StatusTooManyRequests, errors.New("too many votes"))
}
```

## Choosing a key strategy

```go
ratelimit.PerIP(5, time.Minute)       // anonymous: login, register, password reset
ratelimit.PerUser(30, time.Minute)    // signed in: vote, comment, post

&ratelimit.Opts{                      // anything else
    Limit:   10,
    Window:  time.Minute,
    Key:     myKeyFunc,          // func(c app.Context) string; defaults to ClientIP
    Message: "Slow down a moment.",
}
```

`PerRoute` is an alias for `PerUser`, kept because it reads well at a call
site.

**Prefer `PerUser` wherever sign-in is required.** A single IP covers an
office, a university and a mobile carrier's NAT, so limiting by IP there
throttles a crowd for one person's behaviour. `PerUser` falls back to the IP
for an anonymous request, which makes it safe on a route that serves both.

### How keys are built

Both strategies key on **the caller and the route together**, so each route
counts its own budget. Two routes limited at 20 an hour give a caller 20 each,
not 20 between them.

The route in the key is the **matched pattern**, not the request path. On a
route carrying a parameter, `/launch/{slug}/upvote` is one counter — so "20
upvotes an hour" means twenty in total, not twenty per launch. Keying on the
path would let a caller reset their budget by choosing a different target.

For one budget spanning several routes, set `Opts.Key` to something that does
not vary by route.

## Trusting a proxy

```go
ratelimit.TrustProxy(true)
```

`ClientIP` ignores `X-Forwarded-For` until you enable this, because the header
is supplied by the client. Enable it only when a proxy you control sets the
header and strips any incoming copy. Behind a proxy without it, every request
appears to come from the proxy and a per-IP limit throttles your whole site as
one caller.

## What a refused request looks like

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 43
```

`Retry-After` is always set. Send it on your own refusals too — a client that
retries immediately turns one rejected request into a tight loop, which is the
load the limiter exists to prevent.

To customise the body, set `Opts.Message`.

## Scope and limits

**A limit reaches exactly as far as your cache does.**

| Cache store | A limit counts across |
|---|---|
| `memory` | one process |
| `file` | one host |
| `redis` | the cluster |

On the memory store with three instances behind a load balancer, a limit of 5
admits 15. Use Redis for any limit that must hold across instances.

**A fixed window admits up to twice the limit across a boundary.** A caller
can spend the whole limit at the end of one window and the whole limit again at
the start of the next. For sign-in and registration this does not matter. If
you need a strict rate, put the limiter behind something that enforces one.

**Not implemented:** sliding windows, token buckets, per-route configuration
files.

## Behaviour when the cache is down

If the cache cannot be reached, the request is **allowed** and the failure is
logged. A Redis outage therefore degrades the limiter rather than taking down
sign-in, registration and every limited action with it.

`Allow` returns the error rather than swallowing it, so a caller that cannot
accept failing open can check it and refuse:

```go
ok, _, err := ratelimit.Allow(c.RequestContext(), key, 10, time.Minute)
if err != nil {
    // Fail closed instead: refuse rather than admit an uncounted request.
    return c.Error(http.StatusServiceUnavailable, err)
}
```

The error is `ratelimit.ErrNoCache` when no cache is configured at all, which
is a wiring mistake rather than an outage.

## How a window is stored

Each counter is seeded with `cache.Add(ctx, key, 0, window)`, which writes
only if the key is absent, and then incremented. `cache.Increment` takes no
TTL, so seeding is what gives the window an expiry — and because the seed only
lands once, the window rolls rather than extending with every hit.

Every key is stored under the `ratelimit:` prefix (`KeyPrefix`), so flushing
the limiter cannot flush anything else.

## Next steps

- [Cache](/docs/cache/) — the store the counters live in, and how to point it
  at Redis.
- [Authentication](/docs/security/authentication) — the routes most worth
  limiting.
