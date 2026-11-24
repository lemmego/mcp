---
title: CSRF Protection
type: docs
prev: docs/security/current-user
next: docs/security/encryption
weight: 103
sidebar:
  open: true
---

Here is the attack. Someone signs in to your site, so their browser now holds
a session cookie for it. Later they visit a page somewhere else entirely, and
that page quietly submits a form to *your* application — `POST /account/email`,
say. The browser attaches your cookie, because browsers attach cookies by
origin regardless of who asked, and your server sees a perfectly valid request
from a signed-in user who never intended to make it.

That is **cross-site request forgery**, and the defence is to require
something on every state-changing request that the other site cannot know: a
random **token** your own pages include and an attacker's cannot read.

Lemmego does this with `middleware.VerifyCSRF`, and a scaffolded project
already has it installed. The usual way you meet it is a `419` when a form is
submitted without a valid token.

## How it works

1. The first request in a session generates a random token with `crypto/rand` and
   stores it in the session under `_token`. Later requests reuse it — the token
   lives as long as the session does.
2. Every response that passes verification mirrors that token into an
   `XSRF-TOKEN` cookie (skipping `/static`), and exposes it on the request
   context as `_token`.
3. Reads — GET, HEAD and OPTIONS — are not verified.
4. Every other method is verified: the middleware compares the token the request
   carried against the one in the session, using `subtle.ConstantTimeCompare`.
5. A mismatch returns `c.PageExpired()`, which is HTTP 419.

### The token is not rotated per request

A CSRF token is a shared secret for the session, not a one-time nonce, and it is
left in place after a successful write.

Rotating on every verified request rejects any client still holding the previous
token — one whose request was already in flight, one that missed a `Set-Cookie`,
a second tab, or a page restored from the back/forward cache. Each of those gets
a 419, and stays broken until a full page load. Laravel, Rails and Django all
keep a per-session token for the same reason.

The token does change when the session itself is regenerated.

## CSRF Token Sources

The middleware checks for the token in this order, stopping at the first hit:

1. The `X-XSRF-TOKEN` header
2. A `_token` POST form field
3. A `_token` form value, which also covers the query string
4. A `_token` field in a JSON body

## Configuration

```go
middleware.VerifyCSRF(&middleware.CSRFOpts{
    ExcludePatterns: []string{"/api/.*", "/webhook"},
})
```

Excluded patterns are evaluated as regular expressions. Requests matching any pattern skip CSRF verification entirely.

## In Templates

### Go Templates

`res.Template.Render` supplies `_token` from the request context, the same way
it supplies `errors`, so the handler does not have to pass it through:

```go
func TaskCreate(c app.Context) error {
    return c.Render(res.NewTemplate(c, "tasks.page.gohtml"))
}
```

```html
<form method="POST" action="/tasks">
    <input type="hidden" name="_token" value="{{ ._token }}">
</form>
```

An explicit `_token` in `WithData` still wins, if you need to override it.

### Templ

Generated projects ship a `csrf` component in `templates/csrf.templ` that reads
the token off the context:

```go
templ csrf() {
    if val, ok := ctx.Value("_token").(string); ok {
        <input type="hidden" name="_token" value={ val }/>
    }
}
```

Usage:

```go
<form method="POST" action="/tasks">
    @csrf()
</form>
```

### Inertia

CSRF tokens are automatically included in Inertia requests through the `XSRF-TOKEN` cookie.

## API Routes

For JSON API routes, exclude CSRF verification:

```go
middleware.VerifyCSRF(&middleware.CSRFOpts{
    ExcludePatterns: []string{"/api/.*"},
})
```

API clients should use token-based authentication (JWT or API keys) instead.

## Security Notes

- Tokens are generated using `crypto/rand`
- Token comparison uses `subtle.ConstantTimeCompare`, so it leaks nothing through timing
- The cookie carries the *same* value as the session token — that is the
  double-submit pattern. It is not `HttpOnly`, because the point is
  for JavaScript to read it and send it back in the `X-XSRF-TOKEN` header. The
  protection comes from the same-origin policy: a cross-site attacker can cause
  the cookie to be *sent*, but cannot *read* it to populate the header
- The cookie is `SameSite=Lax`, and `Secure` in production
- Excluded paths skip verification entirely, so exclude only routes that
  authenticate some other way

## Next steps

- [Session](/docs/session) — where the token is kept between requests.
- [Frontend](/docs/frontend/) — putting the token in your forms.
- [Testing](/docs/testing/) — getting a valid token in a test request.
