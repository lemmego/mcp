---
title: Authentication
type: docs
prev: docs/security/
next: docs/security/current-user
weight: 101
sidebar:
  open: true
---

HTTP forgets you between requests. Each one arrives on its own, with no memory
of the last, so "being signed in" is something you have to arrange: the
visitor proves who they are once, and every later request carries a token
proving they already did.

That is **authentication** — working out *who* is asking. It is a different
question from what they are allowed to do, which is
[authorization](/docs/authz/) and lives elsewhere.

The `auth` package handles the first question. It accepts two kinds of proof —
a **session cookie**, which the browser sends automatically, and a **JWT**, a
signed token an API client sends in a header — and both end up handing your
handler the same user. Neither credential carries the user itself; both carry
an **id**, and `auth` reads the row back through a loader you supply.

[The current user](../current-user) covers what that means in your handlers;
this page covers configuring it.

## Setup

```go
func LoadProviders() []app.Provider {
    return []app.Provider{
        // The connector comes first: the loader below resolves a repository,
        // and a repository needs a connection.
        &ormconnector.Provider{},
        &auth.Provider{
            Opts: &auth.Opts{
                UserLoader: func(c app.Context, id string) (any, error) {
                    return repos.User(c.App()).FindByID(c.RequestContext(), id)
                },
                DisableSession: false,
                JwtSecret:      config.MustEnv("JWT_SECRET", ""),
                HomeRoute:      "/dashboard",
            },
        },
    }
}
```

`UserLoader` takes an `app.Context` rather than a plain `context.Context`
because `LoadProviders` runs before the service container is populated — there
is no `app.App` in scope to close over. `c.App()` resolves it per request.

Without a loader, `auth` can verify a credential but cannot produce a user.
Protected routes then answer **500**, not 401: a missing loader is a
misconfiguration, and answering 401 would send you looking for a bad password.

## The user model

Your user type implements `UserProvider`:

```go
type UserProvider interface {
    GetID() string
    GetUsername() string
    GetPassword() string
}
```

`GetID` returns a **string**, and that string is the identity — it becomes the
JWT's `sub` and the value stored in the session. `Login` refuses a user whose
`GetID()` is empty rather than minting a credential whose next request fails.

```go
type User struct {
    ID       uint64
    Name     string
    Email    string
    Password string
}

func (u *User) GetID() string       { return strconv.FormatUint(u.ID, 10) }
func (u *User) GetUsername() string { return u.Email }
func (u *User) GetPassword() string { return u.Password }
```

## The loader

```go
func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
    pk, err := strconv.ParseUint(id, 10, 64)
    if err != nil {
        return nil, fmt.Errorf("%w: %q is not a user id", auth.ErrUserNotFound, id)
    }
    user, err := r.db.Model[models.User]().Where(orm.Eq("id", pk)).First(ctx)
    if errors.Is(err, orm.ErrNotFound) {
        return nil, fmt.Errorf("%w: id %d", auth.ErrUserNotFound, pk)
    }
    return user, err
}
```

Two rules make the difference between a good day and a bad one:

**Return `ErrUserNotFound` for an id you cannot even parse.** It is tempting to
treat a malformed id as an error — it is malformed, after all. But the ids that
arrive malformed are the ones from credentials issued before you changed
something, and there can be a lot of them at once. `ErrUserNotFound` signs
those visitors out, one at a time, as they return. An error signs *nothing*
out and answers 503, which is an upgrade that looks exactly like an outage.

**Return the real error for anything else.** A timeout is not "no such user".
`auth` turns any other error into a **503** and leaves the session alone, so a
database blip is a blip. Reporting it as not-found would log out everyone who
happened to make a request during it.

`(nil, nil)` and a typed nil — `(*models.User)(nil)` — are both treated as
not-found. Neither authenticates.

## Login

```go
func login(c app.Context) error {
    user, err := repos.User(c.App()).FindByEmail(c.RequestContext(), input.Email)
    if err != nil {
        return c.Error(422, errors.New("those credentials do not match"))
    }

    result := auth.Login(c, user, input.Email, input.Password)
    if result.Err != nil {
        return c.Error(422, result.Err)
    }
    return c.JSON(app.M{"token": result.JwtToken})
}
```

You still fetch the row yourself, because you need it to check the password.
`Login` then writes only the id.

## Middleware

```go
r.Get("/dashboard", auth.Protected, handlers.Dashboard)    // 401 if anonymous
r.Get("/login", auth.Guest, handlers.LoginForm)            // redirects if signed in
r.Get("/profile", auth.OptionalAuth, handlers.Profile)     // populates, never blocks
```

`Protected` distinguishes three failures that used to look alike:

| Situation | Status |
|---|---|
| No credential, or one naming a user who no longer exists | 401 |
| The user store is unreachable | 503 |
| No `UserLoader` is configured | 500 |

A 401 also **revokes the credential** — the session is destroyed and the `jwt`
cookie expired — so a token naming a deleted user stops being presented rather
than failing a lookup on every request until it expires.

## Logout

```go
auth.Logout(c)
```

This is worth reading closely if you have configured `DisableSession: true`.
With no session there is nothing server-side to destroy, so logging out clears
the cookie and nothing more: **a copy of the token taken beforehand remains
valid until it expires.** There is no revocation. If that matters — and for a
browser client it usually does — keep the session on. `JwtSecret` still works
alongside it, so `/api/*` keeps accepting bearer tokens.

## What is in a JWT

```json
{ "sub": "42", "iat": 1790531597, "exp": 1790617997 }
```

A subject and two timestamps. Nothing else about the user, because a JWT is
signed, not encrypted: anything in it is readable by anyone holding it.

`Opts.JwtClaims` merges your own claims in, but it cannot set `sub`, `iat` or
`exp` — those are applied last and unconditionally. A configured `sub` used to
win, which was harmless while `sub` was decorative and became privilege
escalation the moment it was the identity.

## Choosing your credentials

```go
&auth.Opts{DisableSession: false, JwtSecret: ""}      // session only
&auth.Opts{DisableSession: false, JwtSecret: secret}  // both — the usual choice
&auth.Opts{DisableSession: true,  JwtSecret: secret}  // token only, no revocation
```

An application serving pages wants a session. An application serving only an
API has no cookie jar to renew and can reasonably go token-only, as long as
you have read the `Logout` note above.

## Next steps

- [The current user](/docs/security/current-user) — reading the signed-in user
  in a handler.
- [Session](/docs/session) — the store a login writes into.
- [Authorization](/docs/authz/) — what that user is then allowed to do.
