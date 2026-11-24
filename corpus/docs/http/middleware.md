---
title: Middleware
type: docs
prev: docs/http/routing
next: docs/http/request-handling
weight: 32
sidebar:
  open: true
---

**Middleware** is a function that sits between an arriving request and the
handler that answers it. It sees the request first, and it decides one of two
things: wave it through, or answer it itself and stop there.

That is the whole idea. It earns its place because some questions get asked by
a lot of routes — "is this person signed in?" is the same check on fifty
pages. Write it once as middleware and each route opts in by listing it.

Chain a few together and you have a series of checkpoints:

```text
request → log it → is it signed in? → is it under the rate limit? → handler → response
```

Any link can refuse, and if one does, the links after it and your handler
never run at all.

You get two layers of this, and they differ in *where* they sit:

1. **HTTP middleware** wraps the entire router. It runs before Lemmego has
   worked out which route matches, so it sees every request including ones
   that will 404.
2. **Application middleware** runs after a route has matched, and receives
   Lemmego's `Context` rather than raw `net/http` values.

Reach for the first when something must apply to absolutely everything —
recovering from a panic, logging a request. Reach for the second when your
check needs to know about your application: who is signed in, whether a form
token is valid, whether this person may edit this post.

Don't worry about setting either of these up to begin with: a project from
`lemmego new` already installs panic recovery, logging and CSRF for you.

## HTTP middleware

HTTP middleware uses the standard library's own shape,
`func(http.Handler) http.Handler` — a function that takes the next handler and
returns one wrapping it. That is deliberate: any middleware you already have
for `net/http`, or find in somebody else's library, works here unchanged.

These run first, before any routing happens.

### Built-in HTTP Middleware

```go
middleware.Recoverer()       // Panic recovery
middleware.RequestLogger()   // Request logging
middleware.MethodOverride    // PUT/DELETE via _method field
```

### Registration

HTTP middleware is registered in `bootstrap/http_middleware.go`:

```go
func LoadHTTPMiddlewares() []app.HTTPMiddleware {
    return []app.HTTPMiddleware{
        middleware.Recoverer(),
        middleware.RequestLogger(),
        middleware.MethodOverride,
    }
}
```

## App Middleware

App middleware uses `func(next Handler) Handler` and operates on the Lemmego `Context`.

### Built-in App Middleware

```go
middleware.VerifyCSRF(&middleware.CSRFOpts{})   // CSRF protection
```

### Registration

App middleware is registered in `bootstrap/middleware.go`:

```go
func LoadMiddlewares() []app.Handler {
    return []app.Handler{
        middleware.VerifyCSRF(&middleware.CSRFOpts{
            ExcludePatterns: []string{"/api/.*"},
        }),
    }
}
```

### Route-Scoped Middleware

Apply middleware to specific routes or groups:

```go
// Before middleware (runs before the handler)
r.UseBefore(auth.Protected)

// Per-route middleware — auth.Protected ships with the framework,
// AdminOnly is one you write yourself (see below)
api.Group("/admin").UseBefore(auth.Protected, middleware.AdminOnly)

// After middleware (runs after the handler)
r.UseAfter(middleware.AuditLog)
```

`auth.Protected` rejects unauthenticated requests, `auth.Guest` rejects authenticated
ones (for login/register routes), and `auth.OptionalAuth` populates the user when one
is present without ever blocking.

## Writing Custom Middleware

### App Middleware

```go
func RequestTimer(next app.Handler) app.Handler {
    return func(c app.Context) error {
        start := time.Now()
        err := next(c)
        duration := time.Since(start)
        slog.Info("request completed", "duration", duration)
        return err
    }
}
```

### HTTP Middleware

```go
func CORSMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        next.ServeHTTP(w, r)
    })
}
```

### Registration

```go
// bootstrap/middleware.go
func LoadMiddlewares() []app.Handler {
    return []app.Handler{
        RequestTimer,                    // Custom app middleware
        middleware.VerifyCSRF(nil),      // Built-in
    }
}

// bootstrap/http_middleware.go
func LoadHTTPMiddlewares() []app.HTTPMiddleware {
    return []app.HTTPMiddleware{
        CORSMiddleware,                  // Custom HTTP middleware
        middleware.Recoverer(),          // Built-in
    }
}
```

## Middleware Reference

### Recovery

Catches panics, logs the stack trace, and returns a 500 error. Supports development mode with stack traces in responses:

```go
middleware.Recovery()                                   // Default options
middleware.RecoveryWithOpts(middleware.DevelopmentRecoveryOpts)
middleware.RecoveryWithCustomFunc(customHandler)
```

### CSRF

Protects against cross-site request forgery using session-based tokens. Uses constant-time comparison and supports exclusion patterns:

```go
middleware.VerifyCSRF(&middleware.CSRFOpts{
    ExcludePatterns: []string{"/api/.*", "/webhook"},
})
```

The token is a per-session secret: it is stored in the session, mirrored into the `XSRF-TOKEN` cookie, and not rotated on each request. See [CSRF Protection](/docs/security/csrf).

### Method Override

Enables PUT, PATCH, and DELETE on HTML forms via `_method` field or `X-HTTP-Method-Override` header:

```go
middleware.MethodOverride
```

Usage in forms:

```html
<form method="POST" action="/tasks/1">
    <input type="hidden" name="_method" value="DELETE">
</form>
```

## Next steps

- [Request handling](/docs/http/request-handling) — reading and validating what
  the request carries.
- [Error handling](/docs/http/error-handling) — what to return when middleware
  turns a request away.
- [CSRF protection](/docs/security/csrf) — the middleware a scaffolded project
  already runs.
