## When to use this skill

When capping how often a caller can do something. Use it on anything a stranger
can reach — sign-in, registration, password reset — and on signed-in actions
that are cheap to repeat, such as voting or commenting.

A fixed-window counter stored in the application's cache. There is no provider
to register: the counters live in `lemmego/cache` and the middleware is a plain
function.

## On a route, or anywhere

    r.Post("/login", ratelimit.Middleware(ratelimit.PerIP(5, time.Minute)), handlers.Login)

    ok, retryAfter, err := ratelimit.Allow(ctx, "vote:"+userID, 30, time.Minute)

`Allow` is for anything not serving HTTP — a queued job, a CLI command.

## `X-Forwarded-For` is ignored until you say otherwise

    ratelimit.TrustProxy(true)   // only when a proxy you control sets the header

`ClientIP` reads `X-Forwarded-For` **only** after this is enabled, because the
header is client-supplied. Trusting it unconditionally lets anyone bypass an IP
limit by sending a different value on every request — **which is worse than
having no limiter, because it also looks like one is working.**

Enable it only when something you control overwrites the header at the edge. If
a limit is not biting in production behind a load balancer, this is the first
thing to check; if it is not biting locally, this is the wrong thing to change.

## Pick the key deliberately

`PerIP` is right for an unauthenticated route. For a signed-in action key on
the user — `Caller` does that — because an IP is shared by an office and a
mobile carrier, and an IP limit on a signed-in action rate-limits a building.
`Route` scopes a limit to one path.

Keys are namespaced with `KeyPrefix` (`ratelimit:`), so a cache flush by tag
does not take counters with it and a counter cannot collide with an
application key.

## A refused request answers 429 with `Retry-After`

    HTTP/1.1 429 Too Many Requests
    Retry-After: 43

`Retry-After` is why a client that retries does not retry immediately. Without
it one refused request becomes a tight loop, and the limiter turns into the load
it was meant to prevent. If you write your own refusal response, keep the
header.

## It fails open

If the cache is unreachable, `Allow` returns `true` and logs. The alternative is
a Redis outage becoming a total outage: every sign-in, every registration, every
vote refused because a counter could not be read.

That is the right trade for what a limiter usually guards. **If allowing the
request is unacceptable, check the error** — it is returned, not swallowed.
`ErrNoCache` means no cache is configured at all.

## The limit is approximate at a boundary

A fixed window allows up to twice the limit across a window boundary. If you
need a strict rate, put the limiter behind something that enforces one.

## Check first

`search-docs` section `ratelimit`, `module-api` on `ratelimit`.
