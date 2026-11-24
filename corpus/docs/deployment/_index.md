---
title: Deployment
type: docs
prev: docs/testing/
next: docs/mcp
weight: 152
sidebar:
  open: true
---

Deploying means getting your application onto a machine where other people can
reach it, and keeping it running.

Go makes the first part unusually easy: `go build` produces **a single
executable file** with everything compiled in. There is no runtime to install
on the server, no dependency directory to copy, nothing to install alongside
it — you move one file and run it. That is why most of this page is about
configuration and process management rather than packaging.

Two things do travel separately, and both are covered below: your built
frontend assets, and your environment variables.

## Production build

### Build the Application

```shell
go build -o bin/app ./cmd/app
```

### Build Frontend Assets

```shell
lemmego build
```

This runs `templ generate` (if needed) and `npm run build` / `pnpm build` for production frontend assets.

## Production Configuration

### Environment Variables

Set these for production:

```shell
APP_ENV=production
APP_DEBUG=false
APP_PORT=1490
APP_URL=https://yourdomain.com

# Database
DB_CONNECTION=postgres
DB_HOST=your-db-host
DB_DATABASE=your-db-name
DB_USERNAME=your-db-user
DB_PASSWORD=your-db-password

# Session — see "Sessions on a container platform" below
SESSION_DRIVER=database
SESSION_COOKIE=your_app_session

# Filesystem
FILESYSTEM_DISK=s3
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret
AWS_BUCKET=your-bucket

# Encryption
APP_KEY=base64:your-32-byte-key

# Auth (if using JWT)
JWT_SECRET=your-jwt-secret
```

### Sessions on a container platform

`SESSION_DRIVER` above is `database` rather than `redis`, which is worth
explaining because the wrong choice here is invisible until you deploy.

Of the four drivers, only three are durable, and two of those cost you
infrastructure:

| Driver | Survives a deploy? | Needs |
|---|---|---|
| `memory` | **No** | nothing |
| `file` | Only with a mounted volume | a volume that outlives the container |
| `redis` | Yes | a Redis service |
| `database` | Yes | a `sessions` table |

On a platform where containers are replaced on every deploy, `file` **without
a mounted volume behaves like `memory`**: the store lives in the container
filesystem and dies with it, so every deploy signs out every user. That failure
is quiet — nothing errors, people are simply logged out — so it is easy to ship
and hard to attribute.

If you already run a database, `database` is the cheapest durable answer. See
[Session → Database](/docs/session#database) for the table and the one
provider-ordering requirement.

### Managed Postgres and connection poolers

Managed providers such as Neon and Supabase publish **two** URLs for one
database: a pooled one fronted by PgBouncer in transaction mode, and a direct
one. Their quickstarts point you at the pooled URL.

**Do not run migrations through the pooled URL.** `lib/pq` issues every
parameterised query on the *unnamed* prepared statement over the extended
protocol, and under transaction pooling a `Parse` from one logical connection
can pair with a `Bind` from another. The symptom is intermittent and reads like
a bug in your code:

```text
pq: bind message supplies 2 parameters, but prepared statement "" requires 1
```

The failure is intermittent and can leave a schema half-changed. Pooled
connections are also session-less by design, so an advisory lock, a `SET`, or
a session-scoped temp table applies to whichever backend answered rather than
to your migration.

`migrate` resolves its DSN in this order:

1. `--dsn`
2. `DATABASE_URL_UNPOOLED`
3. `DATABASE_URL`
4. the `DB_*` components

So set both and the right endpoint is chosen for you:

```shell
DATABASE_URL=postgres://...-pooler.../db?sslmode=require
DATABASE_URL_UNPOOLED=postgres://.../db?sslmode=require
```

An explicit `--dsn` still wins, including when you deliberately want the pooled
endpoint. A present-but-empty `DATABASE_URL_UNPOOLED` does not shadow
`DATABASE_URL`.

The same applies to the application itself if it uses `lib/pq`: put the pooling
in `database/sql`, which already holds connections open, and point the app at
the direct endpoint.

### Health checks

`api` ships a health checker and three handlers, and **registers no routes** —
mounting them is the application's job. A project from `lemmego new` does it in
`internal/routes/health.go`:

| Endpoint | Answers | Consumed by |
|---|---|---|
| `/livez` | the process is running | an orchestrator, to restart the container |
| `/readyz` | this instance can serve traffic | a load balancer, to take it out of rotation |
| `/health` | every check, including non-critical ones | a person or a scraper |

The split matters. A liveness probe must not depend on anything external — one
that fails when the database blinks will restart every instance of a healthy
application. A readiness probe checks the critical dependencies and nothing
else. `/health` is where a non-critical degradation shows up.

That is what `Critical` on a check decides:

```go
checker := a.GetHealthChecker()
checker.RegisterDefaultChecks()   // filesystem (non-critical), session (critical)

checker.RegisterCheck("database", app.HealthCheck{
    Name:        "database",
    Description: "The application's database answers",
    Critical:    true,
    Timeout:     3 * time.Second,
    CheckFunc: func(ctx context.Context) error {
        conn, ok := db.Resolve(a)
        if !ok || conn == nil {
            return errors.New("no database is registered")
        }
        return conn.SQLDB().PingContext(ctx)
    },
})
```

Two things that are easy to get wrong:

- **`RegisterDefaultChecks` covers only the filesystem and the session store.**
  A database check is yours to add, and it is the one that makes `/readyz`
  meaningful.
- **Make it a real round trip.** A pool holding a connection object and a
  server that has gone away look identical until something asks. `db.Resolve`
  rather than your ORM's own handle keeps the check the same across connectors.

Register nothing and the endpoints answer `healthy` having verified nothing,
which is a green light wired to no sensor.

### Graceful Shutdown

The framework supports graceful shutdown via `ShutdownProvider`:

```go
type MyProvider struct{}

func (p *MyProvider) Shutdown(ctx context.Context) error {
    // Close database connections, flush logs, etc.
    return nil
}
```

## Deployment Options

### Single Binary

```shell
# Build for your target platform
GOOS=linux GOARCH=amd64 go build -o bin/app ./cmd/app

# Deploy the binary and public/ directory
scp bin/app user@server:/opt/app/
scp -r public user@server:/opt/app/
scp .env user@server:/opt/app/
```

### Docker

```dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build && go build -o bin/app ./cmd/app

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/app .
COPY --from=builder /app/public ./public
COPY --from=builder /app/.env .
EXPOSE 1490
CMD ["./app"]
```

### Process Manager

Use a process manager like supervisor or systemd:

```ini
[Unit]
Description=Lemmego App
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/app
ExecStart=/opt/app/app
Restart=always
EnvironmentFile=/opt/app/.env

[Install]
WantedBy=multi-user.target
```

## Reverse Proxy

For production, run behind a reverse proxy (Nginx, Caddy, Traefik):

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    
    location / {
        proxy_pass http://127.0.0.1:1490;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    location /build/ {
        alias /opt/app/public/build/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

## Next steps

- [Running migrations](/docs/migrations/running-migrations) — migrating a
  database you cannot restart.
- [Configuration](/docs/configuration) — the settings that change for
  production.
- [Logging](/docs/logging) — where output goes once there is no terminal.
