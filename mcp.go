// Package mcp serves a Lemmego application's own state to an AI coding agent
// over the Model Context Protocol.
//
// The problem it solves is narrow and specific: an agent editing a Lemmego
// project cannot see the routing table, the schema, the resolved
// configuration, the migration state or the logs, so it guesses — and the
// expensive guesses are the ones about APIs, where it writes a call against a
// function that does not exist and the mistake survives until somebody
// compiles it.
//
// So the two tools to reach for first are not the database ones. `module-api`
// returns the real exported signatures of a lemmego module at the version this
// project pins, and `search-docs` returns the module's own README. Everything
// else here is context; those two are correction.
//
// # Safety
//
// This is a local development tool and it is shaped like one.
//
//   - **stdio only.** There is no HTTP transport, because an MCP server on a
//     port is an unauthenticated connection to the database for anything that
//     can reach it.
//   - **Read-only by default.** `database-query` refuses anything but a single
//     read statement unless the server was built WithWrites.
//   - **Secrets are withheld.** Configuration goes through RedactMap, which
//     denies by key pattern and also rewrites a credential found inside a
//     value — DATABASE_URL being the case a key-based rule alone gets wrong.
package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/lemmego/api/app"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is reported to the client in the server handshake.
const Version = "0.1.0"

// Server serves one application's state.
type Server struct {
	app app.App

	// allowWrites opens database-query to statements that are not reads. Off
	// by default: an agent that can run UPDATE against a development database
	// can also run it against whatever DATABASE_URL happens to point at, and
	// that has been production more than once.
	allowWrites bool

	// moduleRoots are directories searched for a module's own documentation
	// before falling back to the module cache, so a monorepo checkout reads
	// the working tree rather than the last published copy.
	moduleRoots []string

	// routesOnce guards the one-time route materialisation. Per server rather
	// than package-level, so two servers in one test do not share it.
	routesOnce sync.Once

	// maxRows bounds what a query may return. An agent asking for a whole
	// table should get a bounded answer rather than filling its context with
	// one tool result.
	maxRows int

	// extra are tool registrars contributed by other modules.
	//
	// The dependency has to point this way. This server is the generic
	// reporter and a module like observe is one subsystem, so importing it
	// here to call its registrar would invert that — and would put every
	// future subsystem's dependency tree into everybody's mcp build.
	extra []Registrar
}

// Registrar adds tools to the server.
//
// A module that wants to answer questions over MCP exposes one of these and
// an application passes it to New, so registerTools below stays a readable
// catalogue of what this package itself provides.
type Registrar func(*sdk.Server)

// Option configures a Server.
type Option func(*Server)

// WithWrites allows database-query to execute statements that are not reads.
//
// Separate from construction and named for what it grants, so that enabling it
// is a visible decision in the application's own source rather than a flag
// somebody set once.
func WithWrites() Option {
	return func(s *Server) { s.allowWrites = true }
}

// WithModuleRoots adds directories to search for module documentation, ahead
// of the module cache. Point this at a monorepo checkout to read unreleased
// code.
func WithModuleRoots(paths ...string) Option {
	return func(s *Server) { s.moduleRoots = append(s.moduleRoots, paths...) }
}

// WithTools contributes tools from another module.
//
//	mcp.New(a, mcp.WithTools(observemcp.Tools(recorder)))
//
// Each registrar is called once, after this package's own tools, so a
// contributed tool cannot shadow one a reader would expect to find.
func WithTools(registrars ...Registrar) Option {
	return func(s *Server) { s.extra = append(s.extra, registrars...) }
}

// WithMaxRows bounds the rows a query may return. Zero or less restores the
// default.
func WithMaxRows(n int) Option {
	return func(s *Server) {
		if n > 0 {
			s.maxRows = n
		}
	}
}

const defaultMaxRows = 100

// New builds a server over a booted application.
//
// It takes app.App rather than opening its own connections because every
// answer here has to be the one the application would give: a tool that
// re-derived the DSN from the environment would report on a different database
// than the one the app is using, which is the failure `queue` already had
// before it borrowed db.Connection.
func New(a app.App, opts ...Option) *Server {
	s := &Server{app: a, maxRows: defaultMaxRows}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Serve registers the tools and serves them over stdio until the context is
// cancelled or the client disconnects.
func (s *Server) Serve(ctx context.Context) error {
	server := sdk.NewServer(&sdk.Implementation{
		Name:    "lemmego",
		Version: Version,
	}, nil)

	s.registerTools(server)

	return server.Run(ctx, &sdk.StdioTransport{})
}

// registerTools is the whole catalogue, in one place, so what this server
// exposes can be read without opening every file.
func (s *Server) registerTools(server *sdk.Server) {
	// Correction first — see the package doc.
	s.addModuleAPI(server)
	s.addSearchDocs(server)

	// What this application is.
	s.addAppInfo(server)
	s.addConfig(server)
	s.addListRoutes(server)
	s.addAbsoluteURL(server)

	// What its database looks like.
	s.addDatabaseConnections(server)
	s.addDatabaseSchema(server)
	s.addDatabaseQuery(server)
	s.addMigrationStatus(server)

	// What it is doing now.
	s.addHealth(server)
	s.addReadLogEntries(server)
	s.addLastError(server)

	// What it has been told.
	s.addRecordRule(server)

	// And whatever another module contributes, last — so a contributed tool
	// cannot shadow one above it.
	for _, registrar := range s.extra {
		if registrar != nil {
			registrar(server)
		}
	}
}

// text is the shape every tool here returns: one block of text for the agent
// to read, plus the structured value the SDK publishes alongside it.
//
// Structured output is what a client should prefer, but the text block is what
// makes a result legible when a human reads the transcript — and the whole
// point of this server is that somebody can tell what the agent was told.
func text(body string) *sdk.CallToolResult {
	return &sdk.CallToolResult{
		Content: []sdk.Content{&sdk.TextContent{Text: body}},
	}
}

// failure reports a tool-level error to the agent rather than a protocol
// error, so the agent can read the reason and try something else instead of
// seeing the call fail.
func failure(format string, args ...any) *sdk.CallToolResult {
	return &sdk.CallToolResult{
		IsError: true,
		Content: []sdk.Content{&sdk.TextContent{Text: fmt.Sprintf(format, args...)}},
	}
}
