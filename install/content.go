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
func Guidelines(agent Agent, modules []string) string {
	var b strings.Builder

	b.WriteString(`## Lemmego

This is a Lemmego application — a full-stack Go framework. An MCP server named
` + "`lemmego`" + ` exposes this application's own state; prefer it over guessing.

### Check the API before you call it

**` + "`module-api`" + ` returns the real exported signatures** of a lemmego module at the
version this project pins, read from the compiled package. Use it before calling
anything you have not just read in this session. Hand-written documentation goes
stale and invents functions that never existed; this cannot.

` + "`search-docs`" + ` searches the documentation site — the same pages a reader sees
at lemmego.github.io, written as tasks rather than as a specification. Use it for
how to do something; use ` + "`module-api`" + ` for exact signatures.

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

	if refs := skillRefs(agent, modules); len(refs) > 0 {
		// Named explicitly rather than left to discovery. Claude Code does
		// find .claude/skills on its own, but the previous version of this
		// installer wrote skills to a path nothing reads and mentioned them
		// nowhere, so three documents were inert in every project that had
		// them. A pointer costs four lines and does not depend on the agent
		// having a skill mechanism at all.
		b.WriteString("\n### Deeper guidance, on demand\n\n")
		b.WriteString("Read the matching file before doing one of these things. ")
		b.WriteString("Each holds what this project has watched go wrong.\n\n")
		b.WriteString("| Read this | Before |\n| --- | --- |\n")
		for _, ref := range refs {
			fmt.Fprintf(&b, "| `%s` | %s |\n", ref.path, ref.when)
		}
	}

	b.WriteString("\n### Project rules\n\n")
	b.WriteString("`" + rulesIndexPath + "` maps file globs to rules this project has " +
		"recorded. Read it before planning or editing, and follow the row whose glob " +
		"matches the file you are changing. It is empty until somebody uses " +
		"`record-rule`, which is the tool to reach for when you are told something " +
		"you would otherwise be told again.\n")

	if len(modules) > 0 {
		b.WriteString("\n### Modules this project depends on\n\n")
		b.WriteString(strings.Join(modules, ", "))
		b.WriteString("\n\nAsk `search-docs` or `module-api` about any of them.\n")
		b.WriteString("\n**Before building any substantial feature, ask `list-modules`.** " +
			"It lists every published module, including the ones this project does not " +
			"require yet. `module-api` runs inside the project, so it reports an unused " +
			"module as though it did not exist — and there are published modules for " +
			"things people routinely rewrite by hand.\n")
	}

	return b.String()
}

// rulesIndexPath is where record-rule keeps its index. Duplicated from the
// server's RulesDir rather than imported, because this package deliberately
// depends on nothing — see the package doc — and a test asserts the two
// agree.
const rulesIndexPath = ".ai/rules/index.md"

// skillRef is one skill as the guideline file advertises it.
type skillRef struct {
	path string
	when string
}

// skillRefs lists the skills this project will have, in a stable order.
func skillRefs(agent Agent, modules []string) []skillRef {
	present := map[string]bool{}
	for _, module := range modules {
		present[module] = true
	}
	var refs []skillRef
	for _, module := range sortedKeys(skillTemplates) {
		skill := skillTemplates[module]
		if !present[module] && !skill.always {
			continue
		}
		refs = append(refs, skillRef{path: agent.skillPath(skill.name), when: skill.when})
	}
	return refs
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
