---
title: Queue & Background Jobs
type: docs
prev: docs/cache/events
next: docs/mail/
weight: 126
sidebar:
  open: true
---

Somebody signs up and you send them a welcome email. Your mail server takes
four seconds. They sat and watched a loading spinner for four seconds, and if
the mail server had been down they would have seen an error for something that
worked.

The fix is not to do it during the request. You write down that the email
needs sending, answer immediately, and something else does the work a moment
later. That list of things-to-do is a **queue**, each entry on it is a **job**,
and the separate process that picks them up and runs them is a **worker**.

```go
queue.Dispatch(ctx, SendWelcomeEmail{UserID: user.ID})   // returns at once
```

{{< callout type="warning" >}}
**A job does nothing until a worker is running.** The queue is just a list;
`lemmego run tasker:work` is what empties it. If your jobs seem to vanish, the
first thing to check is whether a worker is up.
{{< /callout >}}

Anything slow or failure-prone belongs here: email, image processing, calling
somebody else's API, nightly cleanups.

## Quick start

The queue provider is already registered in new projects:

```go
// bootstrap/providers.go
&queue.Provider{}
```

With no config, it auto-reads the database connection from your existing `configs/database.go`. Jobs are stored in the same database as your app.

## Defining a Job

Jobs implement the `Job` interface:

```go
package jobs

import (
    "context"
    "fmt"
    "github.com/lemmego/queue"
    "github.com/lemmego/tasker"
)

type SendWelcomeEmail struct {
    UserID int    `json:"user_id"`
    Email  string `json:"email"`
}

func (j *SendWelcomeEmail) Handle(ctx context.Context) error {
    fmt.Printf("Sending email to %s\n", j.Email)
    return nil
}

func init() {
    queue.RegisterJob("*jobs.SendWelcomeEmail", func() tasker.Job {
        return &SendWelcomeEmail{}
    })
}
```

## Dispatching Jobs

```go
queue.Dispatch(ctx, &SendWelcomeEmail{
    UserID: 42,
    Email:  "user@test.com",
})

// With options
queue.Dispatch(ctx, &SendWelcomeEmail{UserID: 43},
    tasker.OnQueue("email"),
    tasker.WithDelay(5*time.Minute),
    tasker.WithTags("priority"),
)
```

## Running Workers

```bash
lemmego run tasker:work --queue=default,email --workers=3
```

Workers claim jobs from the database atomically and process them. Run multiple worker processes across different machines — they coordinate through the shared database.

## Optional Job Interfaces

Customize job behavior by implementing optional interfaces:

| Interface | Purpose |
|-----------|---------|
| `ShouldQueue` | Set a custom queue name |
| `ShouldDelay` | Delay job execution |
| `ShouldRetry` | Max retry attempts |
| `ShouldFail` | Called when all retries exhausted |
| `ShouldTag` | Tags for dashboard filtering |
| `ShouldBatch` | Group jobs into batches |
| `ShouldTimeout` | Per-job timeout |
| `BeforeHandleHook` | Run before job execution |
| `AfterHandleHook` | Run after successful execution |
| `BeforeRetryHook` | Run before each retry |

## Retry & Backoff

By default, jobs use exponential backoff: `2^attempt * 2s`, max 24h. Max 3 attempts. Configure per job:

```go
func (j *MyJob) MaxAttempts() int { return 5 }
```

Failed jobs appear on the dashboard with full error stack traces. Click "Retry" to re-queue them.

## Worker Pools

Configure pools per queue:

```go
&queue.Provider{
    Config: &queue.Config{
        Workers: map[string]int{
            "default": 3,
            "email":   5,
            "images":  10,
        },
    },
}
```

Enable auto-scaling to dynamically adjust workers based on queue backlog:

```go
Config: &queue.Config{
    EnableAutoscale: true,
}
```

## Scheduling Recurring Jobs

```go
scheduler.New(mgr).Register(scheduler.ScheduledJob{
    ID:       "daily-report",
    Schedule: "0 6 * * *",  // every day at 6 AM
    Job:      &GenerateReport{},
    Queue:    "default",
})
```

## Web Dashboard

The dashboard lives at `http://localhost:1490/tasker/`, and it is **closed
until you say who may use it**. Visiting it before then answers 401.

That is deliberate rather than an oversight. The dashboard can retry, cancel
and delete jobs and pause a queue, so the failure mode of an open default is a
public button that clears somebody's work.

```go
&queue.Provider{
    DashboardAuth: func(c app.Context) bool {
        if err := auth.Check(c); err != nil {
            return false
        }
        user, ok := auth.UserAs[*models.User](c)
        return ok && slices.Contains(taskerAdmins(), user.Email)
    },
}
```

Return `true` to let the request through, `false` for 401.

{{< callout type="warning" >}}
**Call `auth.Check` first.** The dashboard is mounted as a raw
`http.Handler`, so none of the router's middleware runs for it and nothing has
looked at the request's cookie by the time your predicate is called. Leave it
out and the user is always absent, so the dashboard refuses everyone —
including the administrators you just listed.
{{< /callout >}}

In a scaffolded project this is already wired, reading a comma-separated list
from `TASKER_ADMINS`:

```bash
TASKER_ADMINS=ada@example.com,grace@example.com
```

`auth.Check` resolves a session cookie and a bearer token alike, so the same
rule covers a browser and an API client — see
[The Current User](/docs/security/current-user). And nothing obliges the rule
to involve a user at all; a shared secret header or an IP allowlist is just as
valid, which is why `DashboardAuth` is a predicate over the request rather
than over a user:

```go
DashboardAuth: func(c app.Context) bool {
    return subtle.ConstantTimeCompare(
        []byte(c.Header("X-Tasker-Key")), []byte(key)) == 1
},
```

CSRF protection is enabled alongside it automatically, since the dashboard
posts from a browser.

Once open:

| Feature | Description |
|---------|-------------|
| Stat cards | Status, jobs/min, processes, failed count |
| Queues table | Available/running/completed/failed per queue, pause/resume |
| Jobs table | Filterable by state (failed, retryable, running, etc.) |
| Job detail | Payload, errors, execution timeline, output |
| Actions | Retry, cancel, batch retry, batch cancel |
| Workers | Connected nodes with uptime |
| Metrics | Per-job-type throughput, avg runtime, fail rate |
| Live updates | Auto-refresh every 10 seconds + SSE real-time stats |

## API Endpoints

All endpoints are mounted under `/tasker/`, and every one of them is behind
`DashboardAuth` — including the `POST`s that retry, cancel and prune:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/tasker/` | Dashboard HTML |
| `GET` | `/tasker/api/stats` | Global stats |
| `GET` | `/tasker/api/jobs` | Paginated job list |
| `GET` | `/tasker/api/jobs/{id}` | Job detail |
| `POST` | `/tasker/api/jobs/{id}/retry` | Retry a job |
| `POST` | `/tasker/api/jobs/{id}/cancel` | Cancel a job |
| `POST` | `/tasker/api/jobs/batch/retry` | Batch retry |
| `POST` | `/tasker/api/jobs/batch/cancel` | Batch cancel |
| `GET` | `/tasker/api/queues` | Queue metrics |
| `POST` | `/tasker/api/queues/{queue}/pause` | Pause a queue |
| `POST` | `/tasker/api/queues/{queue}/resume` | Resume a queue |
| `GET` | `/tasker/api/workers` | Connected nodes |
| `GET` | `/tasker/api/metrics/jobs` | Per-job-type metrics |
| `GET` | `/tasker/api/events` | SSE real-time stream |
| `POST` | `/tasker/api/prune` | Prune old jobs |

## Configuration

### Via provider options

```go
&queue.Provider{
    Config: &queue.Config{
        Driver:             "sql",         // or "redis"
        DSN:                "postgres://...",
        DriverName:         "postgres",
        TablePrefix:        "tasker_",
        RoutePrefix:        "/tasker",
        DefaultQueue:       "default",
        DefaultMaxAttempts: 3,
        Workers:            map[string]int{"default": 3},
    },
}
```

### Via .env

A scaffolded project gets `internal/configs/tasker.go`, which reads these.
Before the queue had a config file, every `TASKER_*` variable was ignored and
the queue ran on its built-in defaults.

```env
TASKER_ROUTE_PREFIX=/jobs
TASKER_ADMINS=ada@example.com,grace@example.com
TASKER_TABLE_PREFIX=myapp_
TASKER_QUEUE=default
TASKER_MAX_ATTEMPTS=3
TASKER_HEARTBEAT_SEC=5
TASKER_REQUEUE_SEC=60
TASKER_PRUNE_HOURS=168
TASKER_AUTOSCALE=false
```

## Which database the queue uses

With the SQL driver and no `TASKER_DSN`, the queue uses the application's own
connection, so jobs live in the same database as everything else and there is
one thing to back up.

Point it somewhere else with `TASKER_DSN` if you want the queue isolated — but
be careful what you point it at. A queue on an in-memory database accepts jobs
and loses every one of them on restart, silently: the tables are recreated
empty and nothing reports that anything was dropped. The queue logs a warning
when it detects one.

## Driver Support

### SQL (PostgreSQL / MySQL / SQLite)

Default driver. Stores jobs in a `tasker_jobs` table with atomic claiming using `SELECT ... FOR UPDATE SKIP LOCKED`. Supports all SQL databases.

### Redis

Alternative driver for node coordination, leader election, and distributed locking (job queue implementation uses database).

## Architecture

```
   ┌──────────────┐     ┌──────────────┐
   │   Web App    │     │  Worker 1    │
   │  Dispatch    │     │  Supervisor  │
   │  Dashboard   │     │  └─Pool (3)  │
   └──────┬───────┘     └──────┬───────┘
          │                    │
          └────────┬───────────┘
                   │
          ┌────────▼────────┐
          │   PostgreSQL    │
          │  (jobs table)   │
          │  SELECT FOR     │
          │  UPDATE SKIP    │
          │  LOCKED         │
          └─────────────────┘
```

Multiple workers share the same database — they claim available jobs atomically. No direct inter-node communication needed. Scale by adding more worker processes.

## CLI Commands

```bash
lemmego run tasker:work            # Start worker
  --queue=default,email            #   Queues to listen on
  --workers=3                      #   Workers per queue

lemmego run tasker:dispatch        # Dispatch test jobs
  --count=10
  --queue=default
  --fail-rate=30                   # % chance of failure
  --delay=500                      # Simulated processing delay (ms)

lemmego run tasker:job retry 5     # Retry job #5
lemmego run tasker:job cancel 5    # Cancel job #5

lemmego run tasker:queue list      # List queue stats
```

## Next steps

- [Mail](/docs/mail/) — the queued sending this worker performs.
- [Notifications](/docs/notifications/) — multi-channel sending over the same
  worker.
- [The connection seam](/docs/database/connection-seam/) — the connection the
  SQL driver borrows.
