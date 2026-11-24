---
title: Templ
type: docs
prev: docs/frontend/go-templates
next: docs/frontend/inertia
weight: 42
sidebar:
  open: true
---

With ordinary templates, a mistake is invisible until somebody loads the
page: rename a struct field and the template keeps referring to the old name,
printing nothing where the value should be. Nothing fails, nothing is logged,
and the page is just quietly wrong.

[Templ](https://templ.guide/) fixes that by **compiling your templates into Go
code**. A component becomes a Go function taking typed arguments, so a renamed
field is a compile error in the same build as the rest of your application.
The cost is a generation step: after editing a `.templ` file you run
`templ generate`, or let `lemmego dev` watch and do it for you.

## Setup

If you chose Templ when you ran `lemmego new`, everything here is already in
place — the files, the configuration and the watcher. There is nothing for you
to set up.

## Rendering a Templ Component

```go
import "github.com/lemmego/templ"

func MyHandler(c app.Context) error {
    return c.Templ(components.Hello("World"))
}
```

Or using the package-level helper:

```go
return templ.Respond(c, components.Hello("World"))
```

## TemplResponse

The `TemplResponse` type implements the `Renderer` interface, enabling it to be used with `c.Render()`:

```go
response := templ.New(c, components.Page(data))
return c.Render(response)
```

## How It Works

1. `templ.New(ctx, component)` creates a `TemplResponse` wrapping a `templ.Component`
2. On render, it sets `Content-Type: text/html`, writes the HTTP status code, then calls `component.Render(ctx.RequestContext(), w)`

## Template Sources

Templ source files are located in `templates/`:

```
templates/
├── base-layout.templ     # Base layout component
├── index.templ           # Index page component
├── method.templ          # Method override helper
└── csrf.templ            # CSRF token helper
```

The generated `_templ.go` files (compiled from `.templ`) should not be edited directly.

## Development Workflow

During development, run Templ in watch mode:

```shell
lemmego dev
```

This starts `templ generate --watch` alongside air, so changes to `.templ` files are automatically recompiled and the server reloads.

## Example Component

```go
// templates/hello.templ
package components

templ Hello(name string) {
    <div class="greeting">
        <h1>Hello, { name }!</h1>
    </div>
}
```

Usage in a handler:

```go
func HelloHandler(c app.Context) error {
    return c.Templ(components.Hello(c.Query("name")))
}
```

## CSRF and Method Helpers

The generated project includes Templ helpers for CSRF tokens and method override:

```go
templ CsrfField() {
    <input type="hidden" name="_token" value={ templ.Raw(csrf.Token) }/>
}

templ MethodField(method string) {
    <input type="hidden" name="_method" value={ method }/>
}
```

Usage in forms:

```go
templ EditForm(task models.Task) {
    <form method="POST" action={ templ.URL("/tasks/" + strconv.Itoa(task.ID)) }>
        @MethodField("PUT")
        @CsrfField()
        <input type="text" name="title" value={ task.Title }/>
        <button type="submit">Update</button>
    </form>
}
```

## Next steps

- [Vite integration](/docs/frontend/vite) — the asset pipeline your components
  link to.
- [Inertia.js](/docs/frontend/inertia) — the same page in React or Vue instead
  of Go.
- [Response handling](/docs/http/response-handling) — the handler side of
  rendering a component.
