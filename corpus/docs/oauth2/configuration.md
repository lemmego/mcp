---
title: Configuration
type: docs
prev: docs/oauth2/installation
next: docs/oauth2/clients
weight: 142
sidebar:
  open: true
---

`lemmego run publish --tags=oauth2-config` writes `internal/configs/oauth.go`. Every
value reads through `config.MustEnv`, so a deployment overrides it with an
environment variable rather than by editing Go.

## Endpoints and identity

| Key | Variable | Default | |
|---|---|---|---|
| `route_prefix` | `OAUTH_ROUTE_PREFIX` | `/oauth` | Where the endpoints mount. The discovery document and the JWKS always stay at the root. |
| `issuer` | `OAUTH_ISSUER` | `APP_URL` | The `iss` claim of every token. |
| `login_route` | `OAUTH_LOGIN_ROUTE` | `/login` | Where a signed-out visitor to the consent screen is sent. |

## Keys

| Key | Variable | Default |
|---|---|---|
| `keys.path` | `OAUTH_KEYS_PATH` | `./storage/oauth` |
| `keys.length` | `OAUTH_KEY_LENGTH` | `2048` |

2048 is also the floor: a smaller key is refused both when generating and when
loading one generated elsewhere.

## Lifetimes

| Key | Variable | Default |
|---|---|---|
| `ttl.access_token` | `OAUTH_ACCESS_TOKEN_TTL` | `1h` |
| `ttl.refresh_token` | `OAUTH_REFRESH_TOKEN_TTL` | `336h` (14 days) |
| `ttl.auth_code` | `OAUTH_AUTH_CODE_TTL` | `60s` |
| `ttl.device_code` | `OAUTH_DEVICE_CODE_TTL` | `600s` |
| `ttl.personal_token` | `OAUTH_PERSONAL_TOKEN_TTL` | `8760h` (a year) |

These are durations and need a unit. A bare number is rejected with a message
saying so rather than being guessed at as seconds.

## Security settings

### `require_pkce` — default `true`

PKCE is required of every client, not only public ones. RFC 9700 recommends
it universally. A public client always requires it regardless of this setting,
because its client id is not a secret and PKCE is the only thing binding an
authorization code to the application that requested it.

### `refresh_rotation` — default `true`

Each refresh spends the presented token and issues a new one. Presenting a
spent token revokes **everything** issued from the same authorization,
including the token the honest client is holding.

That is deliberate, not an over-reaction: one of the two holders is an
attacker and the server cannot tell which, so both are made to authenticate
again.

Turning rotation off leaves the refresh token valid indefinitely and is
meaningfully weaker. It exists for clients that genuinely cannot store a new
token.

### `reuse_grace_period` — default `0s`

How long a rotated refresh token still works, for a client that crashes
between receiving a new token and saving it. Any non-zero value measurably
weakens reuse detection, because a stolen token is usable for that long
without triggering it. Passport has no such window either.

### `revocation` — default `always`

How much work each authenticated request does to find out whether a token has
been revoked.

| Value | Cost | Consequence |
|---|---|---|
| `always` | One indexed read per request | Revoking takes effect immediately |
| `cached` | One read per cache TTL | Reopens a revocation window that long |
| `never` | None | Revoking does nothing until the token expires |

`always` is the cost of revocation meaning what it says. Choose
another only if you have decided that trade knowingly.

### `management_routes` — default `true`

Mounts the page at `{route_prefix}/clients` where a signed-in user registers
and revokes their own clients. It lists only their own, so it needs no
administrator role. Turn it off if the application registers clients another
way or shows them in its own interface.

### `skip_consent_for_first_party` — default `true`

A client marked `--first-party` is one your own application owns, so there is
nobody to ask.

## Scopes

A scope must be registered before a client can request it. One that is not
registered is refused with `invalid_scope` rather than quietly dropped —
silently dropping it would leave a client believing it has access it was never
granted.

```go
"scopes": config.M{
    "orders:read":  "See your orders",
    "orders:write": "Place orders on your behalf",
    "profile":      "Read your profile",
},
```

or from code, which suits scopes that come from somewhere other than
configuration:

```go
server.TokensCan(map[string]string{
    "reports:export": "Export your reports",
})
```

The description is what a user reads on the consent screen, so write it as a
sentence addressed to them.

### The wildcard

`*` grants every registered scope, and **only a first-party client may request
it**. Passport allows any client to; a third-party application that can ask
for everything makes the consent screen a lie.

## CORS

| Key | Variable | Default |
|---|---|---|
| `cors_origins` | `OAUTH_CORS_ORIGINS` | empty |

A comma-separated list of origins allowed to call `/oauth/token` and
`/oauth/device/code` from a browser. Empty means none.

Credentials are never allowed on these endpoints: a browser-based client here
is a public client using PKCE, not one authenticating with a cookie.
`/oauth/introspect` gets no CORS at all, because RFC 7662 restricts it to
confidential clients and a browser is not one.

## Housekeeping

| Key | Variable | Default |
|---|---|---|
| `prune_after_hours` | `OAUTH_PRUNE_AFTER_HOURS` | `168` (a week) |
| `table_prefix` | `OAUTH_TABLE_PREFIX` | `oauth_` |

```bash
lemmego run oauth:purge --dry-run   # look first
lemmego run oauth:purge
```

Expired rows are not removed automatically. Run `oauth:purge` from cron, or
schedule it yourself — it is not wired into the queue, because many projects
turn the queue off and the server must not depend on it.

{{< callout type="warning" >}}
`table_prefix` is read at boot and cannot be changed after the tables exist
without migrating them. It is there for a project whose schema already has
something called `oauth_clients`.
{{< /callout >}}

## Next steps

- [Clients](/docs/oauth2/clients) — the five kinds and what each may do.
- [Tokens and keys](/docs/oauth2/tokens-and-keys) — rotation, JWKS and
  revocation.
