---
title: Glossary
type: docs
prev: docs/prologue
next: docs/installation/
weight: 11
sidebar:
  open: true
---

Every term these docs use, in plain language. If a page uses a word you have
not met, it is defined here.

Entries say what a thing *is* and why it exists, because a definition that
only names the parts leaves you no better off.

## The web, underneath

**HTTP** — the rules browsers and servers use to talk. A client sends a
*request*, the server sends back a *response*, and the connection is done.
Each request stands alone: the server remembers nothing between them unless
you arrange for it to (see *session*).

**Request** — a short message from a browser. It carries a *method*, a
*path*, some *headers*, and sometimes a *body*. That is all your program is
given about what somebody wants.

**Method** (or *verb*) — the first word of a request, naming the kind of
action: `GET` to fetch, `POST` to create, `PUT` to replace, `PATCH` to amend,
`DELETE` to remove. Nothing enforces the meanings; they are a promise that
browsers and caches rely on, which is why a `GET` must never change data.

**Path** — the part of the address after the domain: `/users/42`. What the
request concerns.

**Query string** — the part after `?`, as in `/search?q=go&page=2`. Extra
named values, usually options rather than identity.

**Header** — a named line of metadata on a request or response:
`Content-Type: application/json`, `Cookie: …`. Not the content itself, but
facts about it.

**Body** — the content of a request or response. A submitted form, a JSON
document, an uploaded file, a rendered page.

**Status code** — a three-digit number on a response saying how it went.
`200` fine, `302` look over there instead, `404` no such thing, `403` you may
not, `419` your form token expired, `500` the server broke. The first digit is
the category: `2xx` worked, `3xx` redirect, `4xx` the request was wrong,
`5xx` the server was.

**Cookie** — a small named value the server asks the browser to keep and send
back on every later request. The usual way a server recognises a returning
visitor.

## Running your application

**Framework** — a library that calls *your* code, rather than one you call. It
owns the loop that accepts requests and decides when to run your functions,
so the parts every web application needs — routing, sessions, configuration —
exist already and you fill in what is specific to yours.

**Handler** — one of your functions that answers a request. In Lemmego it is
`func(c app.Context) error`.

**Route** — a rule pairing a method and a path with a handler: `GET /hello` →
run this function. A web application is a list of these.

**Router** — the thing holding the list of routes, which looks up the right
handler when a request arrives.

**Route parameter** — a named hole in a path pattern. `/users/{id}` matches
`/users/42`, and `c.Param("id")` gives you `"42"`. It is how one route serves
every user.

**Middleware** — a handler that runs before the one that answers, and can
stop the request there. A checkpoint: authentication, rate limiting, CSRF
checking. See [Middleware](/docs/http/middleware).

**Context** — the single value a handler receives, carrying everything about
the request and every way of replying. Lemmego calls it `c`.

**Request lifecycle** — the ordered path a request takes: server, HTTP
middleware, router, application middleware, handler, response. See
[the HTTP layer](/docs/http/).

## How the framework is wired

**Bootstrap** — the startup sequence that runs before any request: read
configuration, run providers, mount routes, then listen.

**Provider** — a unit of setup. A provider is handed the application at
startup and registers what it contributes: a database connection, a mail
transport, extra routes. Adding a module to your project means adding its
provider. See [Providers](/docs/core-concepts/provider-plugin-system).

**Service container** — a box of ready-made objects, keyed by type. A
provider puts the mail transport in; a handler takes it out with
`app.Get[*mail.Mailer](a)`. It saves passing every dependency through every
function.

**Dependency injection** — handing a piece of code the things it needs rather
than letting it construct them. The point is substitution: a test can pass a
fake mail transport where production passes a real one, with no change to the
code between.

**Scaffolding** — generated starter files. `lemmego new` writes a working
project; `lemmego gen model Post` writes one file you then edit.

**CLI** — a command-line program. Here, the `lemmego` command.

## Storing data

**SQL** — the language databases speak: `SELECT`, `INSERT`, `UPDATE`,
`DELETE`.

**Table, row, column** — a table is a grid, like a spreadsheet tab. A column
is a named field with a type; a row is one record.

**Primary key** — the column uniquely identifying a row, usually `id`.

**Foreign key** — a column holding another table's primary key, which is how
rows reference each other. Declared as a constraint, the database then refuses
a reference to a row that does not exist.

**Index** — a lookup structure the database keeps so it can find rows by a
column without reading every one. The difference between instant and slow on a
large table.

**ORM** (object-relational mapper) — a layer letting you work in Go structs
and method calls instead of writing SQL by hand. `db.Model[User]().Where(…)`
becomes a `SELECT`, and the rows come back as `[]User`. See
[the ORM](/docs/orm/).

**Model** (or *entity*) — a Go struct standing for one table's row shape.

**Repository** — an object holding all the queries for one kind of model, so
SQL lives in one place rather than scattered through handlers.

**Transaction** — a group of writes that all take effect or none do. If the
second of three fails, the first is undone. Essential whenever two tables
must agree.

**Migration** — a file describing one change to your database's structure —
add a table, add a column — with the code to apply it and the code to undo
it. Kept in order and in version control, so every machine can reach the same
schema. See [Migrations](/docs/migrations/).

**Schema** — the current shape of your database: its tables, columns and
constraints.

**DSN** (data source name) — the connection string locating a database:
`postgres://user:pass@host:5432/dbname`.

**Connection pool** — a set of open database connections kept and reused,
because opening one per request is slow.

**Soft delete** — marking a row deleted with a timestamp instead of removing
it, so it can be restored and so rows referring to it are not orphaned.
Queries then exclude it by default.

**Eager loading** — fetching related rows in one extra query rather than one
per row. Without it, listing 50 posts and reading each author costs 51
queries — the *N+1 problem*.

**Pivot table** (or *join table*) — a table whose rows pair two other tables'
keys, which is how "many of these belong to many of those" is stored.

## Showing a page

**Template** — a file of HTML with holes for your data to fill.

**Server-side rendering (SSR)** — building the final HTML on the server, so
the browser receives a finished page. The alternative sends a near-empty page
plus JavaScript that builds it. Rendering on the server matters for anything a
search engine or a link preview must read.

**Single-page application (SPA)** — a site where JavaScript swaps the content
on navigation rather than loading a new document, so the page does not blank
between views.

**Inertia** — a way to get an SPA's feel while keeping routing and data
loading in Go. Your handler returns a component name and some data; a React or
Vue component renders it. No separate API to build. See
[Inertia](/docs/frontend/inertia).

**Props** — the data a handler passes to a frontend component.

**Hydration** — JavaScript attaching behaviour to server-rendered HTML that
is already on screen, making it interactive without redrawing it.

**Vite** — the tool that bundles your JavaScript and CSS for the browser, and
during development reloads changes instantly.

**HMR** (hot module replacement) — replacing just the changed module in a
running page, so editing a component does not reset what you had typed.

**Asset** — a built JavaScript or CSS file the browser downloads.

## Who is asking, and what they may do

**Authentication** — establishing *who* is making a request. Signing in. See
[Authentication](/docs/security/authentication).

**Authorization** — deciding *what* they may do. A separate question with a
separate answer: you can be authenticated and still not allowed. See
[Authorization](/docs/authz/).

**Session** — server-side memory of one visitor across requests, found by an
id in a cookie. Where "who is signed in" usually lives. See
[Session](/docs/session).

**CSRF** (cross-site request forgery) — an attack where another site causes
*your* browser to submit a request to your application, using cookies it
already holds. Defended by requiring a secret value in the request that only
your own pages include. A `419` is that check failing. See
[CSRF](/docs/security/csrf).

**JWT** (JSON Web Token) — a signed block of data a client holds and presents
on each request. Needs no server-side storage, which also means it cannot be
revoked before it expires.

**Hashing** — turning a password into a value it cannot be recovered from, so
a stolen database does not hand over passwords. Verifying means hashing the
attempt and comparing.

**OAuth2** — the protocol behind "sign in with GitHub": one application gets
limited access to another on a user's behalf, without learning their
password.

## Work that happens later, and faster

**Cache** — a copy of something expensive kept somewhere fast, so the
expensive work is skipped next time. See [Caching](/docs/cache/).

**TTL** (time to live) — how long a cached value stays valid before it is
discarded.

**Queue** — a list of work to do later. A request puts a job on it and
replies immediately; something else does the work. Sending email belongs here,
because a slow mail server should not be a slow sign-up.

**Job** — one unit of such work.

**Worker** — a separate process that takes jobs off the queue and runs them.
Jobs do not run unless a worker is running.

**Cron / scheduler** — a clock that runs a job on a timetable, such as nightly.

**Pub/sub** — one process publishing a message that any number of others
receive, which is how several copies of your application tell each other
something happened.

**SSE** (server-sent events) — a connection held open so the server can push
messages to a browser. One direction only.

**WebSocket** — the same, but both directions.

## Operating it

**Environment variable** — a named value given to a program by the system
that starts it. Where credentials belong, because they stay out of your code
and change per machine.

**`.env`** — a file of environment variables for local development, never
committed.

**Log level** — how important a log line is: debug, info, warn, error. Lets
you keep detail in a file while showing only problems on screen.

**Structured logging** — writing log lines as fields rather than prose, so
they can be searched and filtered.

**Health check** — an endpoint a platform calls to ask whether your
application is working, so a broken instance is taken out of rotation.

**Graceful shutdown** — finishing in-flight requests before exiting, instead
of dropping them.

## Next steps

- [Installation](/docs/installation/) — get the `lemmego` CLI onto your
  machine.
- [Quick start](/docs/quick-start/) — create a project and watch it reload.
- [The HTTP layer](/docs/http/) — where routes, handlers and middleware do
  their work.
