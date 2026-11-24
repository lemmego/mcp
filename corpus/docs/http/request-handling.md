---
title: Request Handling
type: docs
prev: docs/http/middleware
next: docs/http/response-handling
weight: 33
sidebar:
  open: true
---

When somebody submits a form on your site, the fields they typed arrive in the
request's **body**. So does a JSON document sent by a script. Either way you
get bytes, and what you actually want is a Go struct with the fields filled
in — so something has to read those bytes, work out which value belongs to
which field, and populate it. That step is called **binding**, and it is the
first half of this page.

The second half is **validation**: making sure what arrived is usable before
you act on it. Binding gets the data into shape; validation decides whether
you can trust it. You want both, because whoever sent the request wrote that
body, and nothing stops them sending anything they like.

## Binding a body to a struct

### ParseInput

`ParseInput` reads the request body into a struct you supply. The `Content-Type` header picks the decoder, and the decoder picks which struct tag it reads:

If you haven't met them: a **struct tag** is that string in backticks after a
field — ``json:"email"`` — which tells a decoder what the incoming value is
called. Each decoder reads only its own tag, and that matters below.

| Content-Type | Decoder | Tag |
|---|---|---|
| `application/json`, `*+json` | `encoding/json` | `json:` |
| `application/xml`, `text/xml`, `*+xml` | `encoding/xml` | `xml:` |
| anything else | the framework's form binder | `form:` |

"Anything else" covers URL-encoded forms, multipart forms, and a query string on a request with no body. Both bodies are bounded at 1 MiB.

```go
func MyHandler(c app.Context) error {
    var input CreateUserInput
    err := c.ParseInput(&input)
    if err != nil {
        return c.BadRequest(err)
    }
    return c.JSON(app.M{"input": input})
}
```

A field needs the tag for the content type it will actually arrive as. **A struct tagged only `json:` binds nothing from an HTML form** — the form binder reads `form:`, finds no name for the field and leaves it at its zero, so validation reports "required" for a field the person just filled in. Tag both when an endpoint serves both:

```go
Name  string `json:"name" form:"name"`
Email string `json:"email" form:"email"`
```

The form binder also handles what a browser really sends: an unchecked box is absent rather than `false`, a checked one is `on`; a repeated field binds to a slice as `tags`, `tags[]` or `tags[0]` alike; a pointer field distinguishes absent from present-and-empty; and a `*multipart.FileHeader` or `[]*multipart.FileHeader` field binds the uploaded files themselves.

### Input Structs and Validation

Define input structs with a `Validate()` method:

```go
type CreateUserInput struct {
    Name  string `json:"name" form:"name"`
    Email string `json:"email" form:"email"`
}

func (i *CreateUserInput) Validate() error {
    v := app.NewValidator()
    v.Field("name", i.Name).Required().Max(255)
    v.Field("email", i.Email).Required().Email()
    return v.Validate()
}

func NewCreateUserInput(c app.Context) (*CreateUserInput, error) {
    input := &CreateUserInput{}
    if err := c.ParseInput(input); err != nil {
        return nil, err
    }
    return input, input.Validate()
}
```

Use with the handler:

```go
func Store(c app.Context) error {
    input, err := NewCreateUserInput(c)
    if err != nil {
        return c.UnprocessableEntity(err)
    }
    // input is validated and ready to use
}
```

### Size rules

`Min`, `Max` and `Between` measure whatever you give them:

```go
v.Field("age", i.Age).Min(18)                  // the number
v.Field("password", i.Password).Min(8)         // characters
v.Field("topics", i.TopicIDs).Between(1, 5)    // items
```

A string is counted in **characters, not bytes**, so an eight-character
minimum means eight characters whatever alphabet they are in. The message
follows the value — "must be at least 8", "at least 8 characters", "at least
8 items" — so you do not need a different rule for each.

{{< callout type="warning" >}}
Needs `api` v0.12.0. Below that version these three rules applied to `int`
only and **silently did nothing to a string**, so a password minimum written
against an older release was never enforced. See
[Upgrade notes](/docs/appendix/upgrading#api-v0120--min-max-and-between-enforce-on-strings).
{{< /callout >}}

### Changing the messages

Every built-in rule ships with English the framework owns, so a validator
says sentences without any setup. To reword one, publish the catalogue and
edit it:

```shell
lemmego run publish --tags=i18n
```

That writes `resources/lang/en/validation.json` and refuses to overwrite a
file you have already changed. The messages say "This field is required"
rather than naming the field, because both the Inertia and template error
paths render a message next to its own input. If you would rather name it,
`:attribute` is bound on every message:

```json
{ "required": "The :attribute field is required." }
```

and you can set what it resolves to per validator:

```go
v := app.NewValidator()
v.Attributes(map[string]string{"email": "email address"})
```

For a second language, see [Localization](/docs/localization).

### Input Middleware

Decode in the handler, with `c.ParseInput`:

```go
func UserStore(c app.Context) error {
    var input CreateUserInput
    if err := c.ParseInput(&input); err != nil {
        return err
    }
    // ...
}
```

`c.Input(&CreateUserInput{})` is the same decode and returns the value
instead of an error. It takes the struct to decode into — the no-argument
form does not compile.

In practice the decode and the validation belong together, in a constructor
beside the input type, so a handler gets one call and one error:

```go
func NewCreateUserInput(c app.Context) (*CreateUserInput, error) {
    input := &CreateUserInput{ctx: c}
    if err := c.ParseInput(input); err != nil {
        return nil, err
    }
    return input, input.Validate()
}
```

{{< callout type="warning" >}}
`app.Input(&CreateUserInput{})` exists and looks like route middleware, but
it returns an `app.Middleware` and **nothing in the framework accepts one**.
Routes, groups and the router all take `app.Handler`. There is no way to wire
it up, and nothing in the framework or any application built on it does.
{{< /callout >}}

## JSON Body Decoding

For direct JSON decoding without struct binding:

```go
func MyHandler(c app.Context) error {
    var data map[string]any
    err := c.DecodeJSON(&data)
    if err != nil {
        return c.BadRequest(err)
    }
    return c.JSON(data)
}
```

The low-level `DecodeJSONBody` function enforces:
- 1MB maximum body size
- Rejection of unknown fields
- Single JSON object enforcement
- Detailed error messages

## Content-Type Detection

Check what format a client expects:

```go
if c.WantsJSON() {
    // Return JSON response
}
if c.WantsHTML() {
    // Return HTML response
}
if c.WantsXML() {
    // Return XML response
}
```

Check what format the client sent:

```go
if c.HasJSONRequest() { /* JSON body */ }
if c.HasFormDataRequest() { /* multipart form */ }
if c.HasFormURLEncodedRequest() { /* URL-encoded form */ }
```

## File Uploads

Handle file uploads with `c.Upload()`:

```go
func Upload(c app.Context) error {
    file, err := c.Upload("avatar", "avatars")
    if err != nil {
        return c.BadRequest(err)
    }
    return c.JSON(app.M{"path": file.Name()})
}
```

Parameters: the form field name, target directory, and an optional custom filename.

## Next steps

- [Response handling](/docs/http/response-handling) — sending JSON, HTML,
  redirects and files back.
- [Error handling](/docs/http/error-handling) — turning a failed validation
  into a response.
- [Context](/docs/http/context) — the full API of the `c` your handler
  receives.
