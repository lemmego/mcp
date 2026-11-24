## When to use this skill

When pushing events to a connected client over SSE or WebSocket, or when a
private channel is letting the wrong person in.

## Pick the transport deliberately — they are not equally exposed

Both authenticate from a **cookie**, because neither a browser `EventSource`
nor a browser `WebSocket` can set an `Authorization` header. That makes the
origin check load-bearing:

- **A WebSocket is not subject to CORS.** Any page can open one and the browser
  attaches your cookies unless `Origin` is checked. The `ws` transport keeps
  the library's own check and there is **no way to turn it off**.
- **An event stream *is* subject to CORS**, so it is same-origin by default and
  this module sends no CORS headers at all.

Default to SSE. Reach for the WebSocket when the client genuinely needs to send
frames, not merely because it sounds more capable.

## `AllowedOrigins` is required

An unset allowlist is the difference between a private channel and a public
one, so it **fails the boot** rather than defaulting. Entries are a URL
(`https://app.example.com`) or a bare host (`app.example.com`) — a WebSocket
handshake matches on host, so the scheme is stripped before it gets there.

`[]string{"*"}` is honoured for an event stream and means same-origin; it is
stricter on the WebSocket side. Do not reach for `*` to make a local setup
work — name the host.

## Private channels

A channel named with the `private-` prefix goes through the `Authorizer`:

    broadcast.AuthorizerFunc(func(ctx context.Context, r *http.Request, ch broadcast.Channel) error {
        // Return nil to allow. Any error refuses.
    })

`DenyAll()` is the safe starting point and a legitimate production value for an
application with no private channels. Authorisation runs on **subscribe**, not
on publish, so a long-lived connection keeps whatever it was granted — revoke
by closing the connection, not by changing a permission row.

## Publishing

Events are published to a channel through the `Broadcaster`. Keep the payload
small: `MaxEventBytes` is 64 KB and `MaxChannelBytes` is 190, and an oversized
event is `ErrBadEvent` rather than a truncated frame.

An event is a **signal, not a data transfer**. Send "order 41 changed" and let
the client refetch; sending the whole order means every schema change is also a
wire-format change, and the client that renders from the event diverges from
the one that renders from the API.

## Buses

`NewMemoryBus()` is in-process: correct for one instance, wrong the moment
there are two, because a client connected to instance A never sees an event
published on instance B. Use the Redis bus for more than one process.
`ErrBusUnavailable` is what a degraded bus reports.

## What it does not do

Do not design around these: presence channels, arbitrary client-to-server
application messages, a message backlog or true resume, guaranteed delivery and
acks, per-event filtering beyond the channel, compression, long-polling.

Two of those have homes elsewhere. Anything the client wants to *do* goes
through an HTTP route, which already has CSRF, validation and authorization.
Guaranteed delivery is `tasker`'s job, and an in-app notification is already
durable in Postgres.

A heartbeat every 15 seconds keeps intermediaries from closing an idle stream.

## Check first

`search-docs` section `broadcast`, `module-api` on `broadcast`.
