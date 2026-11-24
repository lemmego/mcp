---
title: Context
type: docs
prev: docs/http/error-handling
next: docs/frontend/
weight: 36
sidebar:
  open: true
---

Every handler you write takes one argument — `c app.Context` — and that single
value is how you do everything. It is the request and the response in one
object: you read from it to find out what was asked, and you call methods on
it to answer.

```go
func Show(c app.Context) error {
    id := c.Param("id")           // reading: what was asked for
    return c.JSON(app.M{"id": id}) // answering
}
```

Through `c` you reach the URL parameters, the query string, the body, uploaded
files, headers and cookies, the session, the signed-in user, and every way of
replying. You will not often need to import anything else inside a handler.

This page is the full list, so treat it as reference rather than something to
read through — the pages before it cover the parts you need first.

```go
type Context interface {
    GetSetter
    HttpProvider
    App() App
    Next() error
}
```

## Request Data

### URL Parameters and Query Strings

```go
id := c.Param("id")           // Route parameter: /users/{id}
page := c.Query("page")       // Query string: ?page=2
```

### Request Body & Form Data

```go
body, _ := c.Body()              // Raw form body (map[string][]string)
form, _ := c.Form()              // Parsed form (supports multipart)
```

### File Uploads

```go
file, header, err := c.FormFile("avatar")  // Get uploaded file
if c.HasFile("avatar") {                    // Check if file was uploaded
    // Process file
}
```

### Content Type Detection

```go
c.HasJSONRequest()              // Content-Type: application/json
c.HasFormDataRequest()          // Content-Type: multipart/form-data
c.HasFormURLEncodedRequest()    // Content-Type: application/x-www-form-urlencoded
c.HasMultiPartRequest()         // Is multipart request?
```

### Request Metadata

```go
r := c.Request()              // *http.Request
w := c.ResponseWriter()        // http.ResponseWriter
ctx := c.RequestContext()      // context.Context
method := r.Method
path := r.URL.Path
```

## Input Decoding

```go
// Decode JSON body into struct
var input MyInput
if err := c.DecodeJSON(&input); err != nil { ... }

// Parse the body, whatever it is: the Content-Type picks the decoder
if err := c.ParseInput(&input); err != nil { ... }

// One-liner: parse and store in context
result := c.Input(&MyInput{}).(*MyInput)
```

## Content Negotiation

```go
c.WantsJSON()           // Accept: application/json
c.WantsHTML()           // Accept: text/html
c.WantsXML()            // Accept: application/xml
```

## Reading State

```go
c.IsReading()           // Is this a GET/HEAD/OPTIONS request?
c.Status()              // Get current response status code
```

## Request-Scoped Storage

Store and retrieve data across middleware and handlers:

```go
// Set in middleware
c.Set("user", currentUser)

// Get in handler
user := c.Get("user").(*User)
```

## Session Access

```go
// Read
val := c.Session("key")
str := c.SessionString("key")

// Write
c.PutSession("key", value)

// Read and remove
val := c.PopSession("key")
str := c.PopSessionString("key")
```

## Cookie Management

```go
// Read
cookie := c.Cookie("session_id")

// Write
c.SetCookie(&http.Cookie{
    Name:  "preference",
    Value: "dark-mode",
    Path:  "/",
})
```

## Headers

```go
// Read
ua := c.Header("User-Agent")

// Write
c.SetHeader("X-Request-Id", requestID)
```

## Status

```go
c.Status()                    // Get current status code
c.SetStatus(http.StatusCreated) // Set status
c.WriteStatus(http.StatusOK)  // Write status to response
```

## Response Types

### JSON

```go
return c.JSON(app.M{
    "id":    1,
    "name":  "John",
})
```

### XML

```go
// From a map
return c.XML(app.M{"status": "ok"})

// From a struct
return c.XML(myStruct)

// From a raw string
return c.XML("<status>ok</status>")
```

### HTML

```go
return c.HTML([]byte("<h1>Hello</h1>"))
```

### Text

```go
return c.Text([]byte("Hello, World!"))
```

### Redirect

```go
return c.Redirect("/dashboard")
return c.Back() // redirect to referrer
```

### File Responses

```go
// Serve a file from the filesystem
return c.File("path/to/file.pdf")

// Serve a file from a storage disk
return c.StorageFile("uploads/report.pdf")

// Download a file with a custom filename
return c.Download("path/to/file.pdf", "report.pdf")
```

### Raw Write

```go
c.Write([]byte("raw data"))
```

## Template Rendering

### Go Templates

```go
import "github.com/lemmego/api/res"

tmpl := res.NewTemplate(c, "index.page.gohtml")
tmpl = tmpl.WithData(map[string]any{"title": "Hello"})
return c.Render(tmpl)
```

### Templ

```go
import "github.com/lemmego/templ"

return c.Templ(components.Hello("World"))
```

## Error Helpers

```go
c.Error(status, err)            // Send error with status code

// Typed errors (return these from handlers)
return c.InternalServerError(err)
return c.NotFound()
return c.NotFound(fmt.Errorf("user %d not found", id))
return c.BadRequest(err)
return c.Unauthorized(err)
return c.Forbidden(err)
return c.MethodNotAllowed(err)
return c.PageExpired(err)
return c.UnprocessableEntity(err)
return c.NoContent()              // 204

// Validation error helper
return c.ValidationError(err)
```

## Validation

```go
v := c.Validator()
v.Field("email", input.Email).Required().Email()
if err := v.Validate(); err != nil {
    return c.UnprocessableEntity(err)
}
```

## File Upload Helper

```go
file, err := c.Upload("avatar", "avatars")
// Saves the uploaded file to the configured storage disk
// Returns the created file

// With custom filename:
file, err := c.Upload("avatar", "avatars", "custom-name.jpg")
```

## Referer

```go
ref := c.Referer()            // Referer header
```

## Middleware Chain

```go
func myMiddleware(next app.Handler) app.Handler {
    return func(c app.Context) error {
        // Pre-processing
        err := next(c) // Call the next handler
        // Post-processing
        return err
    }
}
```

## App Access

```go
a := c.App()                  // Access the App instance
cfg := a.Config()             // Configuration
sess := a.Session()           // Session manager
fs := a.FileSystem()          // Filesystem manager
```

## Wrapping a Raw Handler

A handler registered through the router already receives a `Context`. The
exception is a **third-party `http.Handler` mounted on the router** — a
management dashboard, an embedded admin UI — which is handed an
`*http.Request` and nothing else, and is therefore cut off from auth, sessions
and the container.

`app.NewContext` bridges that gap:

```go
func guard(a app.App, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        c := app.NewContext(a, w, r)
        if err := auth.Check(c); err != nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, c.Request())
    })
}
```

Two things to know:

- **`Set` rebinds the request.** Pass `c.Request()` to the handler you wrap,
  not the `r` you started with, or everything your check established — the
  loaded user, most of all — is silently dropped.
- **No router middleware has run.** A raw mount bypasses the handler chain
  entirely, so nothing has looked at the request's cookie yet. Call
  `auth.Check` yourself; reading the user without it finds nobody.

This is what the queue dashboard uses — see
[Queues](/docs/queue#web-dashboard). Reach for it nowhere else.

## Interface Reference

```go
type Context interface {
    GetSetter
    HttpProvider
    App() App
    Next() error
}

type GetSetter interface {
    Get(key string) any
    Set(key string, value any)
}

type HttpProvider interface {
    ParamQueryResolver      // Param, Query
    InputDecoder            // ParseInput, Input, DecodeJSON
    BodyParser              // Body, Form, FormFile, HasFile, HasMultiPartRequest,
                            // HasFormDataRequest, HasFormURLEncodedRequest, HasJSONRequest
    RequestBodyValidator    // Validate, Validator
    HeaderGetSetter         // Header, SetHeader
    AcceptHeaderResolver    // WantsJSON, WantsHTML, WantsXML
    RequestGetSetter        // Request, SetRequest
    RequestResponseResolver // Request, ResponseWriter, RequestContext
    CookieGetSetter         // Cookie, SetCookie
    SessionGetSetter        // Session, SessionString, PopSession, PopSessionString, PutSession
    HttpResponder           // JSON, XML, Text, HTML, Templ, Redirect, Back, File,
                            // StorageFile, Download, Write, Render
    Downloader              // Download
    Uploader                // Upload
    ErrorProvider           // Error, InternalServerError, NotFound, BadRequest,
                            // Unauthorized, Forbidden, PageExpired, MethodNotAllowed,
                            // UnprocessableEntity, NoContent, ValidationError
    IsReading() bool
    Status() int
    SetStatus(code int) HttpResponder
    WriteStatus(code int) HttpResponder
    Referer() string
}
```

## Next steps

- [Frontend](/docs/frontend/) — the rendering approaches behind the template
  helpers.
- [Session](/docs/session) — the store behind the session accessors.
- [The current user](/docs/security/current-user) — who is asking, on every
  authenticated path.
