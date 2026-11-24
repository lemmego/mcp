---
title: Configuration
type: docs
prev: docs/project-structure
next: docs/core-concepts/
weight: 15
sidebar:
  open: true
---

Your application needs to know things that are not code: which database to
connect to, which port to listen on, the password for your mail server. Those
answers differ between your laptop and your production server, and the
password must not be committed to git. **Configuration** is where they live.

Lemmego keeps them in two places, and the split is the thing to understand:

- **`internal/configs/*.go`** — the *shape* of your configuration, in Go. Here
  you say "there is a database setting, and its driver is whatever
  `DB_DRIVER` says, or `sqlite` if nothing says". You commit these files.
- **`.env`** — the *values* for this one machine. Here you say
  `DB_DRIVER=postgres`. You never commit this file; it is in `.gitignore`
  already.

That way the structure is shared and reviewable, while the secrets and the
per-machine details stay out of your repository. An **environment variable**
is simply a named value handed to your program by whatever starts it, and a
`.env` file is a convenient way to set a pile of them while you work locally.

## Reading and writing configuration

You'll find your settings in `internal/configs/`, defined in `init()`
functions — Go runs these automatically when the package loads, so there is
nothing for you to call:

```go
// internal/configs/app.go
package configs

import "github.com/lemmego/api/config"

func init() {
    config.Set("app", config.M{
        "name":  config.MustEnv("APP_NAME", "Lemmego"),
        "port":  config.MustEnv("APP_PORT", 1490),
        "env":   config.MustEnv("APP_ENV", "development"),
        "debug": config.MustEnv("APP_DEBUG", false),
    })
}
```

Config files are auto-loaded via a blank import in `cmd/app/main.go`:

```go
import _ "github.com/lemmego/lemmego/internal/configs"
```

### Reading Configuration

```go
port := config.Get("app.port").(int)
name := config.Get("app.name", "default").(string)
```

Or using the typed accessor on a `config.M` map:

```go
cfg := config.GetAll()
port := cfg.Int("app.port")
debug := cfg.Bool("app.debug")
```

Available accessor methods on `config.M`:
- `String(key, defaultVal...) string`
- `Int(key, defaultVal...) int`
- `Int64(key, defaultVal...) int64`
- `Bool(key, defaultVal...) bool`
- `Float64(key, defaultVal...) float64`
- `Duration(key, defaultVal...) time.Duration`
- `Time(key, defaultVal...) time.Time`

### Environment Variables

The framework loads `.env` files automatically via `github.com/joho/godotenv/autoload`. For type-safe environment variable access:

```go
port := config.MustEnv("APP_PORT", 1490)
debug := config.MustEnv("APP_DEBUG", false)
```

`MustEnv[T]` supports `string`, `int`, `float64`, and `bool` types.

## Configuration Sections

A generated project includes these configuration files:

### `app.go` — Application Settings

| Config Key | Env Variable | Default | Description |
|-----------|-------------|---------|-------------|
| `app.name` | `APP_NAME` | `Lemmego` | Application display name |
| `app.port` | `APP_PORT` | `1490` | HTTP server port |
| `app.env` | `APP_ENV` | `development` | Environment (`production`, `development`) |
| `app.debug` | `APP_DEBUG` | `false` | Enable debug mode |
| `app.url` | `APP_URL` | `http://localhost:1490` | Application URL |

### `database.go` — Database Connections

| Config Key | Env Variable | Default | Description |
|-----------|-------------|---------|-------------|
| `sql.default` | `DB_CONNECTION` | `sqlite` | Default connection name |
| `sql.connections.sqlite.database` | `DB_DATABASE` | `storage/database.sqlite` | SQLite path |
| `sql.connections.libsql.url` | `DB_URL` | `http://127.0.0.1:8080` | [libSQL](/docs/database/libsql/) server |
| `sql.connections.mysql.host` | `DB_HOST` | `127.0.0.1` | MySQL host |
| `sql.connections.mysql.port` | `DB_PORT` | `3306` | MySQL port |

Only the connection the project chose is written, so a SQLite project has no
`mysql` block to ignore.

{{< callout type="warning" >}}
**Your application registers its own database driver.** `DB_DRIVER` names a
driver; it does not supply one. The driver registers itself from an `init`, so
`bootstrap/providers.go` carries a blank import for the one you configured:

```go
_ "github.com/lib/pq"              // postgres
_ "github.com/go-sql-driver/mysql" // mysql
_ "github.com/glebarez/go-sqlite"  // sqlite
_ "github.com/tursodatabase/libsql-client-go/libsql" // libsql
```

`lemmego new` writes exactly the one your project selected. If you change
`DB_DRIVER` later, change that import too, or the connection fails with
`database/sql`'s own message:

```text
sql: unknown driver "postgres" (forgotten import?)
```

This is worth knowing if you are upgrading an older project. `migration`
blank-imported all three until v0.6.0, so every application got them for free
whether or not it had configured one — which also meant linking two drivers
nobody used, the pure-Go SQLite driver alone being twenty-nine packages of
translated libc. It registers none now, and the failure is loud and immediate
rather than silent.
{{< /callout >}}

### `keyvalue.go` — the shared Redis connection

Written when anything in the project uses Redis: the cache, the queue or
sessions. It is one connection that all three fall back to, so a single server
is configured once.

| Config key | Env variable | Default |
|-----------|-------------|---------|
| `keyvalue.connections.redis.host` | `REDIS_HOST` | `localhost` |
| `keyvalue.connections.redis.port` | `REDIS_PORT` | `6379` |
| `keyvalue.connections.redis.password` | `REDIS_PASSWORD` | — |

A subsystem can still point somewhere else by setting its own key —
`CACHE_REDIS_ADDR` or `TASKER_REDIS_ADDR` — which overrides the shared value.

### `cache.go` — the cache

| Config key | Env variable | Default |
|-----------|-------------|---------|
| `cache.driver` | `CACHE_DRIVER` | the driver chosen at scaffold time |
| `cache.prefix` | `CACHE_PREFIX` | `lemmego_cache:` |
| `cache.ttl` | `CACHE_TTL` | `3600` (seconds) |
| `cache.codec` | `CACHE_CODEC` | `json` |
| `cache.lenient` | `CACHE_LENIENT` | `false` |

See [Caching](/docs/cache/) for what each driver can do.

### `tasker.go` — the background queue

| Config key | Env variable | Default |
|-----------|-------------|---------|
| `tasker.driver` | `TASKER_DRIVER` | the driver chosen at scaffold time |
| `tasker.queue` | `TASKER_QUEUE` | `default` |
| `tasker.max_attempts` | `TASKER_MAX_ATTEMPTS` | `3` |
| `tasker.workers.default` | `TASKER_WORKERS` | `3` |
| `tasker.route_prefix` | `TASKER_ROUTE_PREFIX` | `/tasker` |
| — | `TASKER_ADMINS` | empty: nobody may open the dashboard |

With the SQL driver and no `TASKER_DSN`, the queue uses the application's own
database connection, so there is one database rather than two that can drift
apart.

`TASKER_ADMINS` is read by the scaffolded `DashboardAuth` predicate in
`bootstrap/providers.go`, not by the `tasker` config itself — it is a rule your
application owns, so it lives in Go where you can change it to anything. Empty
means the dashboard refuses everyone.

### `filesystems.go` — Storage Disks

| Config Key | Env Variable | Default | Description |
|-----------|-------------|---------|-------------|
| `filesystems.default` | `FILESYSTEM_DISK` | `local` | Default storage disk |
| `filesystems.disks.local.root` | — | `storage/app` | Local storage root |

### `session.go` — Session Configuration

| Config Key | Env Variable | Default | Description |
|-----------|-------------|---------|-------------|
| `session.driver` | `SESSION_DRIVER` | `file` | Session store driver (`memory`, `file`, `redis`, `database`) |
| `session.lifetime` | `SESSION_LIFETIME` | `120` | Session lifetime (minutes) |
| `session.cookie` | `SESSION_COOKIE` | `lemmego_session` | Cookie name |

## Adding Custom Configuration

You can add your own configuration by creating new files in `internal/configs/`:

```go
// internal/configs/mail.go
package configs

import "github.com/lemmego/api/config"

func init() {
    config.Set("mail", config.M{
        "default":  config.MustEnv("MAIL_MAILER", "smtp"),
        "host":     config.MustEnv("MAIL_HOST", "smtp.mailtrap.io"),
        "port":     config.MustEnv("MAIL_PORT", 2525),
    })
}
```

Access your config anywhere in your application:

```go
mailHost := config.Get("mail.host").(string)
```

## Next steps

- [Application bootstrap](/docs/core-concepts/application-bootstrap) — when
  your configuration is read.
- [Environment variables](/docs/appendix/#environment-variables) — every key
  the framework and its modules look for.
- [Deployment](/docs/deployment/) — what to change for production.
