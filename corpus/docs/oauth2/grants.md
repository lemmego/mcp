---
title: Grants
type: docs
prev: docs/oauth2/clients
next: docs/oauth2/protecting-routes
weight: 144
sidebar:
  open: true
---

## Authorization code with PKCE

The flow for an application acting on behalf of a user.

### 1. Send the user to the authorization endpoint

The client generates a random `code_verifier`, derives its S256 challenge, and
redirects:

```text
GET /oauth/authorize
  ?response_type=code
  &client_id=Eg7w1IXAhqieDRJ5cHRY6A
  &redirect_uri=https://app.example.com/callback
  &scope=orders:read
  &state=<random, checked by the client on return>
  &code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM
  &code_challenge_method=S256
```

A signed-out visitor is redirected to `login_route` with a `redirect`
parameter, and lands back here afterwards.

Only `S256` is accepted. RFC 7636 also defines `plain`, and makes it the
default when `code_challenge_method` is omitted — so an omitted method is
rejected rather than treated as "no PKCE requested".

### 2. The user approves

They see [the consent screen](/docs/oauth2/consent-screen) and approve or
refuse. Either way they are redirected back:

```text
https://app.example.com/callback?code=<code>&state=xyz
https://app.example.com/callback?error=access_denied&state=xyz
```

{{< callout type="info" >}}
Errors discovered *before* `redirect_uri` is validated — an unknown
`client_id`, a redirect that matches nothing — are rendered as a page and not
redirected anywhere. Redirecting them would make the authorization endpoint an
open redirector, which is what RFC 6749 §4.1.2.1 is about.
{{< /callout >}}

### 3. Exchange the code

```bash
curl -u "$CLIENT_ID:$CLIENT_SECRET" \
  -d grant_type=authorization_code \
  -d code="$CODE" \
  -d redirect_uri=https://app.example.com/callback \
  -d code_verifier="$VERIFIER" \
  https://auth.example.com/oauth/token
```

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "5Nc8TQ...",
  "scope": "orders:read"
}
```

A code is valid for 60 seconds and is single use.

### What the exchange checks

- The code has not been used. A second attempt is refused **and revokes
  everything issued from that code** — a replay means the code leaked, so the
  tokens from the first, apparently legitimate exchange are no longer
  trustworthy. RFC 6749 §4.1.2 requires the denial and recommends the
  revocation.
- The client redeeming it is the client it was issued to. Otherwise one
  application could redeem another's code.
- `redirect_uri` is the one bound to the code, so a code stolen through one
  redirect cannot be redeemed through another.
- The `code_verifier` derives to the stored challenge, compared in constant
  time.

{{< callout type="warning" >}}
The code is consumed *before* those checks, so a failed check still spends it.
That is intentional: an attacker holding a stolen code but not the verifier
gets one attempt, not an unlimited number of guesses.
{{< /callout >}}

## Refresh token

```bash
curl -u "$CLIENT_ID:$CLIENT_SECRET" \
  -d grant_type=refresh_token \
  -d refresh_token="$REFRESH_TOKEN" \
  https://auth.example.com/oauth/token
```

The presented token is spent and a new one returned. **Store the new one** —
the old one no longer works.

Presenting a spent token revokes every token from the same authorization,
including the successor the honest client holds. One of the two holders is an
attacker and the server cannot tell which, so both must authenticate again
(RFC 6819 §5.2.2.3).

Scopes may be narrowed on refresh and never widened:

```bash
-d scope="orders:read"     # narrower than granted: allowed
-d scope="orders:write"    # not granted originally: invalid_scope
```

## Client credentials

```bash
curl -u "$CLIENT_ID:$CLIENT_SECRET" \
  -d grant_type=client_credentials \
  -d scope="invoices:read" \
  https://auth.example.com/oauth/token
```

No user, so no refresh token. The token's `sub` is the client id, per RFC 9068
§2.2 — which is ambiguous on its own, so code asking "is there a user here?"
should use `Principal.IsClientCredentials()` rather than comparing claims.

## Device authorization

For a CLI, a TV app, or anything without a browser.

### 1. The device asks for a code

```bash
curl -u "$CLIENT_ID:$CLIENT_SECRET" \
  -d scope="profile" \
  https://auth.example.com/oauth/device/code
```

```json
{
  "device_code": "GmRhmhcxhw...",
  "user_code": "BCDF-GHJK",
  "verification_uri": "https://auth.example.com/oauth/device",
  "verification_uri_complete": "https://auth.example.com/oauth/device?user_code=BCDF-GHJK",
  "expires_in": 600,
  "interval": 5
}
```

The device shows the code and the URI — or a QR code pointing at
`verification_uri_complete`, so the user types nothing.

The user code is eight characters from a twenty-character alphabet with no
vowels, so it cannot spell a word, and none of the pairs people confuse
reading aloud: no `O` against `0`, no `I` or `L` against `1`. It is
case-insensitive, and the dash is optional.

### 2. The user enters it

They visit the verification URI, type the code, and approve. Attempts are rate
limited per address — a user code carries about 35 bits, which is not enough
to survive unlimited guessing, and RFC 8628 §5.1 requires the limit for
exactly that reason.

### 3. The device polls

```bash
curl -u "$CLIENT_ID:$CLIENT_SECRET" \
  -d grant_type=urn:ietf:params:oauth:grant-type:device_code \
  -d device_code="$DEVICE_CODE" \
  https://auth.example.com/oauth/token
```

| Response | Meaning |
|---|---|
| `authorization_pending` | Keep waiting |
| `slow_down` | Polling too fast — add five seconds and continue |
| `access_denied` | The user refused |
| `expired_token` | Ten minutes passed; start again |
| a token | Done |

`slow_down` is enforced, not merely advised: polling early raises the stored
interval by five seconds, so a device that ignores the response finds the
window growing.

## Personal access tokens

Not an HTTP grant — there is no client to authenticate and no redirect to
follow. Issue one from your own settings page:

```go
response, token, err := server.CreatePersonalAccessToken(
    ctx, userID, "CI deploy key", oauth2.ParseScopes("deploy:write"),
)
// response.AccessToken is shown to the user once
// token.ID is the jti, which is what revokes it later
```

Default lifetime is a year, and no refresh token is issued. The name is stored
on the row so a user can recognise which token to revoke.

## Next steps

- [Protecting routes](/docs/oauth2/protecting-routes) — checking the token you
  just issued.
- [Tokens and keys](/docs/oauth2/tokens-and-keys) — what is inside one, and how
  it dies.
- [Consent screen](/docs/oauth2/consent-screen) — the screen the code flow goes
  through.
