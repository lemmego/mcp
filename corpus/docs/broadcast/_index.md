---
title: Broadcasting
type: docs
prev: docs/notifications/
next: docs/filesystem/
weight: 129
sidebar:
  open: true
---

`lemmego/broadcast` pushes events to connected clients over a live connection.
Use it for anything the browser should learn about without asking: an unread
badge, a new message, a job that has finished.

Server-sent events and WebSocket sit behind one contract, so the transport is
a configuration choice rather than a rewrite.

## Install

```shell
go get github.com/lemmego/broadcast
```

```go
&provider.Provider{
    AllowedOrigins: []string{"https://example.com"},
}
```

`AllowedOrigins` is required — see [Origins](#origins).

## Publish an event

```go
event, err := broadcast.NewEvent("notification.created", map[string]any{"id": 7})
hub.Publish(ctx, broadcast.UserChannel("42"), event)
```

The channel `private-user.{subject}` is installed for you, so
`broadcast.UserChannel(id)` works with no configuration. It is the channel
[`lemmego/notify`](/docs/notifications/) publishes to.

## Subscribe from the browser

```tsx
useBroadcast(`private-user.${viewer.id}`, {
    "notification.created": () => refetchUnread(),
}, { onResync: refetchUnread });
```

## Authorize a private channel

Register a pattern with a callback that decides whether this request may
subscribe:

```go
&provider.Provider{
    AllowedOrigins: []string{"https://example.com"},
}.
Channel("private-post.{id}", func(c app.Context, p provider.Params) error {
    user, ok := auth.UserAs[*models.User](c)
    if !ok {
        return broadcast.ErrUnauthenticated
    }
    return policies.Post(c).ViewRealtime(user, p.Get("id"))
})
```

The callback runs **before the connection is established**, and again for any
channel added during the connection's life. A refusal is returned as an HTTP
status, so the client sees a failed request rather than a connection that
opens and immediately closes — which a browser cannot distinguish from a
flaky network, and retries forever.

Patterns match segment by segment. They are not regular expressions, so a
pattern cannot accidentally match more than it names.

## Choosing a transport

| | `sse` | `ws` |
|---|---|---|
| Client → server messages | no — reconnects to change channels | yes |
| Liveness detection | a write can succeed long after the peer is gone | a ping is answered |
| Dependencies | none | `coder/websocket` |
| Browser connection cap | one of ~6 per origin on HTTP/1.1 | ~255 |

**Use `sse`** when a proxy such as nginx or Caddy terminates HTTP/2 in front
of your application. It costs one goroutine instead of two, adds no
dependency, and the browser's own reconnect handles most failures.

**Use `ws`** when you serve HTTP/1.1 directly — a single binary with no proxy.
Without HTTP/2 multiplexing, an event stream permanently occupies one of the
browser's ~6 connections per origin, shared across every open tab.

## Origins

Both transports authenticate from a **cookie**, because neither a browser
`EventSource` nor a browser `WebSocket` can set an `Authorization` header.
That makes the origin check the control that protects them, and the two
protocols are not equally exposed:

- **A WebSocket is not subject to CORS.** Any page can open one and the
  browser will attach your cookies unless `Origin` is checked. The `ws`
  transport always checks it; there is no way to turn it off.
- **An event stream is subject to CORS**, so it is same-origin by default.
  This module sends no CORS headers.

{{< callout type="warning" >}}
An unset `AllowedOrigins` is the difference between a private channel and a
public one, so it is a boot error rather than a permissive default.

Entries may be a full URL (`https://example.com`) or a bare host
(`example.com`). A WebSocket handshake is matched against the host, so the
scheme is stripped before the comparison.
{{< /callout >}}

Two rules differ between the transports, and the WebSocket side is stricter in
both:

| Value | `sse` | `ws` |
|---|---|---|
| `[]string{"*"}` | any origin | **same-origin only** |
| `*.example.com` | not matched — the check is an exact comparison | matched |

Write hosts out in full if you need the same configuration to behave
identically on both.

## Delivery guarantees

**`Publish` means "accepted for delivery", not "delivered".** Delivery is
at-least-once, so every event carries `Event.ID` and the client deduplicates
on it.

**There is no backlog.** `Last-Event-ID` is not honoured, and the stream does
not emit `id:` at all rather than inviting the browser to send a header the
server cannot act on. Instead, every ready frame carries `resync: true` and
the client's `onResync` callback fires, so your application re-fetches current
state over HTTP. This is also the path a laptop waking from sleep needs.

**A slow client is disconnected rather than buffered.** An unbounded queue
would be a memory leak one slow client wide, and silently dropping frames
would leave a client wrong with no way to find out. A closed connection is
something the client already handles.

**A lost bus subscription closes the connections that depend on it**, because
an open stream that will never receive anything is otherwise
indistinguishable from a quiet one.

## Buses

The **in-process bus** is the default. A single binary with no Redis is a
supported configuration.

The **Redis bus** is for running more than one replica:

```go
bus, err := redisbus.New(redisbus.Config{
    Addr:   "127.0.0.1:6379",
    Prefix: "myapp",
})
if err != nil {
    return err
}

&provider.Provider{
    AllowedOrigins: []string{"https://example.com"},
    Bus:            bus,
}
```

`Config` also takes `Client` (an existing `goredis.UniversalClient`),
`Username`, `Password`, `DB`, `PoolSize`, `Mode` and `Logger`. Use `BusFor`
instead of `Bus` when the bus needs something from the container that is not
available where providers are declared.

Two requirements:

- **`Prefix` is required.** Redis pub/sub ignores the selected database, so
  two applications sharing one Redis without a prefix deliver each other's
  events.
- **Give it its own client.** Do not pass the one your cache uses. go-redis
  dedicates a connection to each subscription for its lifetime, so sharing
  shows up as the *cache* blocking on reads.

## Capacity planning

About **10,000 connections per process** is comfortable: 16–24KB each for an
event stream, 32–40KB for a WebSocket.

Redis becomes the constraint before HTTP does, which is why
`PatternSubscribe` is the default — one Redis subscription per process with
routing done locally, rather than one Redis connection per connected user.

Beyond that, in order: a dedicated fleet of connection-holding processes;
sharded channels, so Redis carries O(shards) rather than O(users); then Redis
Streams with consumer groups, which also provides the backlog that would make
true resume possible.

## Limitations

Presence channels · arbitrary client-to-server application messages (the
protocol admits three control commands; anything your application *does*
should go through an HTTP route, which already has CSRF, validation and
authorization) · a message backlog and true resume · guaranteed delivery and
acknowledgements · per-event filtering finer than the channel · compression ·
long-polling.

## Next steps

- [Notifications](/docs/notifications/) — the module that publishes here.
- [Deployment](/docs/deployment/#reverse-proxy) — proxy settings a live
  connection needs.
