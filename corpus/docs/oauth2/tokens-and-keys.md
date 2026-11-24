---
title: Tokens and keys
type: docs
prev: docs/oauth2/consent-screen
next: docs/cli/
weight: 147
sidebar:
  open: true
---

## What an access token is

An RS256 JWT following RFC 9068, the JWT profile for OAuth 2.0 access tokens.

```json
{
  "alg": "RS256",
  "kid": "YvIdO6RXh5p8Dg--q_Tz8iEbUwTnmcWsdE9DrZgdiSU",
  "typ": "at+jwt"
}
```

```json
{
  "iss": "https://auth.example.com",
  "sub": "42",
  "aud": ["https://auth.example.com"],
  "client_id": "Eg7w1IXAhqieDRJ5cHRY6A",
  "jti": "nQtvXNiYbLL7I2o6PdzteeJrdKTtcgXRTr_hMqVj_lw",
  "scope": "orders:read",
  "iat": 1790439900,
  "nbf": 1790439900,
  "exp": 1790443500
}
```

`typ: at+jwt` is load-bearing. Without it, any other RS256 token your
application signs — an id token, something unrelated — could be presented here
as an access token.

For a client-credentials token `sub` is the client id, since there is no
resource owner.

## How revocation works

The `jti` is a row in `oauth_access_tokens`. The signature proves the token
was issued; the row proves it still counts.

Every authenticated request therefore costs one RSA verification and one
indexed primary-key read. That read is the cost of revocation meaning
what it says — a self-contained token cannot be un-issued, so something has to
be asked.

`revocation` trades it away if you decide to: see
[Configuration](/docs/oauth2/configuration#revocation--default-always).

Revoking a client cascades to its tokens at revoke time rather than on each
later request, so the cost is paid once by an administrator instead of forever
by every request.

## Key identifiers

The `kid` is the RFC 7638 JWK thumbprint — a SHA-256 over the key's own public
members — rather than a name someone chose.

Two things follow. The same key has the same `kid` in every process, with no
state to keep in sync. And a `kid` is only ever a lookup into a map of keys
already loaded: never a path, a filename or a query. A token claiming

```json
{ "kid": "../../storage/oauth/private.key" }
```

resolves to nothing and is rejected before anything touches a filesystem.

## Rotation

```bash
lemmego run oauth:keys --rotate
```

The active key moves to `storage/oauth/retired/<kid>.key` and a new one takes
its place. The retired key stays in the keyring and in the JWKS, so every
token it signed keeps verifying until it expires. Nothing breaks at the moment
of rotation.

Once no outstanding token could still be signed by it — an access-token
lifetime after rotation — the retired file can be deleted.

{{< callout type="warning" >}}
`oauth:keys` without `--rotate` refuses to overwrite an existing key, and that
refusal is deliberate. Replacing the signing key invalidates every access
token in flight and every refresh token ever issued, with no way back.
{{< /callout >}}

## The JWKS endpoint

```text
GET /.well-known/jwks.json
```

Public, cacheable for an hour, and carrying a strong `ETag` over the document
so a rotation invalidates caches immediately rather than leaving clients
unable to verify new tokens until a TTL expires.

Only public members are ever present. There is no field in the document type
that could carry a private component, and a test asserts that `d`, `p`, `q`,
`dp`, `dq` and `qi` never appear.

## Storing keys

The default is `storage/oauth/`, which the scaffold already ignores in git.
The private key is written `0600` inside a `0700` directory, and loading a
key that is group- or world-readable is refused in production — a key others
can read is one you have to assume is compromised.

For a read-only filesystem, supply the PEM directly:

```bash
OAUTH_PRIVATE_KEY="$(cat private.key)"
```

{{< callout type="warning" >}}
Every instance must hold the *same* key. Two instances that each generated
their own will reject each other's tokens, and the symptom — tokens working
intermittently, depending on which instance answers — is a confusing one to
diagnose.
{{< /callout >}}

## Refresh tokens

Not JWTs. A refresh token is 32 random bytes; only its SHA-256 is stored, so a
dump of the table cannot be exchanged for anything.

Each refresh spends the presented token and issues a new one. Presenting a
spent token revokes the whole chain — see
[Grants](/docs/oauth2/grants#refresh-token).

## Authorization and device codes

Also random and also stored hashed, with short lifetimes: 60 seconds for an
authorization code, ten minutes for a device code. Both are single use,
enforced by a conditional update rather than a read followed by a write, so
two simultaneous requests carrying the same code cannot both succeed.

## Next steps

- [Protecting routes](/docs/oauth2/protecting-routes) — verifying these tokens
  on your API.
- [Configuration](/docs/oauth2/configuration) — lifetimes and key settings.
- [Deployment](/docs/deployment/) — keeping keys out of your image.
