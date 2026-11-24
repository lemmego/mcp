## When to use this skill

When signing somebody in, reading the current user, protecting a route, or
debugging why an authenticated request is answering 401 or 503.

`auth` is **authentication only** — who is making this request. What they are
allowed to do is `authz` (per record) and `rbac` (roles and permissions).

## The UserLoader is the one thing you must write

This package queries nothing. A verified credential carries an identifier and
nothing else, so every authenticated request ends at your loader, and whatever
it returns is what `auth.UserAs` hands the handler — under a session cookie, a
JWT cookie and a bearer token alike.

It takes `app.Context` rather than `context.Context` because `LoadProviders()`
takes no arguments and runs before the container is populated, so a loader
declared there has no `app.App` to close over. `c.App()` is the route from the
declaration to your repository.

**The error contract matters more than the signature**, and getting it wrong
turns a stale credential into an outage:

| Situation | Return | Result |
|---|---|---|
| Found | the user | the handler runs |
| The id names nobody | an error satisfying `errors.Is(err, auth.ErrUserNotFound)` | 401, credential destroyed, next request anonymous |
| An id you cannot even parse | `ErrUserNotFound` — not a failure | same as above |
| The store could not answer | any other error | 503, the credential survives |

That last row is the one to get right. Reporting "could not answer" for an
unparsable id manufactures an outage; reporting `ErrUserNotFound` for a
database blip logs everybody out and then stampedes the sign-in page when they
all retry.

A nil user with a nil error, and a typed nil pointer, are both read as
`ErrUserNotFound`. Neither ever authenticates anybody.

## Reading the current user

    user, ok := auth.UserAs[*models.User](c)
    if !ok {
        // A credential that authenticates a machine and no user — a client
        // credentials token, for example — is authenticated but has no user.
        return c.Error(403, fmt.Errorf("this endpoint needs a user"))
    }

`ok` being false is not the same as unauthenticated. Do not answer 401 for it.

## Middleware

`auth.Protected` is the gate. Put it on the route or the group:

    r.Get("/me", auth.Protected, handlers.ProfileShowHandler)
    r.Group("/admin").UseBefore(auth.Protected)

Authentication first, then authorization — a 401 and a 403 are different
answers and the caller should get the right one.

## Social sign-in

`auth/social/{github,google}` handles the exchange. **The routes are yours to
mount**: the package does not register callbacks, because the paths are part of
the application's URL design and an OAuth redirect URI has to be registered
with the provider anyway.

## Before you write any of it

Run `module-api` on `auth` for exact signatures, and `search-docs` with section
`security/authentication` for the walkthrough. If you are reaching for roles or
permissions, stop and read the `lemmego-rbac` skill — do not model them as
columns on the user.
