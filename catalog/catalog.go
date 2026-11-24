// Package catalog is the list of modules the Lemmego ecosystem publishes.
//
// It exists because every other way an agent could learn what exists is gated
// on the project already depending on it. `module-api` shells out to `go doc`
// inside the project, so it answers "no required module provides package
// github.com/lemmego/tenancy" — which reads as "there is no such thing"
// rather than as "you have not added it". `search-docs` only covers what has
// a documentation page. And the module list in the guideline file is read
// from go.mod, so it can only ever name what is already there.
//
// The result was that a project could not discover tenancy, RBAC or the
// OAuth2 server at all: an agent asked to add multi-tenancy would write it
// from scratch beside a published module that already did it.
//
// Standard library only, and no dependency on the server, so the installer
// can use it too.
package catalog

import (
	"sort"
	"strings"
)

// Category groups modules the way somebody looking for one would group them.
type Category string

const (
	Core        Category = "core"
	Persistence Category = "persistence"
	Identity    Category = "identity"
	Application Category = "application"
	Frontend    Category = "frontend"
	Operations  Category = "operations"
)

// Module is one published module.
type Module struct {
	// Name is the last element of the import path, and what `module-api` and
	// `search-docs` accept as a short name.
	Name string

	// Summary is one line, taken from the module's own README rather than
	// written fresh, because the README is the authoritative spec and a
	// second description is a second thing to keep true.
	Summary string

	Category Category

	// Docs is the documentation site path, for search-docs' section
	// argument. Empty where the module has no page yet, which is itself
	// worth reporting: an agent should know it must fall back to the README.
	Docs string

	// Scaffold reports whether `lemmego new` can add this module to a project
	// at create time, wiring its provider, .env block, migration and worked
	// example. A module without it has to be wired by hand.
	Scaffold bool

	// Provider reports whether the module contributes an app.Provider that
	// must be listed in LoadProviders. Getting this wrong is the single most
	// common way a correctly installed module does nothing.
	Provider bool
}

// Path is the module's import path.
func (m Module) Path() string { return "github.com/lemmego/" + m.Name }

// All is every published module, in no particular order; Sorted and ByName
// are the accessors to use.
//
// Keep this in step with the repositories. TestCatalogueCoversTheWorkspace
// compares it against go.work when the monorepo is present, because a module
// published and not listed here is a module an agent cannot find, which is
// the exact problem this package exists to solve.
var All = []Module{
	// Core.
	{Name: "api", Category: Core, Docs: "core-concepts", Provider: false,
		Summary: "The framework core: routing, context, middleware, DI container, sessions, config, validation."},
	{Name: "cli", Category: Core, Docs: "cli",
		Summary: "The `lemmego` binary: generates a new project, generates code inside one, and runs it."},

	// Persistence.
	{Name: "orm", Category: Persistence, Docs: "orm",
		Summary: "An independent SQL persistence module built directly on database/sql, with an Eloquent-shaped API."},
	{Name: "migration", Category: Persistence, Docs: "migrations",
		Summary: "Database schema migrations with a fluent schema builder, up and down."},
	{Name: "gpa", Category: Persistence, Docs: "database",
		Summary: "A unified, type-safe persistence API over SQL and NoSQL through provider implementations."},
	{Name: "gpaorm", Category: Persistence, Docs: "orm/gpa-adapter",
		Summary: "The GPA provider backed by the Lemmego ORM."},
	{Name: "gpagorm", Category: Persistence, Docs: "database/providers/gorm",
		Summary: "The GPA provider backed by GORM."},
	{Name: "gpabun", Category: Persistence, Docs: "database/providers/bun",
		Summary: "The GPA provider backed by Bun."},
	{Name: "gpamongo", Category: Persistence, Docs: "database/providers/mongodb",
		Summary: "The GPA provider backed by MongoDB."},
	{Name: "gparedis", Category: Persistence, Docs: "database/providers/redis",
		Summary: "The GPA provider backed by Redis."},
	{Name: "ormconnector", Category: Persistence, Docs: "database/connection-seam", Provider: true,
		Summary: "Wires the Lemmego ORM into the application lifecycle, and publishes db.Connection."},
	{Name: "gormconnector", Category: Persistence, Docs: "database/gormconnector", Provider: true,
		Summary: "Wires GORM into the application lifecycle: opens the connection and publishes db.Connection."},
	{Name: "bunconnector", Category: Persistence, Docs: "database/bunconnector", Provider: true,
		Summary: "Wires Bun into the application lifecycle, with optional GPA integration."},

	// Identity.
	{Name: "auth", Category: Identity, Docs: "security/authentication", Provider: true,
		Summary: "Authentication: who is making this request. Sessions, credentials, JWT, LoginAs, and social sign-in."},
	{Name: "authz", Category: Identity, Docs: "authz", Scaffold: true, Provider: true,
		Summary: "Authorization: typed policies, ownership checks and a route gate. Authentication is auth's job, not this one's."},
	{Name: "rbac", Category: Identity, Docs: "rbac", Scaffold: true, Provider: true,
		Summary: "Database-backed roles and permissions, behind a cache. Pairs with authz, which decides per record."},
	{Name: "oauth2", Category: Identity, Docs: "oauth2", Scaffold: true, Provider: true,
		Summary: "An OAuth 2.0 authorization *server*, Passport-shaped: your app issues the tokens. Not a client for signing in elsewhere."},
	{Name: "tenancy", Category: Identity, Docs: "tenancy", Scaffold: true, Provider: true,
		Summary: "Multi-tenancy: which tenant a request is about, and keeping one tenant's rows away from another's."},
	{Name: "modules", Category: Identity,
		Summary: "A registry of the features a tenant has or has not, with a dependency graph and per-tenant enablement."},

	// Application.
	{Name: "mail", Category: Application, Docs: "mail", Scaffold: true, Provider: true,
		Summary: "Transactional email: a mailable describes the message, SMTP sends it, and the queue can carry it."},
	{Name: "notify", Category: Application, Docs: "notifications", Scaffold: true, Provider: true,
		Summary: "One notification delivered over several channels: database, mail and broadcast."},
	{Name: "broadcast", Category: Application, Docs: "broadcast", Scaffold: true, Provider: true,
		Summary: "Events delivered over a live connection: SSE and WebSocket behind one contract, in-process or over Redis."},
	{Name: "media", Category: Application, Docs: "media", Scaffold: true, Provider: true,
		Summary: "Turns an uploaded image into the renditions a page serves: orientation, resize, re-encode, metadata stripped. Pure Go."},
	{Name: "search", Category: Application, Docs: "search", Scaffold: true, Provider: true,
		Summary: "Full-text search with a pluggable engine over Postgres tsvector columns an application already has."},
	{Name: "seo", Category: Application, Docs: "seo", Scaffold: true, Provider: true,
		Summary: "Meta tags, JSON-LD, sitemaps, robots.txt and slugs."},
	{Name: "pay", Category: Application, Docs: "payments", Scaffold: true, Provider: true,
		Summary: "Subscription billing: one hosted checkout and one signed webhook, over Stripe."},
	{Name: "ratelimit", Category: Application, Docs: "ratelimit", Scaffold: true, Provider: true,
		Summary: "A fixed-window rate limiter as middleware and as Allow(), over the application's cache."},
	{Name: "i18n", Category: Application, Docs: "localization", Scaffold: true, Provider: true,
		Summary: "Translation and localization: a JSON catalogue, CLDR plurals, Accept-Language negotiation, shared with the browser."},
	{Name: "cache", Category: Application, Docs: "cache", Provider: true,
		Summary: "Memory, file, Redis and null stores behind one API, with typed access, atomic locks, tags and events."},
	{Name: "queue", Category: Application, Docs: "queue", Provider: true,
		Summary: "Background jobs as a Lemmego provider, wrapping tasker, with a dashboard and a cron scheduler."},
	{Name: "tasker", Category: Application, Docs: "queue",
		Summary: "The distributed, multi-node background job engine underneath queue. Use queue in an application."},
	{Name: "fsys", Category: Application, Docs: "filesystem", Provider: true,
		Summary: "One file-storage interface over local disk, S3, Google Cloud Storage and memory."},

	// Frontend.
	{Name: "inertia", Category: Frontend, Docs: "frontend/inertia", Provider: true,
		Summary: "Inertia.js: server-side routing with a React or Vue front end, including server-side rendering."},
	{Name: "templ", Category: Frontend, Docs: "frontend/templ",
		Summary: "Renders templ components from a Lemmego handler."},

	// Operations.
	{Name: "log", Category: Operations, Docs: "logging", Provider: true,
		Summary: "Configures slog: a readable console, a rotating file, or both. The one owner of slog.SetDefault."},
	{Name: "observe", Category: Operations, Docs: "observe", Scaffold: true, Provider: true,
		Summary: "Records what one request did — queries, logs, cache, jobs, mail — with timings, behind a dashboard and MCP tools."},
	{Name: "apptest", Category: Operations, Docs: "testing",
		Summary: "Boots a real application in a test and drives it over HTTP, with Inertia assertions and a scratch database."},
	{Name: "mcp", Category: Operations, Docs: "mcp", Provider: true,
		Summary: "Serves this application's own state to an AI coding agent, and installs the guidelines and skills."},
}

// Sorted returns the catalogue ordered by category and then name, which is
// the order a reader wants rather than the order the list is written in.
func Sorted() []Module {
	order := map[Category]int{Core: 0, Persistence: 1, Identity: 2, Application: 3, Frontend: 4, Operations: 5}
	out := make([]Module, len(All))
	copy(out, All)
	sort.SliceStable(out, func(i, j int) bool {
		if a, b := order[out[i].Category], order[out[j].Category]; a != b {
			return a < b
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// ByName finds one module.
func ByName(name string) (Module, bool) {
	name = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(name)), "github.com/lemmego/")
	for _, module := range All {
		if module.Name == name {
			return module, true
		}
	}
	return Module{}, false
}

// Match returns the modules whose name or summary mentions every term in the
// query, so "multi tenant" narrows rather than widens.
func Match(query string) []Module {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		return Sorted()
	}
	var out []Module
	for _, module := range Sorted() {
		haystack := strings.ToLower(module.Name + " " + module.Summary + " " + string(module.Category))
		matched := 0
		for _, term := range terms {
			if strings.Contains(haystack, term) {
				matched++
			}
		}
		if matched == len(terms) {
			out = append(out, module)
		}
	}
	return out
}
