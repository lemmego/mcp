# lemmego/mcp

Serves a Lemmego application's own state to an AI coding agent over the
[Model Context Protocol](https://modelcontextprotocol.io), and installs the
guidelines, skills and rules that give the agent context.

```shell
go get github.com/lemmego/mcp
lemmego mcp:install          # configure your editor
```

## The problem this solves

An agent editing a Lemmego project cannot see the routing table, the live
schema, the resolved configuration, the migration state or the logs. So it
guesses — and the expensive guesses are not about data, they are about **APIs**.
It writes a call against a function that does not exist, and the mistake
survives until somebody compiles it.

That is not hypothetical here. The logging documentation in this project was
once fiction throughout: every sample called `logger.Default`, `logger.Verbose`
or `logger.NewHandler`, and the observability section named struct fields that
never existed. An agent reading it would write code that could not build.

So the two tools to reach for first are not the database ones:

| Tool | What it does |
| --- | --- |
| **`module-api`** | The **real** exported signatures of a lemmego module at the version this project pins, read from the compiled package via `go doc`. Prose goes stale and invents functions; this cannot. |
| **`search-docs`** | The modules' own READMEs, which are the authoritative spec for each module. |

Everything else is context. Those two are correction.

## All tools

| Name | Notes |
| --- | --- |
| `module-api` | Real exported signatures at the pinned version; pass a symbol for one declaration with its doc comment |
| `search-docs` | Search the module READMEs, returning matching sections |
| `app-info` | Go version, environment, live database dialect, driver per subsystem, every lemmego module and its resolved version |
| `config` | Resolved configuration, **secrets withheld**; one key or the whole tree |
| `list-routes` | Every registered route with method, resolved path, handler and middleware counts; filterable |
| `absolute-url` | A path turned absolute against the configured `APP_URL` |
| `database-connections` | Which database is actually connected, with the DSN redacted, plus live pool stats |
| `database-schema` | Live tables, or one table's columns with types, nullability and defaults |
| `database-query` | One statement. **Read-only by default** |
| `migration-status` | Applied migrations from `schema_migrations`, newest first |
| `health` | The registered health checks and each result |
| `read-log-entries` | Recent log lines, filterable, from the most recently written file |
| `last-error` | The most recent error-level entry — start a debugging loop here |
| `record-rule` | File a durable project rule under `.ai/rules` and rebuild its index |

### What is deliberately absent

- **No code execution.** Laravel Boost has a `tinker` tool; Go has no `eval`, so
  there is no honest equivalent and a half-built one would be worse than none.
- **No browser logs.** That needs a JavaScript shim in the page. Worth having,
  not in v0.
- **No HTTP transport.** See Safety.

## Safety

This is a local development tool and it is shaped like one.

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
&mcp.Provider{Options: []mcp.Option{mcp.WithWrites()}}
```

or `lemmego run mcp:serve --allow-writes`, which prints a warning to stderr.

**Secrets are withheld.** Configuration goes through `RedactMap`, which denies
by key pattern — `password`, `secret`, `token`, `key`, `credential`, `salt`,
`dsn` and more — and walks nested maps, because
`sql.connections.pgsql.password` is where a password actually lives.

It also **rewrites a credential found inside a value**, which is the case a
key-based rule alone gets wrong and is not hypothetical: `DATABASE_URL` is the
standard name for a Postgres connection string, its name contains no
secret-looking fragment, and its value contains the password. The host, user
and database survive; the password does not.

Deny-by-pattern rather than allow-by-list is deliberate. An allowlist of safe
keys has to be extended for every new configuration block, and the cost of
forgetting is a leaked credential — where the cost of over-redaction is an
agent asking you for a value.

## Wiring it up

`lemmego mcp:install` does this for you. By hand:

```go
// bootstrap/providers.go — last, so everything it reports on is registered
func LoadProviders() []app.Provider {
    return []app.Provider{
        // ...
        &mcp.Provider{},
    }
}
```

Then point your editor at `lemmego run mcp:serve`.

**Why a command and not a route.** The server has to *be* the application: it
reports the routes, schema and configuration this project actually has, which
means compiling this project's code. A tool that re-derived the DSN from the
environment would report on a different database than the one the application
is using — which is the bug `queue` had before it borrowed `db.Connection`.

**Nothing may write to stdout but protocol frames.** One stray `fmt.Println` in
a tool corrupts the stream, and the editor reports a server that will not start
rather than anything about the print. Use stderr.

## `mcp:install`

Writes, per agent:

| File | Contents |
| --- | --- |
| `.mcp.json`, `.cursor/mcp.json`, `.codex/mcp.json` | how to start the server |
| `CLAUDE.md`, `AGENTS.md`, `.cursor/rules/lemmego.mdc` | always-on guidelines, inside a marked block |
| `.ai/skills/*/SKILL.md` | on-demand skills, for the modules the project actually uses |
| `.ai/rules/index.md` | the index recorded rules are filed into |

Agents: `claude-code`, `cursor`, `codex`, `copilot`, `windsurf`. Narrow with
`--agent`, and skip parts with `--no-skills`, `--no-rules`, `--no-guidelines`.

### Installing does not create the server

Writing an editor's configuration and *having* a server are different things,
and only the first is the installer's doing.

`lemmego new` registers the provider for you. An **existing** project has
nobody to do it: Go has no equivalent of Composer's package auto-discovery,
which is what lets Laravel's installer assume the server is there. So
`mcp:install` checks for both halves — the module in `go.mod` and the provider
in `bootstrap/providers.go` — and prints exactly what is missing.

Without that check it wrote a perfectly good `.mcp.json` pointing at a command
that did not exist, and the editor showed a red cross with no reason given.

**Only the marked block of a guideline file is replaced.** `CLAUDE.md` is a
file people write in by hand, and a tool that overwrote it would lose their
work. Everything outside `<!-- BEGIN lemmego-guidelines -->` and its `END`
survives, and reinstalling replaces the block rather than appending a second.

**An existing MCP config is merged, not replaced**, because several agents share
`.mcp.json` and configuring two in turn must not have each erase the other. A
config that exists and cannot be parsed is left strictly alone and reported: it
is somebody's work, and discarding it to fix a syntax error is not this tool's
call.

**Recorded rules are never overwritten.** `.ai/rules` should be committed —
those are the team's decisions, unlike your agent's own memory, which is
personal and session-scoped. The generated config and guideline files can be
gitignored; they are regenerated.

### Why install does not boot the application

`lemmego mcp:install` is a **CLI** command, not `lemmego run mcp:install`.
`run` does `go run ./cmd/app`, which boots the application — and a freshly
scaffolded project whose database credentials are not filled in cannot boot.
That is exactly when somebody wires up their editor.

This repository paid for the inverse once: `lemmego new` generated its app key
by running the project's own `appkey` command through `run`, the boot failed on
missing credentials, and **every new project started life with no key at all**,
unable to authenticate anybody.

So `mcp/install` is a separate package that imports **only the standard
library** — no `api/app`, no MCP SDK, no cobra — which is what lets
`lemmego/cli` import it without dragging the framework into a CLI that
deliberately keeps it out. That is measured by `install/deps_test.go` rather
than intended.

## Guidelines, skills and rules

Three layers, because they are loaded at different times and cost different
amounts:

| | Loaded | Holds |
| --- | --- | --- |
| **Guidelines** | upfront, every request | what an agent needs *before* it knows what it is doing |
| **Skills** | on demand | detail for one domain |
| **Rules** | by glob, when editing a matching file | decisions about *your* application |

Guidelines are short on purpose: everything loaded upfront costs context on
every single request. They carry the tool list and the handful of framework
facts counter-intuitive enough to cause real damage — that provider order is
load-bearing, that a 200 response proves nothing under Inertia, that a test
suite skips silently without `APPTEST_POSTGRES_DSN`.

Skills ship only for the modules a project depends on, and only where this
project has watched an agent get it wrong: `migration` (the schema builder's
sharp edges on Postgres), `apptest` (the ways a suite passes without running),
`inertia` (browser globals during render, the double `<title>`).

## Testing

```shell
go test ./...
GOWORK=off go mod tidy && GOWORK=off go test ./...
```

The dependency guard in `install/` is the one to watch: it asserts the
installer reaches nothing outside the standard library, which is the whole
reason the package split exists.
