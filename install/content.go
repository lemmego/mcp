package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProjectModules reports which lemmego modules a project depends on, read from
// its go.mod.
//
// go.mod rather than the build, because this package must work without
// compiling the project — which is the whole reason it is separate from the
// server. The server's app-info tool reports build-resolved versions instead,
// and the two can legitimately disagree: go.mod says what was asked for.
func ProjectModules(dir string) ([]string, error) {
	body, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		// Not an error worth failing on: an installation in a directory
		// without a go.mod still wants the MCP config written. The guidelines
		// simply cannot be tailored.
		return nil, nil
	}

	seen := map[string]bool{}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		path := fields[0]
		if !strings.HasPrefix(path, "github.com/lemmego/") {
			continue
		}
		seen[strings.TrimPrefix(path, "github.com/lemmego/")] = true
	}

	modules := make([]string, 0, len(seen))
	for name := range seen {
		modules = append(modules, name)
	}
	sort.Strings(modules)
	return modules, nil
}

// Guidelines is the always-on block written into CLAUDE.md or AGENTS.md.
//
// Short on purpose. Everything loaded upfront costs context on every single
// request, so this holds only what an agent needs *before* it knows what it is
// doing: that the tools exist, which ones to reach for first, and the handful
// of framework facts that are counter-intuitive enough to cause real damage.
// Depth belongs in skills, which load on demand.
func Guidelines(modules []string) string {
	var b strings.Builder

	b.WriteString(`## Lemmego

This is a Lemmego application — a full-stack Go framework. An MCP server named
` + "`lemmego`" + ` exposes this application's own state; prefer it over guessing.

### Check the API before you call it

**` + "`module-api`" + ` returns the real exported signatures** of a lemmego module at the
version this project pins, read from the compiled package. Use it before calling
anything you have not just read in this session. Hand-written documentation goes
stale and invents functions that never existed; this cannot.

` + "`search-docs`" + ` returns the modules' own READMEs, which are the authoritative
specification for each module — ahead of any other documentation, including the
docs site.

### The other tools

| Tool | Use it for |
| --- | --- |
| ` + "`list-routes`" + ` | what this application serves, before adding a route |
| ` + "`database-schema`" + ` | the live schema, which is not always what the migrations say |
| ` + "`database-query`" + ` | read-only unless the server was started with --allow-writes |
| ` + "`migration-status`" + ` | what is applied, before writing a new migration |
| ` + "`config`" + ` | resolved configuration, with secrets withheld |
| ` + "`health`" + ` | the registered checks and what is degraded |
| ` + "`last-error`" + `, ` + "`read-log-entries`" + ` | start debugging here |
| ` + "`record-rule`" + ` | when told something you would otherwise be told again |

### Framework facts worth knowing upfront

- **Providers run in ` + "`LoadProviders`" + ` order, and the order is load-bearing.** A
  provider that resolves something another registers must come after it. The
  database connector publishes ` + "`db.Connection`" + `, so anything borrowing the pool —
  the queue, database-backed sessions — is listed after it.
- **A clean build and a 200 response prove very little here.** Inertia falls
  back to client rendering when server-side rendering fails, so a broken page
  answers 200 and looks correct to a human while being empty to a crawler.
  Check for ` + "`data-server-rendered=\"true\"`" + ` in the body, not the status code.
- **` + "`apptest.WithScratchDB()`" + ` takes its server from the application's own
  configuration**, which under a test binary means ` + "`.env.test`" + `. There is no
  environment variable naming a database, and it fails rather than skipping
  when the server is unreachable — because a suite that skips and one that
  passes are the same exit code.
- **` + "`config.MustEnv`" + ` treats a variable that is present and empty as a decision**,
  and panics on a value it cannot parse. A stray ` + "`FOO=`" + ` line is not the same as an
  absent one.
- **Write migrations with raw ` + "`tx.Exec`" + ` when the schema matters.** The schema
  builder has sharp edges on Postgres: multi-column index names carry no table
  prefix and collide, ` + "`Enum`" + ` emits no constraint at all, and ` + "`ForeignID`" + ` adds its
  own sequence.

### Verifying your own work

This project's standard is that a test is not trusted until it has been seen to
fail. Revert the code a new test protects, **confirm the revert actually
applied**, watch the test fail, then restore. A no-op patch reads exactly like a
vacuous test.
`)

	if len(modules) > 0 {
		b.WriteString("\n### Modules this project depends on\n\n")
		b.WriteString(strings.Join(modules, ", "))
		b.WriteString("\n\nAsk `search-docs` or `module-api` about any of them.\n")
	}

	return b.String()
}

// skillTemplates are the on-demand skills, keyed by the module that triggers
// them. A skill is written only when the project depends on its module, so a
// project that does not use payments carries no payment skill.
//
// Deliberately not one skill per module: a skill earns its place by holding
// something an agent gets wrong, and for most modules the README plus
// module-api is enough. These are the ones where this project has watched the
// mistake happen.
var skillTemplates = map[string]struct {
	name        string
	description string
	body        string
}{
	"migration": {
		name:        "lemmego-migrations",
		description: "Write and run Lemmego migrations, including the schema builder's sharp edges on Postgres.",
		body: `## When to use this skill

When adding, editing or rolling back a migration in a Lemmego project.

## Check first

Run ` + "`migration-status`" + ` before writing one. A version that collides with an
applied migration will not run, and the collision is silent.

## The schema builder's sharp edges on Postgres

Each of these was found by shipping it, so prefer raw ` + "`tx.Exec`" + ` whenever the
exact DDL matters:

- **Multi-column index names carry no table prefix**, and Postgres index names
  are schema-global — so two tables both wanting ` + "`(user_id, position)`" + ` collide
  and the second fails the batch. Write multi-column indexes as raw SQL.
- **` + "`Enum`" + ` emits no CHECK constraint at all** on Postgres, silently. Use
  ` + "`String`" + ` plus a raw CHECK.
- **` + "`ForeignID(col)`" + ` adds a BIGSERIAL**, giving the column its own sequence and
  an implicit default. Use ` + "`UnsignedBigInt`" + ` plus ` + "`Foreign(col).References(...)`" + `.
- **` + "`TimestampTz(name, p)`" + ` with p > 0 emits invalid SQL** — always pass 0.
- **A database DEFAULT never applies through the ORM**, which names every mapped
  column in its INSERT, so a ` + "`DEFAULT true`" + ` bool is written as Go's zero value.
  Set it in ` + "`BeforeCreate`" + ` as well.

## The whole batch is one transaction

With a hardcoded 30s timeout, so ` + "`CREATE INDEX CONCURRENTLY`" + ` is impossible and
the batch is all-or-nothing. Postgres DDL is transactional, so functions and
triggers are fine.

## Never migrate through a connection pooler

` + "`migrate`" + ` resolves ` + "`--dsn`" + `, then ` + "`DATABASE_URL_UNPOOLED`" + `, then ` + "`DATABASE_URL`" + `, then the
` + "`DB_*`" + ` parts. Managed providers publish a pooled URL and point you at it; under
transaction pooling lib/pq's unnamed prepared statements fail intermittently as
` + "`bind message supplies N parameters, but prepared statement \"\" requires M`" + `.

## Verify reversibility

Up, down, and up again on a throwaway database. That catches a down migration
that leaves a table behind and an up that only works on an empty schema.
`,
	},
	"apptest": {
		name:        "lemmego-testing",
		description: "Write tests for a Lemmego application with apptest, including the ways a suite can silently pass.",
		body: `## When to use this skill

When writing or debugging tests in a Lemmego project.

## The failure this project keeps hitting

**A suite that skips is indistinguishable from one that passes in the exit
code.** ` + "`apptest.WithScratchDB()`" + ` fails rather than skipping for exactly that
reason, so an unreachable database is now loud — but the lesson generalises to
every ` + "`t.Skip`" + ` in a suite.

The giveaway is the runtime. If a package that boots the application finishes in
a second, it did nothing. Check with ` + "`go test -v`" + ` and look for ` + "`--- SKIP`" + `.

The server comes from ` + "`.env.test`" + `, not from an environment variable. Two ways
to name a database is two things that can disagree, and they did: the old
` + "`APPTEST_POSTGRES_DSN`" + ` supplied host, user, password and port but not options,
so a managed URL's ` + "`sslmode=require`" + ` survived into a local server and every
test panicked.

## Verify the probe, not just the test

A new test is not trusted until it has been seen to fail:

1. Revert the code the test protects.
2. **Confirm the revert actually applied** — grep for it. A no-op patch reads
   exactly like a vacuous test.
3. Watch the test fail.
4. Restore, and re-run.

Real examples from this project: a test passed by asserting two identical
redirects were identical; another exercised a helper that predated the feature;
a third went through a code path that forced the value it was asserting on.

## Asserting on rendered output

Read the Inertia props rather than grepping the HTML. A page echoes its own
query back, so a substring check against the body is true for any query whose
text appears in a result. React also inserts ` + "`<!-- -->`" + ` between static text and an
interpolated value, so ` + "`Continue with GitHub`" + ` is emitted as
` + "`Continue with <!-- -->GitHub`" + ` and a naive match fails.

## What the harness gives you

A real boot with every provider, a request builder, Inertia assertions,
` + "`ActingAs`" + `, a Postgres schema per test, and ` + "`Server()`" + `/` + "`Client()`" + ` when a test needs a
real connection — ` + "`httptest.NewRecorder`" + ` cannot be hijacked, so a WebSocket
upgrade fails and an SSE handler blocks forever without them.
`,
	},
	"inertia": {
		name:        "lemmego-inertia",
		description: "Work with Inertia pages and server-side rendering in a Lemmego application.",
		body: `## When to use this skill

When adding or changing an Inertia page, or debugging a blank one.

## A 200 proves nothing

Inertia falls back to client rendering when the server-side render fails. The
page answers **200 and looks perfect to a human** while a crawler receives an
empty ` + "`<div id=\"app\">\"`" + `. Check the body for ` + "`data-server-rendered=\"true\"`" + ` and for
real markup inside the container — not the status code.

Outside production the SSR guard turns that into a 500 with the reason, which is
why development is strict and production is not.

## Never touch a browser global during render

The renderer is Node: ` + "`window`" + `, ` + "`document`" + ` and ` + "`localStorage`" + ` all throw. A page that
read ` + "`window.location.search`" + ` while rendering took down every request to it. Pass
what you need as a prop, or read it in ` + "`useEffect`" + `, which never runs in the
renderer.

## The title is rendered in Go

` + "`lemmego/seo`" + ` writes the real ` + "`<title>`" + ` server-side, so Inertia's ` + "`<Head title>`" + `
emits a **second** one. Update the tab title in an effect instead.

## Props and auth ordering

Shared props are resolved before the route's own middleware, so an eagerly
computed prop sees a signed-out viewer even for a signed-in person. Pass a
` + "`func() any`" + ` closure so it resolves after auth.
`,
	},
}

// writeSkills writes one SKILL.md per relevant module.
func writeSkills(dir string, modules []string) ([]string, error) {
	present := map[string]bool{}
	for _, module := range modules {
		present[module] = true
	}

	var written []string
	for module, skill := range skillTemplates {
		if !present[module] {
			continue
		}
		rel := filepath.Join(".ai", "skills", skill.name, "SKILL.md")
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		body := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n# %s\n\n%s",
			skill.name, skill.description, skill.name, skill.body)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return nil, err
		}
		written = append(written, rel)
	}
	sort.Strings(written)
	return written, nil
}

// ServerWiring reports whether a project can actually serve MCP.
//
// It exists because configuring an editor and having a server are two
// different things, and only one of them is this package's doing. A project
// scaffolded by `lemmego new` has the provider registered; an existing project
// does not, and nothing in Go discovers it — there is no equivalent of
// Composer's package auto-discovery, which is what lets Laravel's installer
// assume the server is there.
//
// Without this check the installer wrote a perfectly good .mcp.json pointing
// at a command that does not exist, and the editor showed a red cross with no
// reason given.
type ServerWiring struct {
	// ModuleRequired reports whether go.mod requires github.com/lemmego/mcp.
	ModuleRequired bool
	// ProviderRegistered reports whether the provider appears in
	// bootstrap/providers.go.
	ProviderRegistered bool
}

// Complete reports whether the project will answer `lemmego run mcp:serve`.
func (w ServerWiring) Complete() bool {
	return w.ModuleRequired && w.ProviderRegistered
}

// DetectServerWiring inspects a project for the two things the server needs.
//
// Text matching rather than parsing: go.mod is read as text anyway for
// ProjectModules, and providers.go is a file the project owns and may have
// reshaped. A false positive here costs a redundant instruction; parsing it
// properly would cost a dependency on go/ast in a package whose whole point is
// that it has none.
func DetectServerWiring(dir string) ServerWiring {
	var wiring ServerWiring

	if body, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
		wiring.ModuleRequired = strings.Contains(string(body), serverModulePath)
	}

	// The conventional location first, then a short search, because a project
	// may have moved it.
	candidates := []string{
		filepath.Join(dir, "bootstrap", "providers.go"),
		filepath.Join(dir, "bootstrap", "provider.go"),
	}
	for _, path := range candidates {
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if strings.Contains(string(body), serverModulePath) {
			wiring.ProviderRegistered = true
			break
		}
	}
	return wiring
}

const serverModulePath = "github.com/lemmego/mcp"

// WiringInstructions is what to tell somebody whose project cannot serve yet.
//
// Written out in full rather than pointing at documentation, because this is
// read at the moment the thing has just failed, and a link is one more step
// between the person and a working editor.
func WiringInstructions(wiring ServerWiring) string {
	if wiring.Complete() {
		return ""
	}

	var b strings.Builder
	b.WriteString("This project cannot serve MCP yet, so your editor will show the server as failed.\n")
	b.WriteString("The configuration above is correct; the server itself is missing.\n\n")

	step := 1
	if !wiring.ModuleRequired {
		fmt.Fprintf(&b, "%d. Add the module:\n\n     go get %s\n\n", step, serverModulePath)
		step++
	}
	if !wiring.ProviderRegistered {
		fmt.Fprintf(&b, "%d. Register the provider in bootstrap/providers.go, last in the list,\n", step)
		b.WriteString("   so it reports on everything the providers above it register:\n\n")
		b.WriteString("     import lemmegomcp \"github.com/lemmego/mcp\"\n\n")
		b.WriteString("     func LoadProviders() []app.Provider {\n")
		b.WriteString("         providers := []app.Provider{ /* ...existing... */ }\n\n")
		b.WriteString("         // A development tool: it serves this application's configuration\n")
		b.WriteString("         // and database to whatever starts it.\n")
		b.WriteString("         if config.MustEnv(\"APP_ENV\", \"development\") != \"production\" {\n")
		b.WriteString("             providers = append(providers, &lemmegomcp.Provider{})\n")
		b.WriteString("         }\n")
		b.WriteString("         return providers\n")
		b.WriteString("     }\n\n")
		step++
	}
	fmt.Fprintf(&b, "%d. Check it: lemmego run mcp:serve --help\n", step)
	b.WriteString("   Then restart your editor.\n")
	return b.String()
}
