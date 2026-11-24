package install

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// This file is the skill library: the documents an agent loads on demand,
// rather than the always-on block in content.go.
//
// There is one per published module, minus the families where a single skill
// covers several — the GPA adapters and the connectors — and minus the few an
// agent reads from rather than writes against. TestEveryModuleHasASkillOrAReason
// in the catalog package keeps that true: a new module has to arrive with a
// skill or with the reason it does not need one.
//
// That is a change of policy. The library was three skills for thirty-eight
// modules on the argument that documentation plus module-api was enough for
// the rest, and the argument does not survive contact: the documentation says
// what a module does, and a skill says what somebody gets wrong doing it. An
// agent writing a notification needed to be told to send after the commit, not
// inside it; one writing an image upload needed to be told that stripping EXIF
// without first applying orientation turns every portrait photograph on its
// side. Neither is in a signature, and a skill costs nothing until it loads.
//
// So each one holds either a convention no signature can express — where a
// file goes, how thin a handler should be — or a trap this ecosystem has
// watched somebody fall into. A skill that only restates the README should be
// deleted rather than kept for symmetry.
//
// The bodies are markdown files rather than Go string literals. They were
// literals, and a markdown document inside a backtick-quoted Go string cannot
// contain a fenced code block, so the skills had no code examples — which for
// a document about how to write code is most of the value. They are also now
// reviewable as prose in a diff.

//go:embed skills/*.md
var skillBodies embed.FS

// skillTemplate is one on-demand document.
type skillTemplate struct {
	// name is the directory or file name, the skill's identifier, and the
	// basename of its body under skills/.
	name string
	// description is the front matter line an agent reads to decide whether
	// this skill is relevant to what it is about to do.
	description string
	// when fills the "Before" column of the table in the guideline file, so
	// an agent with no skill mechanism at all still knows when to open it.
	when string
	// always writes the skill even when the keyed module is absent, for the
	// ones that describe how to write for this framework rather than how to
	// use one module of it.
	always bool
}

// body reads the skill's markdown.
//
// A missing file is a programming error rather than a runtime condition — the
// files are compiled in — so this panics, and TestEverySkillHasABody turns it
// into a test failure instead of somebody's install.
func (s skillTemplate) body() string {
	raw, err := skillBodies.ReadFile("skills/" + s.name + ".md")
	if err != nil {
		panic(fmt.Sprintf("install: skill %s has no body: %v", s.name, err))
	}
	return string(raw)
}

// skillTemplates are the on-demand skills, keyed by the module that triggers
// one. A module skill is written only when the project depends on that module,
// so a project without payments carries no payments skill; a skill marked
// `always` describes the framework itself and is written regardless.
var skillTemplates = map[string]skillTemplate{
	// --- Always on: how to write for this framework at all. -----------------
	"*framework": {
		name:        "lemmego-conventions",
		when:        "adding a handler, model, route, repository or service — read this first",
		always:      true,
		description: "Where code goes in a Lemmego project, and how thin each layer stays: resourceful handlers, one model per file, lean route files, database work in repositories.",
	},

	// --- Persistence. -------------------------------------------------------
	"orm": {
		name:        "lemmego-orm",
		when:        "writing a query, a relationship, or an association write",
		description: "Query, write and relate with the Lemmego ORM, including the generic-method shapes that cannot be expressed any other way.",
	},
	"migration": {
		name:        "lemmego-migrations",
		when:        "writing or running a migration, or changing a schema",
		description: "Write and run Lemmego migrations, including the schema builder's sharp edges on Postgres.",
	},

	// --- Identity. ----------------------------------------------------------
	"auth": {
		name:        "lemmego-auth",
		when:        "signing somebody in, reading the current user, or protecting a route",
		description: "Authentication in a Lemmego application: the UserLoader you must write, reading the current user, and which middleware does what.",
	},
	"authz": {
		name:        "lemmego-authz",
		when:        "deciding whether this user may do this to this record",
		description: "Policies and gates with authz: Can versus Authorize, and why an unregistered ability is denied.",
	},
	"rbac": {
		name:        "lemmego-rbac",
		when:        "adding roles or permissions, or wiring rbac into authz",
		description: "Database-backed roles and permissions: opaque scopes, why both authz hooks are needed, and why the scope never comes from an argument.",
	},
	"tenancy": {
		name:        "lemmego-tenancy",
		when:        "anything multi-tenant: resolving a tenant, scoping a query, or row-level security",
		description: "Multi-tenancy without leaking rows: a header selects but does not authorise, the two boot checks, and what the package deliberately does not cover.",
	},
	"oauth2": {
		name:        "lemmego-oauth2",
		when:        "issuing tokens, registering clients, or protecting an API with OAuth2",
		description: "Running an OAuth2 authorization server. This module issues tokens; it is not a client for signing in with Google.",
	},

	// --- Application. -------------------------------------------------------
	"i18n": {
		name:        "lemmego-i18n",
		when:        "adding a translated message, a plural, or a locale",
		description: "Translation with i18n: CLDR plural form names rather than Laravel's pipe syntax, namespaces, and why Go does not format dates here.",
	},
	"queue": {
		name:        "lemmego-queue",
		when:        "dispatching background work or adding a scheduled job",
		description: "Background jobs with queue over tasker, including the provider ordering that makes the difference between working and silently not.",
	},
	"cache": {
		name:        "lemmego-cache",
		when:        "caching a value, taking a lock, or invalidating by tag",
		description: "The cache module: typed access, atomic locks with owner tokens, and tagged invalidation.",
	},

	"gpa": {
		name:        "lemmego-gpa",
		when:        "using the persistence API, or deciding whether to",
		description: "The Go Persistence API and its adapters: registering a provider, typed repositories, and what comes back as unsupported.",
	},

	// --- Application. -------------------------------------------------------
	"mail": {
		name:        "lemmego-mail",
		when:        "sending an email",
		description: "Transactional email: why Queue beats Send on a request, the three interfaces, and testing without a relay.",
	},
	"notify": {
		name:        "lemmego-notify",
		when:        "sending one notification over several channels",
		description: "Multi-channel notifications: sending after the commit rather than inside it, channel name constants, and the contract test.",
	},
	"broadcast": {
		name:        "lemmego-broadcast",
		when:        "pushing events to a connected client, or securing a private channel",
		description: "Realtime over SSE and WebSocket: why the two differ on origin checking, why AllowedOrigins is required, and what is deliberately not built.",
	},
	"media": {
		name:        "lemmego-media",
		when:        "accepting an image upload or generating renditions",
		description: "Image ingest: why orientation must be applied before metadata is stripped, why nothing is upscaled, and cleaning up after a failed insert.",
	},
	"search": {
		name:        "lemmego-search",
		when:        "adding full-text search over existing tables",
		description: "Full-text search: the required Visible predicate, why the query never throws, why only Snippet is HTML, and why results are grouped rather than blended.",
	},
	"seo": {
		name:        "lemmego-seo",
		when:        "adding page metadata, JSON-LD, a sitemap or a slug",
		description: "Meta tags and structured data: why the head is rendered in Go rather than in the page component, and how slugs and streamed sitemaps work.",
	},
	"pay": {
		name:        "lemmego-pay",
		when:        "taking a payment or running a subscription",
		description: "Billing: entitlement from the webhook rather than the redirect, idempotent out-of-order deliveries, and money as an integer.",
	},
	"ratelimit": {
		name:        "lemmego-ratelimit",
		when:        "capping how often a caller can do something",
		description: "Rate limiting: picking a key, why X-Forwarded-For is ignored by default, why it fails open, and the Retry-After header.",
	},
	"fsys": {
		name:        "lemmego-fsys",
		when:        "storing or serving a file",
		description: "File storage over local disk, S3, GCS and memory — and the two things to get right before serving an upload.",
	},
	"modules": {
		name:        "lemmego-modules",
		when:        "deciding which features a tenant has",
		description: "The per-tenant feature registry: the boot-time dependency graph, the route gate, and the three-package dependency split.",
	},

	// --- Frontend. ----------------------------------------------------------
	"templ": {
		name:        "lemmego-templ",
		when:        "rendering a templ component",
		description: "Rendering templ from a handler, and why a .templ edit does nothing until the generator runs.",
	},

	// --- Operations. --------------------------------------------------------
	"log": {
		name:        "lemmego-log",
		when:        "configuring logging or wondering where a log line went",
		description: "Logging: there is no log.Info, the five channels, rotation defaults, and the one owner of slog.SetDefault.",
	},
	"observe": {
		name:        "lemmego-observe",
		when:        "debugging what one request did, or finding an empty observe dashboard",
		description: "Request-scoped debugging with observe, including the driver wrapper without which the dashboard shows requests with nothing underneath them.",
	},
	"apptest": {
		name:        "lemmego-testing",
		when:        "writing a test, or trusting one that passed",
		description: "Write tests for a Lemmego application with apptest, including the ways a suite can silently pass.",
	},
	"inertia": {
		name:        "lemmego-inertia",
		when:        "touching an Inertia page, a layout, or server-side rendering",
		description: "Work with Inertia pages and server-side rendering in a Lemmego application.",
	},
}

// writeSkills writes one skill file per relevant module, where this agent
// reads them.
func writeSkills(dir string, agent Agent, modules []string) ([]string, error) {
	present := map[string]bool{}
	for _, module := range modules {
		present[module] = true
	}

	var written []string
	for _, key := range sortedKeys(skillTemplates) {
		skill := skillTemplates[key]
		if !present[key] && !skill.always {
			continue
		}
		rel := agent.skillPath(skill.name)
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(skillBody(agent, skill)), 0o644); err != nil {
			return nil, err
		}
		written = append(written, rel)
	}
	sort.Strings(written)
	return written, nil
}

// skillBody renders a skill in the front matter its agent expects.
//
// Cursor's rules take a description and an alwaysApply flag and no name — the
// name is the filename. Claude Code's skills take a name and a description.
// Getting this wrong does not error; it produces a file the agent ignores,
// which is the failure this whole area had.
func skillBody(agent Agent, skill skillTemplate) string {
	if agent == Cursor {
		return fmt.Sprintf("---\ndescription: %s\nalwaysApply: false\n---\n\n# %s\n\n%s",
			skill.description, skill.name, skill.body())
	}
	return fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n# %s\n\n%s",
		skill.name, skill.description, skill.name, skill.body())
}

// SkillNames lists every skill this package can write, for a test that checks
// nothing in skills/ is orphaned.
func SkillNames() []string {
	names := make([]string, 0, len(skillTemplates))
	for _, key := range sortedKeys(skillTemplates) {
		names = append(names, skillTemplates[key].name)
	}
	return names
}

// SkillModules lists the module keys that trigger a skill, so a test can drive
// a project's go.mod from the real list rather than a hand-kept copy of it.
//
// The empty-string-free key "*framework" marks a skill that is written
// regardless; it is not a module and is excluded here.
func SkillModules() []string {
	modules := make([]string, 0, len(skillTemplates))
	for _, key := range sortedKeys(skillTemplates) {
		if skillTemplates[key].always {
			continue
		}
		modules = append(modules, key)
	}
	return modules
}
