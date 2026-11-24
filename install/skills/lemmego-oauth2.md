## When to use this skill

When this application needs to **issue** OAuth2 tokens — act as the
authorization server that other applications authenticate against.

## It is a server, not a client

This is the most common misreading. If the goal is "let people sign in with
Google or GitHub", this is the wrong module: that is `auth/social/{github,google}`,
covered by the `lemmego-auth` skill. `oauth2` is Passport-shaped — *your*
application issues the tokens.

## Protecting an API

    r.Group("/api").UseBefore(oauth2Provider.Protect("orders:read"))

`Protect` sets both the `*Principal` and `auth`'s own user key, so a handler
already written against the current user keeps working against a bearer token
unchanged — and `auth` never learns this package exists, which keeps the
dependency pointing one way. Supply `UserResolver` to have it hand over a real
user row instead of the principal.

## The migration is yours

    lemmego publish --tags=migrations

That writes a real migration into the configured migration path, as ordinary
`migration.Create` calls. The file is then **yours** to edit: add a column,
widen a type, drop the device-code table if you will never use the device
grant. Nothing in the package reads it back.

What the server does expect is that the columns it reads and writes still exist
with compatible types. **Removing a table disables the grant that uses it;
removing a column the server writes fails at runtime rather than at migrate
time** — so a trimmed migration needs the matching grant disabled too.

## Client management and consent

A signed-in user manages their own clients at `{prefix}/clients`, and sees
**their own and nobody else's** — that ownership check is what makes the page
safe to expose. Both it and the consent screen are plain Go template pages with
no JavaScript, so they render in any preset including a pure API project.

A first-party client can skip the consent screen; a third-party one must not.

## Check first

`search-docs` section `oauth2` for the walkthrough, `module-api` for
signatures. The README documents what is deliberately missing, which is worth
reading before promising a grant type.
