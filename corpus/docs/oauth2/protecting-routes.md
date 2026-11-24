---
title: Protecting routes
type: docs
prev: docs/oauth2/grants
next: docs/oauth2/consent-screen
weight: 145
sidebar:
  open: true
---

## Requiring a token

```go
provider := &oauth2.Provider{}

api := r.Group("/api")
api.UseBefore(provider.Protect())
{
    api.Get("/me", handlers.Me)
}
```

## Requiring scopes

```go
orders := r.Group("/api/orders")
orders.UseBefore(provider.Protect("orders:read"))
```

A token without every named scope gets `403` with
`{"error": "insufficient_scope"}` — not `401`. The distinction matters to a
client: the token is valid and re-authenticating will not help, so it needs a
*different* token rather than a fresh one. RFC 6750 §3.1.

{{< callout type="warning" >}}
Register middleware on a group **before** its routes. A group snapshots its
middleware into each route as the route is added, so a `UseBefore` that comes
after a `Get` silently does nothing.
{{< /callout >}}

## Reading the token

```go
func Me(c app.Context) error {
    principal, ok := oauth2.PrincipalFrom(c)
    if !ok {
        return c.Unauthorized(errors.New("no token"))
    }

    return c.JSON(app.M{
        "user":   principal.UserID,     // empty for client credentials
        "client": principal.ClientID,
        "scopes": principal.Scopes,
    })
}
```

| Method | |
|---|---|
| `HasScope("x")` | One scope |
| `HasAllScopes("x", "y")` | Every one |
| `HasAnyScope("x", "y")` | At least one |
| `IsClientCredentials()` | No user is involved |

Use `IsClientCredentials()` rather than checking `sub` against `client_id`
yourself. For a machine token the two are equal by design, and reading the
claim invites treating a client id as a user id.

## Working with the auth package

`Protect` does not resolve users itself. It establishes the identity and hands
it to `auth`, which loads the row through the application's own `UserLoader` —
the same one a session cookie or a bearer JWT goes through. So a handler
written for either of those works unchanged behind a token:

```go
user, ok := auth.UserAs[*models.User](c)   // the same type on every path
```

For a **client-credentials** token there is no user, and `ok` is false.
`auth.IsAuthenticated(c)` is still true: the caller is verified, it is simply
not a person. Reach for the client through `PrincipalFrom` instead.

{{< callout type="info" >}}
The dependency only ever points one way. `oauth2` calls `auth.SetSubject` for
a user token and `auth.SetAuthenticated` for a machine one; `auth` knows
nothing about OAuth2. That seam is public, so any guard that verifies an
identity some other way — mTLS, an upstream proxy header — plugs in the same
way and produces the same type. See
[The Current User](/docs/security/current-user).
{{< /callout >}}

{{< callout type="warning" >}}
Upgrading from `oauth2` v0.1.x? `Provider.UserResolver` was replaced by
`auth.Opts.UserLoader` — see
[Upgrade notes](/docs/appendix/upgrading#oauth2-v020--one-user-loader).
{{< /callout >}}

## Combining with session authentication

A route can accept either a session cookie or a bearer token:

```go
api.UseBefore(provider.Protect(), auth.Protected)
```

`Protect` establishes the subject from a token when one is present;
`auth.Protected` then loads it, or falls back to the session. Either way the
handler sees one type.

## Checking a token from elsewhere

Another service can verify a token without calling this one, using the
published keys:

```text
GET /.well-known/jwks.json
```

Verify RS256 against the key matching the token's `kid`, and check `iss`,
`aud` and `exp`.

{{< callout type="warning" >}}
Offline verification cannot see revocation. A token revoked a second ago still
verifies until it expires, because nothing about the signature changes. Where
that matters, call the introspection endpoint instead — it reads the
revocation row:

```bash
curl -u "$CLIENT_ID:$CLIENT_SECRET" \
  -d token="$ACCESS_TOKEN" \
  https://auth.example.com/oauth/introspect
```

Introspection is limited to confidential clients, so a browser cannot use it
to probe tokens it was never issued.
{{< /callout >}}

## Next steps

- [Tokens and keys](/docs/oauth2/tokens-and-keys) — what the middleware is
  verifying.
- [Authorization](/docs/authz/) — policies, once the token says who it is.
- [Authentication](/docs/security/authentication) — combining tokens with
  session sign-in.
