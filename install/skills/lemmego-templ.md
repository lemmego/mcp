## When to use this skill

When rendering a [templ](https://templ.guide) component from a Lemmego handler.

## The whole surface is two calls

    func ShowHandler(c app.Context) error {
        return templ.Respond(c, views.Show(post))
    }

    // When the status or a header matters:
    return templ.New(c, views.Show(post)).Status(201).Render()

That is the module. It is a renderer, not a framework — anything else you want
is `app.Context`'s job.

## Generated code is a build step

`templ generate` writes a `_templ.go` beside every `.templ` file, and **the Go
code is what compiles**. Two consequences:

- A `.templ` edit does nothing until you regenerate. `make watch` runs the
  generator; `make run` on its own may not.
- The generated files are build output. Do not edit a `_templ.go`, and do not
  hand-write one to avoid installing the binary.

If a page renders stale markup and the template clearly says otherwise,
regenerate before debugging anything else.

## Type safety is the reason to choose it

A component takes typed parameters, so a renamed model field is a compile
error rather than a blank space on a page. That is the trade against Go
templates, which resolve names at render time.

Pass the data a component needs as parameters. Reaching into a context inside a
component gives up the property you chose templ for.

## Choosing between the options

| | |
| --- | --- |
| Go templates | no build step, names resolved at render time |
| templ | a build step, names checked by the compiler |
| Inertia | a React or Vue front end, server-side routing |

`templ` and Inertia compose: the `templ_inertia_*` presets render the document
shell with templ and the page with Inertia.

## Escaping

templ escapes interpolated values by default. `templ.Raw` opts out and is the
one call to review carefully — it is how user content becomes markup.

## Check first

`search-docs` section `frontend/templ`, `module-api` on `templ`. For Inertia
pages and server-side rendering see the `lemmego-inertia` skill.
