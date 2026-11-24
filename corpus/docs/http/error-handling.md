---
title: Error Handling
type: docs
prev: docs/http/response-handling
next: docs/http/context
weight: 35
sidebar:
  open: true
---

Sometimes you can't answer a request. The record isn't there, the visitor
isn't signed in, the input makes no sense. HTTP has a status code for each of
those, and it matters which one you send — a `404` and a `500` mean quite
different things to a browser, a crawler, a cache and your own monitoring.

The good news is that you don't have to build those responses yourself. Your
handler just **returns an error**, and the framework turns it into the right
status in the right format — an HTML error page for a browser, JSON for an API
client — in one place, the same way every time. That is why your handlers
return `error`.

## Typed HTTP errors

These are error values standing for a status code. Return one and the
framework knows what to send:

```go
app.ErrUnauthorized          // 401
app.ErrForbidden             // 403
app.ErrNotFound              // 404
app.ErrBadRequest            // 400
app.ErrMethodNotAllowed      // 405
app.ErrPageExpired           // 419
app.ErrUnprocessableEntity   // 422
app.ErrInternalServerError   // 500
```

Return them from handlers manually:

```go
func MyHandler(c app.Context) error {
    return app.ErrNotFound
}
```

## Context Error Helpers

The context provides convenience methods for common error responses:

```go
func MyHandler(c app.Context) error {
    return c.NotFound()
    return c.Unauthorized()
    return c.Forbidden()
    return c.BadRequest(err)
    return c.InternalServerError(err)
    return c.ValidationError(err)
    return c.PageExpired()
    return c.MethodNotAllowed()
    return c.UnprocessableEntity(err)
}
```

These methods automatically negotiate JSON vs HTML responses based on the request's `Accept` header.

## Error Map

The `ErrMap` lets you map error types to custom handler functions:

```go
// bootstrap/errmap.go
func LoadErrMap() app.ErrMap {
    return app.ErrMap{
        app.ErrNotFound: func(c app.Context) error {
            if c.WantsJSON() {
                return c.JSON(app.M{"error": "not found"})
            }
            return c.Render(res.NewTemplate(c, "404.page.gohtml"))
        },
        app.ErrUnauthorized: func(c app.Context) error {
            return c.Redirect("/login")
        },
        app.ErrInternalServerError: func(c app.Context) error {
            return c.Render(res.NewTemplate(c, "500.page.gohtml"))
        },
    }
}
```

The generated project includes default error pages for common status codes in `templates/`:
- `401.page.gohtml`
- `403.page.gohtml`
- `404.page.gohtml`
- `419.page.gohtml`
- `500.page.gohtml`

## Custom Error Types

Implement the `HttpError` interface for custom typed errors:

```go
type ValidationFailed struct {
    Field   string
    Message string
}

func (e *ValidationFailed) GetHttpMessage() app.HttpMessage {
    return app.HttpMessage{
        Status: http.StatusUnprocessableEntity,
        Message: e.Field + ": " + e.Message,
    }
}
```

## App-Level Error Handling

The framework's recovery middleware catches panics and returns appropriate responses:

```go
middleware.Recovery()                     // Production: log + 500
middleware.RecoveryWithOpts(middleware.DevelopmentRecoveryOpts) // Dev: log + stack trace
```

## Next steps

- [Context](/docs/http/context) — the error helpers on the request context.
- [Logging](/docs/logging) — where a failed request gets written down.
- [Request debugging](/docs/observe/) — reading back everything one request
  did.
