---
title: Clients
type: docs
prev: docs/oauth2/configuration
next: docs/oauth2/grants
weight: 143
sidebar:
  open: true
---

A client is an application allowed to ask for tokens. Which kind you register
decides how it authenticates and which grants it may use.

## The management page

A signed-in user manages their own clients at `{route_prefix}/clients`:
registering one, seeing what they have, and revoking it. It is a plain Go
template with no JavaScript, in the same style as the consent screen, so it
works whatever the application's frontend is.

It lists **their own** clients and nobody else's. That is what makes it safe
without an administrator role — a page showing every client would leak the
existence of every integration to whoever reached it. A client belonging to
someone else is reported as "No such client" rather than "forbidden", so the
page cannot be used to find out which client ids exist.

A secret is shown exactly once, immediately after registration. It is stored
hashed, so a refresh genuinely cannot recover it.

{{< callout type="info" >}}
The page is a typed route, so it inherits the framework's CSRF protection and
session — unlike the token endpoint, which is a raw handler precisely so that
CSRF does not apply to it. It carries its own anti-forgery token as well,
because the REST preset installs no CSRF middleware at all and a page that can
register an OAuth client is not one to leave unprotected.
{{< /callout >}}

Set `management_routes` to `false` to turn it off, or supply `ClientsView` on
the provider to render it yourself.

## The five kinds

### Confidential — the default

A server-side application that can keep a secret.

```bash
lemmego run oauth:client \
  --name "Acme Dashboard" \
  --redirect-uri https://acme.example.com/callback
```

Authenticates with its id and secret, by HTTP Basic or in the request body.
Uses the authorization code grant with PKCE.

### Public

A single-page application or a mobile app. Its code ships to the user, so it
has no secret to keep.

```bash
lemmego run oauth:client --public \
  --name "Acme Mobile" \
  --redirect-uri com.acme.app:/oauth2redirect
```

Authenticates with its id alone, which is not a secret — so **PKCE is what
binds an authorization code to the application that requested it**, and is
required regardless of `require_pkce`. Sending a `client_secret` is rejected
rather than ignored, because it means the caller is confused about which
client it is.

The redirect above is a *private-use URI scheme*, which RFC 8252 §7.1
recommends for native applications: a reverse-domain scheme the operating
system routes back to the app. A loopback redirect such as
`http://127.0.0.1:1234/callback` also works, and its port is allowed to differ
at request time because a native app cannot know which port it will be given.

### Machine to machine

No user is involved; the application acts as itself.

```bash
lemmego run oauth:client --client-credentials \
  --name "Billing Sync" --scopes invoices:read,invoices:write
```

Must be confidential — a public client has no secret, so there would be
nothing to authenticate. No refresh token is issued, per RFC 6749 §4.4.3: the
client can always ask for another token with the credentials it already holds,
so a refresh token would be a second credential with no extra capability and
one more thing to leak.

### Device

For anything that cannot open a browser.

```bash
lemmego run oauth:client --device --name "Acme CLI"
```

See [the device grant](/docs/oauth2/grants#device-authorization).

### Personal access

The client personal access tokens are issued against. There is exactly one.

```bash
lemmego run oauth:client --personal --name "Personal Access Client"
```

{{< callout type="info" >}}
Passport keeps a whole `oauth_personal_access_clients` table to record which
client this is. Lemmego gives it a fixed id instead, so a primary-key lookup
answers the same question with one fewer table and one fewer way to end up
inconsistent. That is why the schema has five tables where Passport has six.
{{< /callout >}}

## First-party clients

```bash
lemmego run oauth:client --first-party --name "Acme Web" --redirect-uri ...
```

A client your own application owns. Consent is skipped for it — there is
nobody to ask — and it is the only kind that may request the `*` scope.

## Redirect URIs

A redirect is matched by **exact string equality**. Not a prefix, not a
subpath, no wildcards.

That is stricter than it may look, and it is deliberate: every relaxation of
this rule has been the root cause of a real token theft, and RFC 9700 §2.1
says simple string comparison. These all fail against a registered
`https://app.example.com/callback`:

```text
https://app.example.com/callback/        a trailing slash
https://app.example.com/callback?x=1     an added query
https://app.example.com/callback/extra   a subpath
https://app.example.com.evil.test/cb     a suffixed host
https://app.example.com@evil.test/cb     userinfo that reads as the host
```

The only exception is a loopback address, where the port may differ.

Registration itself refuses a URI carrying a fragment, one with userinfo, a
relative URI, and plaintext `http` to anything but a loopback address.

{{< callout type="warning" >}}
`http://localhost:1234/callback` is **not** treated as loopback.
`localhost` resolves through the resolver and can be pointed elsewhere by a
hosts file or a DNS answer; `127.0.0.1` and `[::1]` cannot. RFC 8252 §8.3 says
to use the literal addresses.
{{< /callout >}}

## Limiting a client's scopes

```bash
lemmego run oauth:client --name "Reports" --scopes reports:read --redirect-uri ...
```

A client with a scope list may never be granted anything outside it, whatever
it asks for. Leave it empty and the client may request any registered scope —
subject, as always, to the user agreeing.

## Secrets

A client secret is 32 bytes from `crypto/rand`, stored as its SHA-256 digest.

{{< callout type="info" >}}
A user's password is hashed with bcrypt; a client secret is not. The
difference is that a password is human-chosen and low-entropy, so the work
factor is what makes a dictionary attack expensive. A client secret has no
dictionary — there is nothing to slow down — while cost-10 bcrypt would add
tens of milliseconds to every request at an unauthenticated endpoint, which is
both a throughput problem and an amplification vector. bcrypt also silently
truncates at 72 bytes.
{{< /callout >}}

Comparison is constant-time, and a client that does not exist performs the
same work as one with a wrong secret, so the endpoint cannot be used to learn
which client ids are real.

## Revoking a client

```go
store.RevokeClient(ctx, clientID, time.Now())
```

Revoking cascades to every token the client holds, at revoke time rather than
on each later request — so an administrator pays the cost once instead of
every request paying it forever.

## Next steps

- [Grants](/docs/oauth2/grants) — the flow your client will use.
- [Consent screen](/docs/oauth2/consent-screen) — what the user sees, and
  skipping it.
