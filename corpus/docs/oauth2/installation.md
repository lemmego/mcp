---
title: Installation
type: docs
prev: docs/oauth2/
next: docs/oauth2/configuration
weight: 141
sidebar:
  open: true
---

## Install

```bash
go get github.com/lemmego/oauth2
```

## Register the provider

In `bootstrap/providers.go`, **below the database connector**:

```go
func LoadProviders() []app.Provider {
    return []app.Provider{
        &fs.Provider{},
        &session.Provider{},
        &ormconnector.Provider{},   // or gormconnector / bunconnector
        &oauth2.Provider{},
        &auth.Provider{ /* ... */ },
    }
}
```

Order matters. Providers run one at a time in the order they are listed, and
each can only resolve what the ones before it registered. The OAuth2 provider
asks for the connection the database connector publishes, so listing it above
the connector leaves it with no database.

## Set it up

```bash
lemmego run oauth:install
```

That generates the RSA keypair tokens are signed with, writes it to
`storage/oauth`, adds it to `.gitignore` if it is not already covered, and
prints the remaining steps:

```bash
lemmego run publish --tags=oauth2-config,oauth2-migrations
go build ./...
lemmego run migrate up
```

{{< callout type="info" >}}
The tags are namespaced. `--tags` selects what gets published, so a
bare `migrations` would match every package that ships one, and `config`
already means something to the queue and cache modules. Passing no tags at all
publishes everything every registered provider offers.
{{< /callout >}}

{{< callout type="warning" >}}
`go build ./...` between publishing and migrating is not optional, and is the
step people miss. The migration file was written by a running binary that does
not contain it yet — `migrate up` will not see it until the project is
rebuilt.
{{< /callout >}}

## Create a client

```bash
lemmego run oauth:client --name "My App" --redirect-uri https://app.example.com/callback
```

```text
Client ID:     Eg7w1IXAhqieDRJ5cHRY6A
Client secret: JB9GxakmOueY1DgCdiJM2GPFamjdmS6FUmvxQ6EXtHI

This is the only time the secret can be shown; it is stored hashed.
```

The secret really is unrecoverable: only its SHA-256 is stored. If it is lost,
register a new client or update the existing one.

See [Clients](/docs/oauth2/clients) for the other kinds.

## Check what got mounted

```bash
lemmego run oauth:routes
```

```text
  GET  /oauth/authorize                          consent screen
  POST /oauth/authorize                          the user's decision
  POST /oauth/token                              token endpoint
  POST /oauth/device/code                        device authorization
  GET  /oauth/device                             user code entry
  POST /oauth/device                             device decision
  POST /oauth/revoke                             RFC 7009 revocation
  POST /oauth/introspect                         RFC 7662 introspection
  GET  /.well-known/jwks.json                    public keys
  GET  /.well-known/oauth-authorization-server   RFC 8414 metadata

Issuer: https://auth.example.com
Signing key: YvIdO6RXh5p8Dg--q_Tz8iEbUwTnmcWsdE9DrZgdiSU
```

The two `/.well-known/` documents always live at the site root, whatever
`route_prefix` is set to, because RFC 8414 requires it.

## Deploying

Two things need attention in production.

**The signing key.** `storage/oauth/private.key` is not in version control, so
it must reach the server another way — a mounted secret, a provisioning step,
or `OAUTH_PRIVATE_KEY` holding the PEM for a read-only filesystem. Two
instances with *different* keys will reject each other's tokens.

**The issuer.** `OAUTH_ISSUER` (falling back to `APP_URL`) becomes the `iss`
claim of every token and the issuer in the discovery document. A wrong value
breaks every consumer and is invisible until somebody integrates, so the
provider logs an error at boot if it is empty or points at localhost while in
production.

## Next steps

- [Configuration](/docs/oauth2/configuration) — endpoints, keys, lifetimes and
  scopes.
- [Clients](/docs/oauth2/clients) — creating the client that will ask for
  tokens.
- [Grants](/docs/oauth2/grants) — choosing the flow for your client.
