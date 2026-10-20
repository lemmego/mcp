package mcp

import (
	"fmt"
	"os"

	"github.com/lemmego/api/app"
	"github.com/spf13/cobra"
)

// Provider registers the `mcp:serve` command on an application.
//
// A command rather than a route, for the reason stdio is the only transport:
// the server talks to one editor over a pipe, and an HTTP endpoint would be an
// unauthenticated connection to the database for anything that could reach it.
//
// List it last in LoadProviders. It needs the database connection, the session
// store and the health checks to already be registered — not to serve them,
// but because a tool that resolves them at call time reports whatever was
// registered by the time Provide ran.
type Provider struct {
	// Options are passed to New when the command runs.
	Options []Option
}

func (p *Provider) Provide(a app.App) error { return nil }

func (p *Provider) AddCommands() []app.Command {
	return []app.Command{
		func(a app.App) *cobra.Command {
			var allowWrites bool
			var moduleRoots []string
			var maxRows int

			cmd := &cobra.Command{
				Use:   "mcp:serve",
				Short: "Serve this application's state to an AI agent over MCP (stdio)",
				Long: "Serves a Model Context Protocol server on stdin/stdout, exposing this " +
					"application's routes, schema, configuration, migrations, logs and health " +
					"to an AI coding agent.\n\n" +
					"Not meant to be run by hand: an editor starts it. `lemmego mcp:install` " +
					"writes the configuration that makes that happen.",
				RunE: func(cmd *cobra.Command, args []string) error {
					options := append([]Option{}, p.Options...)
					if allowWrites {
						options = append(options, WithWrites())
					}
					if len(moduleRoots) > 0 {
						options = append(options, WithModuleRoots(moduleRoots...))
					}
					if maxRows > 0 {
						options = append(options, WithMaxRows(maxRows))
					}

					// Nothing may be written to stdout but protocol frames —
					// one stray Println corrupts the stream and the editor
					// reports a server that will not start. Said here because
					// it is the first thing somebody adding a tool gets wrong.
					if allowWrites {
						fmt.Fprintln(os.Stderr, "lemmego mcp: writes are ENABLED; the agent can modify this database")
					}

					return New(a, options...).Serve(cmd.Context())
				},
			}

			cmd.Flags().BoolVar(&allowWrites, "allow-writes", false,
				"Let the agent run statements that modify data (off by default)")
			cmd.Flags().StringSliceVar(&moduleRoots, "module-root", nil,
				"Directory to search for module READMEs before the module cache; repeatable")
			cmd.Flags().IntVar(&maxRows, "max-rows", 0,
				"Maximum rows a query may return (default 100)")
			return cmd
		},
	}
}
