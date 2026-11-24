---
title: Quick Start
type: docs
prev: docs/installation/
next: docs/project-structure
weight: 13
sidebar:
  open: true
---

## Creating an application

With the `lemmego` CLI installed, create your project:

```shell
lemmego new my-project
```

This writes you a complete, working application — directories, configuration,
a route file, a home page — rather than an empty folder. Generating files like
this is called **scaffolding**, and it means your first `lemmego run` serves a
real page.

The first question asks how much you want to be asked:

- **Quick start** wants four answers and picks sensible defaults for
  everything else.
- **Customise** walks you through each part and lets you choose.

If you take Quick start, here is what you are agreeing to. Don't worry about
getting these right now — every one of them can be changed later, and each
links to the page that explains it:

| | Default | What it is |
|---|---|---|
| Database | SQLite | A database kept in a single file, so there is no server for you to install or run |
| SQL layer | [the Lemmego ORM](/docs/orm/) | Lets you query in Go — `db.Model[User]()` — instead of writing SQL by hand |
| [Cache](/docs/cache/) | file | Keeps the results of slow work so you don't repeat it, here in files on disk |
| [Queue](/docs/queue) | SQL | Holds work to be done later, so a slow job doesn't make your visitor wait |
| [Sessions](/docs/session) | file | How your app remembers who is signed in between requests |
| Storage | local | Where uploaded files go — your own disk, rather than S3 |

The other two answers are your **module name** (your Go import path, like
`github.com/you/my-project`) and a **preset** — the overall shape of the
project. `mvc` gives you a server-rendered site with pages and templates;
`rest_api` gives you JSON endpoints and no frontend at all.

The database, cache and queue can each be set to **None**, which leaves that
part out of the project entirely — no dependency, no provider, no config file.
See [the CLI reference](/docs/cli/#leaving-a-part-out).

Everything is also available as a flag with `--non-interactive`:

```shell
lemmego new my-project --non-interactive \
  --module github.com/username/my-project \
  --database postgres --cache redis --queue redis
```

Once configured, navigate into your project directory:

```shell
cd my-project
```

## Running the Application

Start the development server:

```shell
lemmego run
```

Or use hot-reload:

```shell
lemmego dev
```

The `dev` command orchestrates three parallel processes:
- **air** — Go hot reload on `:1490`
- **templ generate --watch** — Templ file watcher (if `.templ` files exist)
- **vite dev** — Frontend dev server (if `package.json` exists)

Visit `http://localhost:1490` in your browser — you should see the application's homepage.

## Development Workflow

The generated project includes a `Makefile` with common tasks:

| Command | Description |
|---------|-------------|
| `make dev` | Everything at once: hot reload, frontend, and your database if it needs one |
| `make run` | Run the application without hot reload |
| `make build` | Build for production |
| `make test` | Run the test suite |
| `make migrate` | Run pending migrations |
| `make rollback` | Roll the last batch back |
| `make migration n=name` | Create a new migration |
| `make model n=name` | Generate a model |
| `make handlers n=name` | Generate CRUD handlers |
| `make input n=name` | Generate input validation |
| `make form n=name` | Generate a form component |
| `make appkey` | Generate a new `APP_KEY` |
| `make deps` | Tidy modules and install `templ` and `air` |
| `make libsql` | Run a [libSQL](/docs/database/libsql/) server on its own |
| `make ssr-start` | Start the Inertia server-side renderer |
| `make ssr-stop` | Stop it |
| `make ssr-check` | Report whether it is running |

`make dev` is the one you'll use. It runs [`lemmego dev`](/docs/cli/), which
starts the frontend dev server, the Go hot reloader and — for a libSQL
project — the database, then shuts them all down together on `Ctrl+C`.

## Frontend Options

Lemmego supports multiple frontend approaches:

- **Go Templates** (`*.page.gohtml`) — server-rendered HTML with layouts and partials
- **Templ** (`*.templ`) — type-safe Go templates with compile-time checking
- **Inertia.js** with React or Vue — SPA-like experience with server-side routing

The `make watch` command handles Go Templates and Templ reloading. For Inertia, run `npm run dev` separately.

## Project Structure

After scaffolding, your project will have this structure:

```
├── bootstrap/          # Application bootstrap layer
│   ├── providers.go    # Service provider registration
│   ├── routes.go       # Route registration
│   ├── middleware.go   # App-level middleware
│   └── commands.go     # CLI commands
├── cmd/app/main.go     # Application entry point
├── internal/           
│   ├── commands/       # CLI command implementations
│   ├── configs/        # Configuration (auto-loaded)
│   ├── handlers/       # HTTP request handlers
│   ├── inputs/         # Input validation structs
│   ├── middleware/      # App-specific middleware
│   ├── migrations/     # Database migrations
│   ├── models/         # Domain models
│   ├── plugins/        # Extension point
│   ├── repos/          # Repository layer
│   └── routes/         # Route definitions
├── resources/          # Frontend source (JS, CSS, templates)
├── templates/          # Go/Templ templates
├── public/             # Compiled static assets
└── storage/            # Runtime data (database, sessions, uploads)
```

## Next steps

- [Project structure](/docs/project-structure) — what each generated directory
  is for.
- [Configuration](/docs/configuration) — reading settings and adding your own.
- [The HTTP layer](/docs/http/) — add your first route and handler.
