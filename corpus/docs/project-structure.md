---
title: Project Structure
type: docs
prev: docs/quick-start/
next: docs/configuration
weight: 14
sidebar:
  open: true
---

`lemmego new` gives you a lot of directories, and the names only make sense
once somebody tells you the rule behind them. Here it is: **each directory
holds one kind of decision**, and the ones that change most often are kept
away from the ones that change least.

So your business rules do not live in the same file as your HTTP handling, and
neither lives in the file that wires the application together at startup. When
you come back in six months to change how a price is calculated, you want one
place to look.

Three names worth knowing before the tree below:

- **`bootstrap/`** — runs once at startup, before any request. It is the list
  of what your application is made of.
- **`internal/`** — your actual application. Go treats the name specially:
  nothing outside your module can import anything under it, so this code is
  yours alone and you can rearrange it freely.
- **`resources/`** — things that are *not* Go: templates, TypeScript, CSS.
  They get compiled or bundled into `public/`, which is what visitors
  download.

## Directory anatomy

```
my-project/
├── bootstrap/                  # Application bootstrap layer
│   ├── providers.go            # Service provider registration
│   ├── routes.go               # Route callback aggregation
│   ├── middleware.go            # App-level middleware
│   ├── http_middleware.go       # HTTP-level middleware
│   ├── commands.go             # CLI command registration
│   └── errmap.go               # Error-to-handler mapping
│
├── cmd/app/main.go             # Entry point (builder pattern)
│
├── internal/                    # Unexported application code
│   ├── commands/                # CLI command implementations
│   │   └── appkey.go            # Generate APP_KEY
│   │
│   ├── configs/                 # Configuration (auto-loaded via init())
│   │   ├── app.go               # App name, port, env
│   │   ├── database.go          # SQL & Redis connections
│   │   ├── filesystems.go       # Storage disks
│   │   └── session.go           # Session settings
│   │
│   ├── handlers/                # HTTP controllers
│   │   └── task_handler.go      # Example CRUD handler
│   │
│   ├── inputs/                  # Input validation structs
│   │   └── task_input.go        # Example with Validate()
│   │
│   ├── middleware/              # App-specific middleware
│   │
│   ├── migrations/              # Database migrations
│   │   ├── create_sessions_table.go
│   │   └── create_tasks_table.go
│   │
│   ├── models/                  # Domain models with hooks
│   │   ├── task.go
│   │   └── user.go
│   │
│   ├── plugins/                 # Plugin extension point
│   │
│   ├── repos/                   # Repository layer (GPA)
│   │   ├── repo.go              # Generic SQLRepo[T] helper
│   │   └── task_repo.go         # Typed repository wrapper
│   │
│   └── routes/                  # Route definitions
│       ├── web.go               # Browser-facing routes
│       └── api.go               # JSON API routes
│
├── resources/                   # Frontend source
│   ├── css/app.css              # Tailwind CSS source
│   ├── js/                      # TypeScript/JavaScript
│   │   ├── app.(tsx|js)         # Inertia entry point
│   │   ├── ssr.(tsx|js)         # SSR entry point
│   │   └── Pages/               # Inertia page components
│   └── views/root.html          # Inertia root layout
│
├── templates/                   # Go & Templ templates
│   ├── *.page.gohtml            # Go template pages
│   ├── *.layout.gohtml          # Go template layouts
│   ├── *_templ.go               # Generated Templ code
│   └── *.templ                  # Templ source files
│
├── public/                      # Compiled static assets
│   └── build/manifest.json      # Vite build manifest
│
├── storage/                     # Runtime data
│   ├── database.sqlite          # Default database
│   ├── images/                  # Uploaded files
│   └── session/                 # File-based sessions
│
├── .air.toml                    # Air hot-reload config
├── Makefile                     # Development workflow
├── package.json                 # Node.js dependencies
├── vite.config.js               # Vite configuration
└── go.mod                       # Go module
```

## Key Architecture Patterns

### Builder Pattern (Entry Point)

```go
// cmd/app/main.go
func main() {
    webApp := app.Configure()
    webApp.WithRoutes(bootstrap.LoadRoutes()).
        WithHTTPMiddlewares(bootstrap.LoadHTTPMiddlewares()).
        WithMiddlewares(bootstrap.LoadMiddlewares()).
        WithCommands(bootstrap.LoadCommands()).
        WithProviders(bootstrap.LoadProviders()).
        WithErrMap(bootstrap.LoadErrMap()).
    Run()
}
```

### Auto-Loaded Configuration

Configuration files use `init()` functions, triggered by a blank import in `main.go`:

```go
import _ "github.com/lemmego/lemmego/internal/configs"
```

### Convention-Based Routing

Routes are organized into `web.go` (browser) and `api.go` (JSON) files, aggregated in `bootstrap/routes.go`:

```go
func LoadRoutes() []app.RouteCallback {
    return []app.RouteCallback{
        routes.WebRoutes,
        routes.ApiRoutes,
    }
}
```

## Directory Conventions

| Directory | Purpose | Auto-generated |
|-----------|---------|---------------|
| `bootstrap/` | App initialization wiring | Yes |
| `internal/commands/` | CLI command implementations | Via generators |
| `internal/configs/` | Environment-based configuration | Run `lemmego new` |
| `internal/handlers/` | HTTP request handlers | Via `lemmego g handlers` |
| `internal/inputs/` | Input validation structs | Via `lemmego g input` |
| `internal/models/` | Domain models with hooks | Via `lemmego g model` |
| `internal/migrations/` | Database migration files | Via `lemmego g migration` |
| `internal/repos/` | Data access repositories | Via `lemmego gorm:repo` |
| `internal/routes/` | HTTP route definitions | Via `lemmego new` |

## Next steps

- [Configuration](/docs/configuration) — what lives in `internal/configs/`.
- [CLI & generators](/docs/cli/) — the commands that write into these
  directories.
- [Application bootstrap](/docs/core-concepts/application-bootstrap) — what
  `bootstrap/` runs at startup.
