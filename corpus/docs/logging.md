---
title: Logging
type: docs
prev: docs/migrations/running-migrations
next: docs/observe/
weight: 90
sidebar:
  open: true
---

When something goes wrong in production you cannot put a breakpoint in it. All
you have is whatever your application wrote down at the time — which is what
**logs** are for, and why it is worth writing them deliberately rather than
reaching for `fmt.Println`.

Two things make a log useful later. **Levels**, so you can keep every detail
in a file while showing only problems on a terminal: `debug`, `info`, `warn`,
`error`. And **structure** — writing values as named fields rather than
gluing them into a sentence, so you can search for every line about one user
instead of grepping prose.

In Lemmego, logging is just Go's own `log/slog`. There is nothing to wrap, and
nothing new to import:

```go
import "log/slog"

slog.Info("server started", "port", 1490)
slog.Warn("slow request", "duration", "2.5s")
slog.Error("database connection failed", "error", err)
```

There is no `logger.Info`. A package that logs should be able to
import `log/slog` and nothing else, so it works outside an application too —
in a test, in a command, in a library you later extract.

What Lemmego provides is the part `slog` leaves to you: a configured handler,
installed as the process default. That is `logger.Provider`, which configures
[`github.com/lemmego/log`](https://github.com/lemmego/log) from the
environment.

```go
// bootstrap/providers.go
func LoadProviders() []app.Provider {
	return []app.Provider{
		&logger.Provider{},
		// everything else
	}
}
```

Register it **first**. Every provider after it logs while it boots, and a
record emitted before it runs goes to `slog`'s bare default — not the file,
format or level the application asked for.

## Channels

A channel is where records go. One variable picks it:

| `LOG_CHANNEL` | Where records go |
|---|---|
| `console` | Readable, coloured lines, split across stdout and stderr. The development default. |
| `file` | JSON in a rotating file. |
| `json` | JSON on stderr, for a platform that collects stdio. |
| `stack` | Console **and** file together. The production default. |
| `null` | Discarded. |

`stack` is the production default rather than `file` because a container
platform collects stdio, and a deployment whose only log is a file inside an
ephemeral filesystem has a log nobody will read. Both is the answer that works
in both places.

The console channel checks whether it is writing to a terminal before it
colours anything, and honours `NO_COLOR` and `TERM=dumb`. So
`make run > app.log` gives you a readable file rather than one full of escape
sequences.

### Which stream a line goes to

For a server, warn and above go to **stderr** and everything below to
**stdout**. For a CLI command, the whole console channel stays on stderr.

The reason is the platform rather than the logger. Railway, Cloud Run and
Heroku classify a line by the stream it arrived on, so a server that writes
info to stderr has every routine line recorded as an error — and severity
filtering stops meaning anything exactly when somebody is hunting a real one.
A command has the opposite need, because its stdout is the output somebody ran
it for, and that is what keeps `cmd > out.txt` useful.

`LOG_SPLIT_STREAMS` overrides the choice, for the case `LOG_QUIET` already
exists for: a worker or a scheduler is a long-running server that happens to
be spelled as a subcommand, so the console/command distinction answers the
wrong question for it.

```shell
LOG_SPLIT_STREAMS=true           # on a worker or scheduler
```

Setting `Console` to a writer of your own disables the split, since naming one
writer and then dividing it in two would disregard the choice.

Requires `lemmego/log` v0.3.0 and `lemmego/api` v0.13.0.

## Configuration

Every variable has a working default, so the whole block can be left out. An
empty value is treated as absent rather than as a choice — a stray
`LOG_CHANNEL=` line does not quietly demote production to the console — and a
value that cannot be read falls back to the default and says so once, through
the logger it just configured.

```shell
LOG_CHANNEL=console
LOG_LEVEL=debug                  # debug, info, warn, error
LOG_FILE=storage/logs/app.log
LOG_DAILY=true                   # put the date in rotated filenames
LOG_MAX_SIZE=33554432            # rotate past 32 MiB
LOG_MAX_FILES=7                  # keep this many rotated files
LOG_SOURCE=true                  # add file:line to each record
LOG_SPLIT_STREAMS=true           # warn+ to stderr, the rest to stdout
```

`LOG_LEVEL` falls back to `APP_DEBUG`: with `APP_DEBUG=true` and no
`LOG_LEVEL`, the level is debug.

To fix something in code and leave the rest configurable, set the field and
nothing else:

```go
&logger.Provider{Options: log.Options{Channel: log.ChannelStack}}
```

A zero field falls back to the environment, then to the default.

## Rotation

The file channel rotates on **size and date**, because the two answer
different questions: a size limit stops a logging loop filling the disk
overnight, and a date in the filename makes yesterday's log findable.

Rotated files are named `app-2006-01-02-150405.log` beside the active one, and
`LOG_MAX_FILES` prunes the oldest. Log files are created `0600` inside a `0700`
directory — a log holds whatever you logged, which is usually more than you
meant to make world-readable.

A write that fails does not panic and does not silently vanish: it falls back
to stderr and says why, once.

## Commands

A CLI command's own output is why somebody ran it, and info-level boot chatter
buries it. When the process is a command rather than a web server, the console
is raised to warn — and the **file is left at full level**, so a command that
fails overnight still leaves a complete record.

## Tests

A test boot writes nowhere. `apptest` boots the real provider list from the
project root, so without that override every test would append to the
application's own log file: shared mutable state between the suite and
whatever else is running, plus megabytes of boot records that are not about
any one test.

A test that wants to assert on logging says so, and that wins:

```go
&logger.Provider{Options: log.Options{Channel: log.ChannelConsole, Console: buffer}}
```

## Reaching the logger

Ordinary logging goes through `slog`. For the two things `slog` cannot tell
you — where the file is, and how to close it:

```go
if l, ok := logger.Logger(a); ok {
	fmt.Println("logging to", l.Path())
}
```

The application announces the path once at boot for the same reason.

## Composing handlers

`lemmego/log` exports the pieces, so a deployment that needs something the
channels do not express builds it directly — a fan-out is any number of
handlers, each getting its own copy of the record:

```go
rotator, err := log.NewRotator(log.RotateOptions{
	Path:     "storage/logs/audit.log",
	MaxFiles: 30,
	Daily:    true,
})
if err != nil {
	return err
}

slog.SetDefault(slog.New(log.NewFanout(
	log.NewConsoleHandler(os.Stderr, log.ConsoleOptions{Level: slog.LevelInfo}),
	slog.NewJSONHandler(rotator, &slog.HandlerOptions{Level: slog.LevelDebug}),
)))
```

`lemmego/log` depends on nothing but the standard library, which is the point:
a package that formats text should not cost its importers a cloud SDK.

## Seeing what a request did

A log file tells you what happened; it cannot tell you which request each
line belongs to. When you need that — every query a page ran, with timings,
and the log lines between them — reach for
[`lemmego/observe`](https://github.com/lemmego/observe).

```go
// bootstrap/providers.go
&logger.Provider{Options: log.Options{
    // Log lines land under the request that produced them.
    Extra: []slog.Handler{observeslog.Handler()},
}},
&observeprovider.Provider{
    DashboardAuth: func(c app.Context) bool { return isStaff(c) },
},
```

It records requests, SQL statements, cache operations, jobs and mail, and it
reads three ways: a dashboard at `/observe`, `lemmego run observe:last` in a
terminal, and a set of MCP tools so an agent can ask the same questions.

Set `OBSERVE_ENABLED=true` to turn it on — it is off by default, because the
entries hold request headers, bound query parameters and SQL.

{{< callout type="warning" >}}
**`DashboardAuth` is not optional.** Leaving it nil is not "no
authentication", it is "no access": the dashboard answers 401 to everything
until you give it a predicate. That default is deliberate, because the
alternative is a page that hands anybody who can reach it your headers and
your statements.
{{< /callout >}}

## Metrics

There is no metrics collector in the framework. `lemmego/observe` records what
individual requests did, which is a different job from counting them — export
to Prometheus or OpenTelemetry from your own middleware when you need
aggregates.

## Next steps

- [Request debugging](/docs/observe/) — everything one request did, with
  timings.
- [Error handling](/docs/http/error-handling) — what reaches the log when a
  handler fails.
- [Deployment](/docs/deployment/) — where logs go in production.
