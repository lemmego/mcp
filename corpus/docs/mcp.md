---
title: AI Agents (MCP)
type: docs
prev: docs/deployment/
next: docs/prompting
weight: 153
sidebar:
  open: true
---

`lemmego/mcp` serves your application's own state to an AI coding agent over the
[Model Context Protocol](https://modelcontextprotocol.io), and installs the
guidelines, skills and rules that give the agent context.

```shell
lemmego mcp:install
```

A project created by `lemmego new` already has the server registered. For an
existing project, see [Adding it to an existing project](#adding-it-to-an-existing-project).

## Start with these two tools

An agent editing your project cannot see the routing table, the live schema,
the resolved configuration, the migration state or the logs, so it guesses.
The costly guesses are about **APIs**: a call written against a function that
does not exist survives until somebody compiles it.

Three tools address that directly, and they are worth naming in your agent's
instructions:

| Tool | What it gives the agent |
| --- | --- |
| **`module-api`** | The real exported signatures of a lemmego module at the version your project pins, read from the compiled package |
| **`search-docs`** | The documentation site's pages, compiled into the server so they are available whether or not you have a checkout of them |
| **`list-modules`** | Every module the ecosystem publishes, including the ones your `go.mod` does not mention yet |

The rest supply context about your running application.

`list-modules` covers the question the other two cannot. `module-api` runs
inside your project, so a module you have not added answers `no required
module provides package …` — which reads as "there is no such thing". An agent
then writes multi-tenancy, or roles and permissions, from scratch beside a
published module that already does it.

## All tools

<div class="overflow-auto">

| Name | Notes |
| --- | --- |
| `module-api` | Real exported signatures at the pinned version; pass a symbol for one declaration with its doc comment |
| `search-docs` | Search the documentation site, returning the matching sections |
| `list-modules` | Every published module, what it is for, and how to add the ones this project lacks |
| `app-info` | Go version, environment, live database dialect, the driver chosen for each subsystem, every lemmego module and its resolved version |
| `config` | Resolved configuration, **secrets withheld**; one key or the whole tree |
| `list-routes` | Every registered route with method, resolved path, handler and middleware counts; filterable by method or path |
| `absolute-url` | A path turned absolute against the configured `APP_URL` |
| `database-connections` | Which database is actually connected, the DSN with credentials withheld, live pool statistics |
| `database-schema` | Live tables, or one table's columns with types, nullability and defaults |
| `database-query` | One statement. **Read-only by default** |
| `migration-status` | Applied migrations from `schema_migrations`, newest first |
| `health` | The registered health checks and each result |
| `read-log-entries` | Recent log lines, filterable, from the most recently written file |
| `last-error` | The most recent error-level entry |
| `record-rule` | File a durable project rule under `.ai/rules` and rebuild its index |

</div>

### Not available

- **Code execution.** Go has no `eval`, so there is no equivalent of a
  `tinker` tool.
- **Browser logs.** These need a JavaScript shim in the page.
- **An HTTP transport.** See [Safety](#safety).

## Safety

This is a local development tool.

**stdio only.** An MCP server on a port is an unauthenticated connection to
your database for anything that can reach it. There is no HTTP transport and
adding one is not a planned feature.

**Read-only by default.** `database-query` accepts only `SELECT`, `WITH`,
`SHOW`, `EXPLAIN`, `TABLE` and `VALUES`, and only one statement per call.

The guard is an **allowlist, not a denylist**, because a denylist has to
enumerate every way SQL can change something and will miss one — `TRUNCATE`,
`GRANT`, `COPY`, `VACUUM`, `REINDEX`, `CLUSTER`, `COMMENT`, `SECURITY LABEL`,
`DO`, `CALL`, `REFRESH MATERIALIZED VIEW`, and whatever the next server version
adds. Getting that list wrong means an agent drops a table. Stacked statements
are refused outright, because a head-only check would pass
`SELECT 1; DROP TABLE users`.

To lift it:

```go
&lemmegomcp.Provider{Options: []lemmegomcp.Option{lemmegomcp.WithWrites()}}
```

or `lemmego run mcp:serve --allow-writes`, which warns on stderr.

**Secrets are withheld.** Configuration is redacted by key pattern —
`password`, `secret`, `token`, `key`, `credential`, `salt`, `dsn` and more —
walking nested maps, because `sql.connections.pgsql.password` is where a
password actually lives.

It also **rewrites a credential found inside a value**, which is the case a
key-based rule alone gets wrong and is not hypothetical: `DATABASE_URL` is the
standard name for a Postgres connection string, its name contains no
secret-looking fragment, and its value contains the password. The host, user
and database survive; the password does not.

## Adding it to an existing project

`lemmego new` registers the provider for you. An existing project has nobody to
do it — Go has no equivalent of Composer's package auto-discovery — so there
are two steps, and `mcp:install` tells you if either is missing.

```shell
go get github.com/lemmego/mcp
```

Then register it **last** in `bootstrap/providers.go`:

```go
import lemmegomcp "github.com/lemmego/mcp"

func LoadProviders() []app.Provider {
    providers := []app.Provider{
        // ...everything else...
    }

    // A development tool: it serves this application's configuration and
    // database to whatever starts it.
    if config.MustEnv("APP_ENV", "development") != "production" {
        providers = append(providers, &lemmegomcp.Provider{})
    }

    return providers
}
```

Last, because it reports on everything the providers above it register — the
routes, the connection, the session store, the health checks — and listed
earlier it would resolve nothing.

**Not in production**, because it serves your configuration and database to
whatever starts it. It is stdio-only, so this is not a hole being closed; it is
a development tool, and a production binary should not carry a command that
reads the database. Remove the guard if you want it everywhere.

Then `lemmego mcp:install` and restart your editor.

## What `mcp:install` writes

<div class="overflow-auto">

| File | Contents |
| --- | --- |
| `.mcp.json`, `.cursor/mcp.json`, `.codex/mcp.json` | how to start the server |
| `CLAUDE.md`, `AGENTS.md`, `.cursor/rules/lemmego.mdc` | always-on guidelines, inside a marked block |
| `.ai/skills/*/SKILL.md` | on-demand skills, for the modules the project actually uses |
| `.ai/rules/index.md` | the index recorded rules are filed into |

</div>

Agents: `claude-code`, `cursor`, `codex`, `copilot`, `windsurf`. Narrow with
`--agent`, and skip parts with `--no-skills`, `--no-rules`, `--no-guidelines`.

**Only the marked block of a guideline file is replaced.** `CLAUDE.md` is a
file people write in by hand, so everything outside
`<!-- BEGIN lemmego-guidelines -->` and its `END` survives, and reinstalling
replaces the block rather than appending a second one.

**An existing MCP config is merged, not replaced**, because several agents
share `.mcp.json`. A config that exists and cannot be parsed is left strictly
alone and reported — it is your work, and discarding it to fix a syntax error
is not the tool's call.

**Recorded rules are never overwritten.** `.ai/rules` should be committed:
those are your team's decisions, unlike an agent's own memory, which is
personal and session-scoped. The generated config and guideline files can be
gitignored.

## Guidelines, skills and rules

Three layers, loaded at different times and costing different amounts:

<div class="overflow-auto">

| | Loaded | Holds |
| --- | --- | --- |
| **Guidelines** | upfront, every request | what an agent needs *before* it knows what it is doing |
| **Skills** | on demand | detail for one domain |
| **Rules** | by glob, when editing a matching file | decisions about *your* application |

</div>

Guidelines are kept short, because everything loaded upfront costs context on
every single request.

Skills ship only for the modules your project depends on. Rules are recorded
through the `record-rule` tool rather than written by hand, because the index
agents consult is regenerated there — a rule file added manually is never
discovered.

## Troubleshooting

### The server shows as failed, with a red cross

The configuration points at `lemmego run mcp:serve` and your project has no
such command, which means the provider is not registered. Re-run
`lemmego mcp:install`: it checks for the module and the provider and prints
exactly what is missing. See
[Adding it to an existing project](#adding-it-to-an-existing-project).

### `CONNECTION_CLOSED`

Update the CLI:

```shell
go install github.com/lemmego/cli/cmd@latest
```

The server speaks over stdin, so it needs a CLI that forwards it —
`cli` v0.5.5 or later. On an older CLI the server starts, reads EOF
immediately, treats that as the client disconnecting and exits without
writing anything, which the editor reports as a server that will not start.

### Nothing may write to stdout but protocol frames

If you add a tool, remember that one stray `fmt.Println` corrupts the stream
and the editor reports a server that will not start, saying nothing about the
print. Use stderr.

## Next steps

- [Request debugging](/docs/observe/) — the recorder most of these tools read.
- [Logging](/docs/logging) — the log entries an agent can ask for.
