package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/lemmego/mcp/catalog"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type listModulesInput struct {
	Query    string `json:"query,omitempty" jsonschema:"what you are trying to do, e.g. 'multi-tenancy', 'roles and permissions', 'send email'"`
	Category string `json:"category,omitempty" jsonschema:"restrict to one of: core, persistence, identity, application, frontend, operations"`
	InUse    bool   `json:"in_use,omitempty" jsonschema:"only the modules this project already depends on"`
}

type moduleRow struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Summary  string `json:"summary"`
	Category string `json:"category"`
	Version  string `json:"version,omitempty"`
	InUse    bool   `json:"in_use"`
	Docs     string `json:"docs,omitempty"`
	Scaffold bool   `json:"scaffold"`
	Provider bool   `json:"provider"`
}

type listModulesOutput struct {
	Modules []moduleRow `json:"modules"`
}

func (s *Server) addListModules(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "list-modules",
		Description: "Every module the Lemmego ecosystem publishes, whether or not this project " +
			"uses it — with what each one is for, and how to add the ones it does not have yet. " +
			"Ask this BEFORE building something from scratch: there are published modules for " +
			"multi-tenancy, roles and permissions, an OAuth2 server, translation, full-text " +
			"search, billing, notifications and realtime, and none of them are discoverable " +
			"through module-api, which can only see what go.mod already requires.",
	}, s.listModules)
}

func (s *Server) listModules(_ context.Context, _ *sdk.CallToolRequest, in listModulesInput) (*sdk.CallToolResult, listModulesOutput, error) {
	modules := catalog.Match(in.Query)

	if want := strings.ToLower(strings.TrimSpace(in.Category)); want != "" {
		var kept []catalog.Module
		for _, module := range modules {
			if string(module.Category) == want {
				kept = append(kept, module)
			}
		}
		if len(kept) == 0 {
			return failure("no category %q; the categories are core, persistence, "+
				"identity, application, frontend, operations", in.Category), listModulesOutput{}, nil
		}
		modules = kept
	}

	out := listModulesOutput{}
	for _, module := range modules {
		version := moduleVersion(module.Path())
		if in.InUse && version == "" {
			continue
		}
		out.Modules = append(out.Modules, moduleRow{
			Name:     module.Name,
			Path:     module.Path(),
			Summary:  module.Summary,
			Category: string(module.Category),
			Version:  version,
			InUse:    version != "",
			Docs:     module.Docs,
			Scaffold: module.Scaffold,
			Provider: module.Provider,
		})
	}

	if len(out.Modules) == 0 {
		return failure("nothing in the catalogue matched %q", in.Query), out, nil
	}
	return text(renderModules(out.Modules, in.Query)), out, nil
}

// renderModules writes the catalogue the way a reader of the transcript can
// follow: grouped, with what is already here separated from what is not,
// because "you already have this" and "you could add this" lead to completely
// different next actions.
func renderModules(rows []moduleRow, query string) string {
	var b strings.Builder
	if query != "" {
		fmt.Fprintf(&b, "%d module(s) matched %q.\n", len(rows), query)
	} else {
		fmt.Fprintf(&b, "%d published modules.\n", len(rows))
	}

	byCategory := map[string][]moduleRow{}
	var order []string
	for _, row := range rows {
		if _, seen := byCategory[row.Category]; !seen {
			order = append(order, row.Category)
		}
		byCategory[row.Category] = append(byCategory[row.Category], row)
	}

	var absent []moduleRow
	for _, category := range order {
		fmt.Fprintf(&b, "\n## %s\n\n", category)
		group := byCategory[category]
		sort.SliceStable(group, func(i, j int) bool { return group[i].Name < group[j].Name })
		for _, row := range group {
			state := "not in this project"
			if row.InUse {
				state = "in use"
				if row.Version != "" {
					state += " " + row.Version
				}
			} else {
				absent = append(absent, row)
			}
			fmt.Fprintf(&b, "- **%s** (%s) — %s\n", row.Name, state, row.Summary)
			if row.Docs != "" {
				fmt.Fprintf(&b, "  docs: search-docs with section %q\n", row.Docs)
			} else {
				fmt.Fprintf(&b, "  docs: no page on the site yet — read the module's README, "+
					"and module-api once it is required\n")
			}
		}
	}

	if len(absent) > 0 {
		b.WriteString("\n## Adding one of these\n\n")
		b.WriteString("```sh\n")
		for _, row := range absent {
			fmt.Fprintf(&b, "go get %s\n", row.Path)
		}
		b.WriteString("```\n\n")
		b.WriteString("Then, for each module marked as contributing a provider, add it to " +
			"`LoadProviders` in `bootstrap/providers.go`. **The order matters**: a provider " +
			"that resolves something another registers has to come after it, and a module " +
			"that is installed but not listed there compiles and does nothing — which is " +
			"the most common way this goes wrong.\n\n")
		var scaffoldable []string
		for _, row := range absent {
			if row.Scaffold {
				scaffoldable = append(scaffoldable, row.Name)
			}
		}
		if len(scaffoldable) > 0 {
			fmt.Fprintf(&b, "These can also be wired automatically in a *new* project with "+
				"`lemmego new <name> --modules %s`, which adds the provider, the .env block, "+
				"any migration and one worked example: %s.\n",
				scaffoldable[0], strings.Join(scaffoldable, ", "))
		}
		b.WriteString("\nCheck `module-api` for the real signatures once it is required; " +
			"this catalogue says what a module is for, not how to call it.\n")
	}

	return b.String()
}
