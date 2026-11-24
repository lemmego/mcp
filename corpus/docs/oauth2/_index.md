---
title: OAuth2 Server
type: docs
prev: docs/localization/
next: docs/oauth2/installation
weight: 140
sidebar:
  open: true
---

{{< callout type="info" >}}
**Most applications never need this.** It is for letting *other* people's
applications call your API on your users' behalf. If you only want your own
users to sign in, [Authentication](/docs/security/authentication) is the page
you want.
{{< /callout >}}

**OAuth2** is how one application gets permission to use another on somebody's
behalf, without being told their password. You have been on the user's side of
it: an app asks to access your Google Drive, Google asks you whether that is
all right, and the app receives a **token** — a key that opens some of your
account, and only the parts you agreed to.

This module puts your application in Google's position in that story.

The `oauth2` module turns your application into an OAuth 2.0 **authorization
server**: it issues and revokes the tokens other applications use to call your
API, manages the clients that may ask for them, and runs the screen where a
user approves or refuses.

This is the server side — the equivalent of Laravel Passport. Signing your
users *in* to somebody else's provider, "log in with GitHub", is the client
side and is a different, much smaller thing this module does not do.

```go
// bootstrap/providers.go — below the database connector
&ormconnector.Provider{},
&oauth2.Provider{},
```

{{< cards >}}
{{< card link="installation" title="Installation" subtitle="Set it up in an existing project" >}}
{{< card link="configuration" title="Configuration" subtitle="Every key and environment variable" >}}
{{< card link="clients" title="Clients" subtitle="The five kinds, and how to register them" >}}
{{< card link="grants" title="Grants" subtitle="Authorization code, refresh, device and the rest" >}}
{{< card link="protecting-routes" title="Protecting routes" subtitle="Bearer tokens and scopes" >}}
{{< card link="consent-screen" title="Consent screen" subtitle="The default, and how to replace it" >}}
{{< card link="tokens-and-keys" title="Tokens and keys" subtitle="JWTs, JWKS, rotation, revocation" >}}
{{< /cards >}}

## What it supports

| Grant | Use |
|---|---|
| Authorization code + PKCE | A third-party web, mobile or single-page application acting for a user |
| Refresh token | Renewing access without asking the user again |
| Client credentials | A machine calling your API as itself, with no user involved |
| Device authorization | A CLI, a TV app, anything that cannot open a browser |
| Personal access tokens | A token a user creates for themselves, like an API key |

## What it does not support

**The implicit grant** returned tokens directly in a redirect, where they
landed in browser history and referrer headers. It is obsolete; the
authorization code grant with PKCE replaces it.

**The password grant** required the client application to handle your users'
passwords directly, which defeats the point of delegated authorization. RFC
9700 deprecates it.

Neither is disabled by a setting you could switch back on — they are not
implemented. The token endpoint names them in its error message so a client
that tries gets an answer rather than a puzzle:

```json
{
  "error": "unsupported_grant_type",
  "error_description": "the password grant is not supported; use the authorization code grant with PKCE"
}
```

## Standards

The implementation follows RFC 6749 (the core framework), RFC 6750 (bearer
token usage), RFC 7636 (PKCE), RFC 7009 (revocation), RFC 7638 (JWK
thumbprints), RFC 7662 (introspection), RFC 8414 (server metadata), RFC 8628
(the device grant), RFC 9068 (the JWT profile for access tokens), and the
current best practice in RFC 9700.

## Requirements

A database. Clients, authorization codes, refresh tokens and the revocation
list are all persisted, so a project scaffolded with `--database none` cannot
run the server — the provider says so at boot rather than failing later:

```text
oauth2: this application has no database, and an OAuth2 server cannot run without one.
```

It works with any SQL connector — the Lemmego ORM, GORM or Bun, with or
without GPA — because it resolves the connection through
[`api/db`](/docs/database/connection-seam) rather than through an ORM.

## Implementation notes

This is a hand-written implementation rather than a wrapper around a library
such as `ory/fosite`, which keeps the API Lemmego-shaped and avoids a second
framework's vocabulary in your code.

The trade-off is that a protocol mistake here is a security bug. The module
carries an adversarial test suite covering code replay, code and redirect
substitution, PKCE downgrade and stripping, refresh reuse, scope elevation,
`alg:none`, algorithm confusion, `kid` injection, `typ` confusion and a forged
`jti`. That suite proves those specific attacks are blocked. It does not prove
the absence of attacks, and it would be wrong to read it as doing so.

## Next steps

- [Installation](/docs/oauth2/installation) — install it and mount the
  endpoints.
- [Grants](/docs/oauth2/grants) — the five flows it issues tokens through.
- [Protecting routes](/docs/oauth2/protecting-routes) — requiring a token on
  your own API.
