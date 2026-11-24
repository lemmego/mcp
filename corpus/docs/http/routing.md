---
title: Routing
type: docs
prev: docs/http/
next: docs/http/middleware
weight: 31
sidebar:
  open: true
---

## What a route is

When someone visits your site, their browser sends your application a short
message. Stripped of everything else, it comes down to two pieces of
information:

```text
GET /users/42
```

A **method** — `GET` — telling you what kind of action this is, and a
**path** — `/users/42` — telling you what it concerns. That is very nearly all
you are given, and from it you have to decide which of your Go functions
should answer.

A **route** is how you write down one of those decisions. It pairs a method
and a path with a function to run:

> `GET` + `/hello` → run this function

That is all a route is. Your whole application is, underneath, a list of these
pairings, and the **router** is what holds the list and finds the right one
when a request arrives. The function it picks is called a **handler**, because
it handles the request.

If nothing matches, there is no function to run, so your visitor gets a `404
Not Found`. That is not a bug in your code — just the absence of a line
covering that address.

### Methods

The method is a verb describing intent, and the convention is near-universal:

| Method | Means | Typically |
|---|---|---|
| `GET` | fetch something, change nothing | viewing a page |
| `POST` | create something | submitting a form |
| `PUT` | replace something | saving a whole record |
| `PATCH` | change part of something | editing one field |
| `DELETE` | remove something | deleting a record |

Nothing in HTTP stops you breaking these meanings — you *could* delete a
record from a `GET` route. But browsers, caches and crawlers all assume you
have not, and they will happily re-request a `GET` on their own. So keep
`GET` for reading, always.

## Writing routes

You'll write your routes in `internal/routes/`. Each method on the router
registers one:

```go
// internal/routes/web.go
package routes

func WebRoutes(a app.App) {
    r := a.Router()

    r.Get("/hello", func(c app.Context) error {
        return c.Text([]byte("World"))
    })

    r.Post("/submit", func(c app.Context) error {
        return c.JSON(app.M{"status": "ok"})
    })
}
```

Read that first one as: *when a `GET` arrives for `/hello`, run this
function.*

Your handler takes one argument, `c app.Context`, and returns an `error`. That
argument carries everything about the request, and it is also how you answer —
`c.Text` sends plain text, `c.JSON` sends JSON. Return `nil` and you are
saying the request is answered; return an error and the framework turns it
into an error page or a JSON error for you, so you don't have to build one.

`app.M` is just shorthand for `map[string]any`, so that second handler sends
`{"status":"ok"}`.

There is a method for every HTTP method: `Get`, `Post`, `Put`, `Patch`,
`Delete`, `Head`, `Options`, `Connect` and `Trace`.

## Route parameters

Most of your addresses won't be fixed. `/users/42` and `/users/43` are
different pages, and you obviously can't write a route for each one — you
don't know your users' ids yet, and there may be millions of them.

So a path pattern can have a **hole** in it, with a name:

```text
/users/{id}
```

That matches `/users/42`, `/users/43`, and `/users/` followed by anything
without a slash in it. Whatever lands in the hole — `42` — is the **route
parameter**, and the router keeps it for you under the name you chose. You
read it back with `c.Param("id")`:

```go
r.Get("/users/{id}", func(c app.Context) error {
    id := c.Param("id")
    return c.Text([]byte("User: " + id))
})
```

One route now serves every user, and the parameter tells your handler which
one was asked for.

Two things to keep in mind about that value:

- **You always get a string**, because a URL is text. `/users/42` hands you
  `"42"`, so converting it is up to you — use `strconv.Atoi`, which also tells
  you when somebody has asked for `/users/banana`.
- **Somebody else typed it.** Anyone can request `/users/999999` or
  `/users/'%20OR%201=1`, so treat it as input you need to check, never as a
  fact about your data.

You can have several parameters in one pattern, and each fills exactly one
segment — it stops at the next `/`:

```go
r.Get("/launch/{slug}/comments/{commentID}", handlers.ShowComment)
```

{{< callout type="info" >}}
**A hole must be a whole segment.** `/users/{id}` is valid; `/user-{id}` and
`/@{handle}` are not, because there the fixed text and the hole share one
segment, and Go's router cannot split a segment in two.

If you want a URL of that shape — `/@alice`, say — rewrite the path in
[HTTP middleware](/docs/http/middleware) before matching happens: the request
arrives for `/@alice`, your middleware turns it into `/u/alice`, and an
ordinary `/u/{handle}` route answers it.
{{< /callout >}}

## Route groups

Sometimes you'll have twenty routes that all start with `/api`, and fifteen
that all need the same sign-in check. A **group** lets you say either of those
things once.

It helps to know what a group actually is, because it is less than it sounds:
it is a small object holding a path prefix and a list of middleware, and it
sticks both onto each route as you register it. When you call
`api.Get("/users", …)` on a group whose prefix is `/api`, what reaches the
router is an ordinary route for `/api/users`. Once your routes are registered
there is no group left at all — just routes, exactly as if you had typed each
one out in full.

So a group costs you nothing while your application runs. What you get is
writing the prefix and the rule **once**:

```go
api := r.Group("/api")
api.Get("/users", handlers.UserIndex)    // → /api/users
api.Post("/users", handlers.UserStore)   // → /api/users

admin := r.Group("/admin")
admin.UseBefore(auth.Protected)          // applies to every route below
admin.Get("/dashboard", handlers.AdminDashboard)
```

Every route in that `admin` group now needs a signed-in user, because
`UseBefore` attached the check to the group instead of to each route.

That is the real reason to reach for a group. Repeating `auth.Protected` on
thirty admin routes works fine until the day you add the thirty-first and
forget — and that one unprotected route will look exactly like its thirty
neighbours in the file. Put it on the group and a new route cannot forget
something it never had to mention.

Groups nest, and the prefixes join:

```go
v1 := api.Group("/v1")
v1.Get("/products", handlers.ProductIndex)   // → /api/v1/products
```

## Middleware on a route

**Middleware** is a function that runs *before* your handler and gets to
decide whether your handler runs at all — a checkpoint on the way in. Checking
that somebody is signed in is the usual example: look at the request, then
either wave it through or answer it yourself with a redirect to the sign-in
page.

The good news is that there is nothing new to learn here. Middleware in
Lemmego is not a special type — it is an ordinary handler, listed before the
one that answers:

```go
r.Get("/search", auth.OptionalAuth, handlers.SearchIndex)
r.Post("/posts", auth.Protected, handlers.PostStore)
```

Each one runs in turn. To pass control along, a handler calls `c.Next()`; to
stop the chain, it simply returns without calling it. That is how
`auth.Protected` turns away a visitor who is not signed in.

And `auth.Protected` is `func(c app.Context) error`, exactly like the handlers
you write. That is the entire contract — there is no middleware interface for
you to satisfy.

[Middleware](/docs/http/middleware) covers writing your own, and the second
kind that wraps the entire router.

## Registering your route files

Writing a function full of routes is not quite enough — something has to call
it. You'll find that list in `bootstrap/routes.go`:

```go
// bootstrap/routes.go
package bootstrap

func LoadRoutes() []app.RouteCallback {
    return []app.RouteCallback{
        routes.WebRoutes,
        routes.ApiRoutes,
    }
}
```

A `RouteCallback` is just `func(a app.App)` — a function that gets handed your
running application and uses it to reach `a.Router()`. When you add a new route
file, add its function here too.

Don't worry about creating this file: `lemmego new` writes it for you, with
`WebRoutes` and `ApiRoutes` already listed.

## Router reference

```go
type Router interface {
    Group(prefix string) *routeGroup
    UseBefore(handlers ...Handler)
    UseAfter(handlers ...Handler)
    Get(pattern string, handlers ...Handler) *route
    Post(pattern string, handlers ...Handler) *route
    Put(pattern string, handlers ...Handler) *route
    Patch(pattern string, handlers ...Handler) *route
    Delete(pattern string, handlers ...Handler) *route
    Head(pattern string, handlers ...Handler) *route
    Options(pattern string, handlers ...Handler) *route
    Connect(pattern string, handlers ...Handler) *route
    Trace(pattern string, handlers ...Handler) *route
    Use(middlewares ...HTTPMiddleware)
    HasRoute(method string, pattern string) bool
    Handle(pattern string, handler http.Handler)
    HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}
```

`Handle` and `HandleFunc` mount a plain `net/http` handler, for reusing code
written against the standard library.

Want to see everything you have registered, including the routes modules
added on your behalf? The `list-routes` tool on the [MCP server](/docs/mcp)
reports every route's method, resolved path and handler from your running
application.

## Next steps

- [Middleware](/docs/http/middleware) — writing your own checkpoints.
- [Request handling](/docs/http/request-handling) — reading form and JSON
  bodies, and validating them.
- [Context](/docs/http/context) — the full API of the `c` every handler
  receives.
