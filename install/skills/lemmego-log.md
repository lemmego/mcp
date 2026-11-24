## When to use this skill

When configuring logging, adding a log line, or working out where a log went.

## There is no `log.Info`

`slog` is the facade. This module configures `slog`'s default handler and then
gets out of the way:

    slog.Info("order placed", "order_id", order.ID, "cents", order.Cents)

Do not look for a `log.Info`, and do not add a wrapper. A second facade means
two ways to log the same thing, and library code calls `slog` whatever an
application prefers — so half the lines in a file would come from the facade
nobody chose.

`lemmego/log` plus the `api/logger` provider is **the one owner of
`slog.SetDefault`**. Calling `slog.SetDefault` anywhere else in an application
takes the configured handler away, and the symptom is formatting that silently
reverts.

## The five channels

| `LOG_CHANNEL` | What you get |
| --- | --- |
| `console` | readable lines on stderr — the development default |
| `file` | JSON in a rotating file |
| `json` | JSON on stderr, for a platform that collects stdio |
| `stack` | console **and** file — the production answer |
| `null` | discards everything, for a test asserting on something else |

A small closed set on purpose: the value comes from an environment variable,
and a typo should name the valid options rather than quietly pick a default.

**`stack` is the production choice.** A container platform collects stdio, so
a deployment whose only log is a file on an ephemeral filesystem has a log
nobody will read; one whose only log is stdio has nothing to grep next week.

The two halves use **different formats** by design — a file is read by `grep`
and `jq`, a console by a person — so neither compromises. Do not try to make
them match.

## Rotation

Defaults: `storage/logs/app.log`, 32 MB per file, 7 files kept, mode `0600`.
The mode matters: a log holds request paths, user ids and sometimes a token in
an error string, and `0644` makes that world-readable on a shared host.

Rotation is by size, not by date. A quiet application can keep one file for
months, which is correct — rotating an empty file gains nothing.

## Composing beyond the five

`NewFanout(handlers...)` writes one record to several handlers. Build handlers
directly when you need something the channel list does not cover; that is the
intended escape hatch rather than adding a sixth channel name.

## Standard library only

This module depends on nothing outside the standard library, and that is load
bearing: every other package wants to import a logger, framework modules
included. If it needed `app.Provider` it would drag the framework's whole
dependency tree into a package whose job is formatting text. So do not add a
framework import here — put anything that needs `app.App` in `api/logger`.

## Reading logs back

Prefer the MCP tools when debugging: `last-error` to start a loop, and
`read-log-entries` for recent lines, filterable. They read the most recently
written file, so they answer nothing useful under `LOG_CHANNEL=json` or
`console` — set `stack` or `file` if an agent is meant to read them.

## Check first

`search-docs` section `logging`, `module-api` on `log`.
