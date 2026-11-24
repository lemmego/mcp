---
title: The Current User
type: docs
prev: docs/security/authentication
next: docs/security/csrf
weight: 102
sidebar:
  open: true
---

## The short version

```go
user, ok := auth.UserAs[*models.User](c)
```

This returns your own user type on every authenticated request, whether the
caller arrived with a session cookie, a bearer JWT, or an OAuth2 access token.
`ok` is false when nobody is signed in — and also when the caller is a machine
holding a client-credentials token, which is authenticated but is not a person.

If you only need the identifier, skip the lookup:

```go
id, ok := auth.UserID(c)
```

## How it resolves

A credential does not carry the user. It carries an **id**, which `auth`
resolves through the `UserLoader` you configured in
[Setup](../authentication#setup). One loader, one code path, one type,
whichever way the request authenticated — a session cookie, a JWT and an
OAuth2 bearer token all arrive in your handler as the same `*models.User`.

## What it costs

**One indexed lookup per authenticated request.**

What it buys is that the user is *current*. A token is a claim about the past —
it says who you were when it was minted. Resolving through storage means a
change to the row is visible on the very next request carrying the same
unchanged token:

```
UPDATE users SET name = 'Ada Lovelace' WHERE id = 42;
```

```console
$ curl /api/me -H "Authorization: Bearer $TOKEN"
{"user":{"id":42,"name":"Ada Lovelace", ...}}
```

The same goes for a deactivated or deleted account: refused on the next
request, not at expiry. With the user embedded in the token, neither is
possible.

The lookup happens at most once per request. `auth` memoises both the user and,
separately, a failure to load one — so a route group guard, `Protected` and
your handler do not make three trips, and a failing store does not make three
failing trips.

## Reading it

```go
func Dashboard(c app.Context) error {
    user, ok := auth.UserAs[*models.User](c)
    if !ok {
        return c.Error(403, errors.New("this page needs a user"))
    }
    return c.JSON(app.M{"user": user})
}
```

Behind `auth.Protected` the request is already authenticated, so a false `ok`
there means specifically "authenticated, but not as a person" — a machine
token. 403 is the correct answer; 401 would invite it to try authenticating
again, which it already did successfully.

Other accessors:

| Call | Answers |
|---|---|
| `auth.UserAs[T](c)` | the user as your type — what you want |
| `auth.UserID(c)` | the verified id, with no lookup |
| `auth.IsAuthenticated(c)` | is this request authenticated at all, user or not |
| `auth.AuthUser(c)` | the user as `any`, for code that cannot name `T` |

These are package-level functions rather than methods on `auth.Auth` on
purpose: an application can run the OAuth2 bearer guard without registering
the auth provider, and `auth.Get(a)` would panic there.

## Machine callers

An OAuth2 client-credentials token authenticates a client, and there is no
user behind it. `auth.IsAuthenticated(c)` is true, `auth.UserAs` is false, and
`auth.AuthUser(c)` is nil.

`Protect` used to put an `*oauth2.Principal` under the user key so that "the
current user" was never nil. It was never nil and frequently wrong, which is
worse. Reach for the client through OAuth2's own accessor:

```go
principal, ok := oauth2.PrincipalFrom(c)   // client, scopes, token id
```

## Writing your own guard

Anything that verifies an identity by some other route — an mTLS certificate,
a header from an upstream proxy, a signed webhook — hands `auth` the subject
and lets the loader do the rest. This is the same seam `oauth2.Protect` uses:

```go
func TrustedProxyAuth(c app.Context) error {
    id := c.Header("X-Authenticated-User")
    if id == "" {
        return c.Next()
    }
    auth.SetSubject(c, id)   // auth's loader runs; the type matches everywhere
    return c.Next()
}
```

Use `auth.SetAuthenticated(c)` instead when the caller is verified but is not
a user.

## Next steps

- [Authorization](/docs/authz/) — policies and gates over the user you just
  read.
- [Context](/docs/http/context) — the rest of the request-scoped API.
