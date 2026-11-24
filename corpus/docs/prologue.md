---
title: Prologue
type: docs
prev: /docs
next: docs/glossary
weight: 10
sidebar:
  open: true
---

Lemmego is a full-stack web framework for Go. You write handlers, models and
migrations in Go; you render with Go templates, Templ, or React and Vue over
Inertia; and you reach a database through a type-safe layer that keeps its
types at compile time rather than behind `any`.

This page is a map. If you would rather start typing,
[install the CLI](/docs/installation/) and come back.

{{< callout type="info" >}}
**New to web frameworks?** The [Glossary](/docs/glossary) defines every term
these docs use — route, handler, middleware, provider, migration, session — in
plain language, saying what each thing is and why it exists. Keep it open in a
tab; pages link into it on first use.
{{< /callout >}}

## Start here

Read these five in order and you will have a running application you
understand. It takes about half an hour.

{{< cards >}}
{{< card link="../installation" title="1. Installation" subtitle="Install the lemmego CLI" icon="terminal" >}}
{{< card link="../quick-start" title="2. Quick start" subtitle="Create, run and hot-reload a project" icon="fire" >}}
{{< card link="../project-structure" title="3. Project structure" subtitle="What each directory is for" icon="folder-tree" >}}
{{< card link="../configuration" title="4. Configuration" subtitle="Config files, .env and defaults" icon="adjustments" >}}
{{< card link="../http" title="5. The HTTP layer" subtitle="Routing, middleware, requests, responses" icon="annotation" >}}
{{< /cards >}}

## Then build something

The sidebar runs in the order most people need things. Each page links to the
next, so you can read straight through a section or jump to the one topic you
came for.

- **[Frontend](/docs/frontend/)** — pick a rendering approach and wire up Vite.
- **[The ORM](/docs/orm/)** — queries, writes and relationships in plain Go
  structs. Start here if your project uses the default SQL layer.
- **[GPA and providers](/docs/database/)** — the provider-backed abstraction
  underneath. Read it if you need GORM, Bun, MongoDB or Redis, or more than one
  store at once.
- **[Migrations](/docs/migrations/)** — a schema builder and a versioned,
  reversible history.
- **[How the framework fits together](/docs/core-concepts/)** — the bootstrap
  pipeline, the service container and the provider system. Worth reading once
  before you add a module, because every module arrives as a provider.

## Reference while you work

- **[Security](/docs/security/)** — authentication, the current user, CSRF,
  sessions, authorization and rate limiting.
- **[Modules](/docs/cache/)** — cache, queue, mail, notifications, realtime,
  file storage, media, search, SEO and payments. Each is a separate Go module
  you add only if you want it.
- **[OAuth2 server](/docs/oauth2/)** — issue tokens to your own clients.
- **[Tooling and shipping](/docs/cli/)** — the CLI, the test harness,
  deployment, and the MCP server that lets an AI agent read your application's
  live state.

## What makes it different

- **Types survive the database.** `Repository[T]` and `db.Model[T]()` return
  `*User` and `[]Post`, not `interface{}` and a type assertion.
- **Everything is a provider.** Modules register themselves through one
  interface and reach the application through a container, so adding mail or
  payments does not mean editing framework code.
- **One CLI for the whole loop.** Scaffolding, code generation, migrations,
  hot reload, a queue dashboard and a production build.
- **Modules are opt-in.** A new project depends on the core and nothing else;
  a part you decline — a cache, a queue, a database — is absent rather than
  configured off.

## Conventions in these docs

Commands are shown as you would type them in a project directory:

```shell
lemmego run migrate
```

Go examples assume the imports a generated project already has. Where an
import is not obvious, it is shown.

{{< callout type="info" >}}
Lemmego is pre-1.0 and under active development. Minor versions may change
public APIs. [Upgrade notes](/docs/appendix/upgrading) lists every change that
needs action on an existing project.
{{< /callout >}}
