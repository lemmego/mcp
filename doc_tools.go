package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- module-api -------------------------------------------------------------

type moduleAPIInput struct {
	Module string `json:"module" jsonschema:"a lemmego module name such as cache, orm, or api/app"`
	Symbol string `json:"symbol,omitempty" jsonschema:"one exported name to show in full"`
}

type moduleAPIOutput struct {
	Module  string `json:"module"`
	Version string `json:"version,omitempty"`
	Package string `json:"package"`
	Doc     string `json:"doc"`
}

func (s *Server) addModuleAPI(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "module-api",
		Description: "The REAL exported API of a lemmego module at the version this project " +
			"pins — every function, type and method signature, from the compiled package rather " +
			"than from prose. Use this before calling anything you have not just read. " +
			"Hand-written documentation goes stale and invents functions that never existed; " +
			"this cannot, because it reads the code. Pass a symbol to see one declaration with " +
			"its doc comment.",
	}, s.moduleAPI)
}

func (s *Server) moduleAPI(ctx context.Context, _ *sdk.CallToolRequest, in moduleAPIInput) (*sdk.CallToolResult, moduleAPIOutput, error) {
	pkg := normaliseModulePath(in.Module)
	if pkg == "" {
		return failure("no module given"), moduleAPIOutput{}, nil
	}

	args := []string{"doc", "-short", pkg}
	if symbol := strings.TrimSpace(in.Symbol); symbol != "" {
		// Not -short for a single symbol: the doc comment is the reason to
		// ask about one, and -short strips it.
		args = []string{"doc", pkg, symbol}
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	// Run from the project so the version resolved is the version this
	// project pins. Run from anywhere else and `go doc` would answer about
	// whatever is newest, which is the exact mistake this tool exists to
	// prevent.
	cmd.Dir = projectDir()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return failure("go doc %s: %v\n%s", strings.Join(args[1:], " "), err, output), moduleAPIOutput{}, nil
	}

	out := moduleAPIOutput{
		Module:  pkg,
		Version: moduleVersion(pkg),
		Package: pkg,
		Doc:     string(output),
	}

	var b strings.Builder
	if out.Version != "" {
		fmt.Fprintf(&b, "%s @ %s\n\n", pkg, out.Version)
	} else {
		fmt.Fprintf(&b, "%s\n\n", pkg)
	}
	b.WriteString(out.Doc)
	return text(b.String()), out, nil
}

// normaliseModulePath lets an agent say "cache" or "api/app" instead of the
// full import path, because it will, and refusing a short name teaches it to
// guess the long one.
func normaliseModulePath(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.Contains(name, ".") {
		return name // already an import path
	}
	return "github.com/lemmego/" + strings.TrimPrefix(name, "lemmego/")
}

// moduleVersion reports the version of the module providing pkg, from the
// running binary's build info.
func moduleVersion(pkg string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	best := ""
	version := ""
	for _, dep := range info.Deps {
		// Longest matching prefix, so api/app resolves to api rather than to
		// an unrelated module that happens to share a shorter prefix.
		if strings.HasPrefix(pkg, dep.Path) && len(dep.Path) > len(best) {
			best, version = dep.Path, dep.Version
			if dep.Replace != nil {
				version = fmt.Sprintf("%s (replaced by %s)", dep.Version, dep.Replace.Path)
			}
		}
	}
	return version
}

// --- search-docs ------------------------------------------------------------

type searchDocsInput struct {
	Query  string `json:"query" jsonschema:"what to look for"`
	Module string `json:"module,omitempty" jsonschema:"restrict the search to one lemmego module"`
}

type docHit struct {
	Module  string `json:"module"`
	Heading string `json:"heading"`
	Excerpt string `json:"excerpt"`
}

type searchDocsOutput struct {
	Hits []docHit `json:"hits"`
}

func (s *Server) addSearchDocs(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "search-docs",
		Description: "Search the lemmego modules' own READMEs, which are the authoritative " +
			"specification for each module — ahead of any other documentation. Returns the " +
			"matching sections. Use it for how a module is meant to be used and why; use " +
			"module-api for exact signatures.",
	}, s.searchDocs)
}

func (s *Server) searchDocs(_ context.Context, _ *sdk.CallToolRequest, in searchDocsInput) (*sdk.CallToolResult, searchDocsOutput, error) {
	query := strings.ToLower(strings.TrimSpace(in.Query))
	if query == "" {
		return failure("no query given"), searchDocsOutput{}, nil
	}
	terms := strings.Fields(query)

	readmes := s.findREADMEs(in.Module)
	if len(readmes) == 0 {
		return failure("found no module READMEs to search; point the server at a monorepo " +
			"checkout with mcp.WithModuleRoots, or rely on module-api instead"), searchDocsOutput{}, nil
	}

	var hits []docHit
	for module, path := range readmes {
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, section := range splitSections(string(body)) {
			lower := strings.ToLower(section.heading + "\n" + section.body)
			matched := 0
			for _, term := range terms {
				if strings.Contains(lower, term) {
					matched++
				}
			}
			// Every term, so a two-word query narrows rather than widens. A
			// scoring search would be better and needs an index; this is
			// honest about being a grep.
			if matched == len(terms) {
				hits = append(hits, docHit{
					Module:  module,
					Heading: section.heading,
					Excerpt: truncate(section.body, 1200),
				})
			}
		}
	}

	out := searchDocsOutput{Hits: hits}
	var b strings.Builder
	if len(hits) == 0 {
		fmt.Fprintf(&b, "nothing in %d module READMEs matched %q\n", len(readmes), in.Query)
		return text(b.String()), out, nil
	}
	fmt.Fprintf(&b, "%d section(s) matched %q:\n", len(hits), in.Query)
	for _, hit := range hits {
		fmt.Fprintf(&b, "\n--- %s — %s ---\n%s\n", hit.Module, hit.Heading, hit.Excerpt)
	}
	return text(b.String()), out, nil
}

// findREADMEs locates each module's README, preferring a working-tree checkout
// over the module cache so unreleased documentation is what gets read.
func (s *Server) findREADMEs(only string) map[string]string {
	found := make(map[string]string)

	roots := append([]string{}, s.moduleRoots...)
	// The parent of the project directory, so a project inside a monorepo
	// checkout finds its siblings without being configured.
	if dir := projectDir(); dir != "" {
		roots = append(roots, filepath.Dir(dir))
	}

	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if only != "" && name != strings.TrimPrefix(only, "lemmego/") {
				continue
			}
			if _, taken := found[name]; taken {
				continue
			}
			readme := filepath.Join(root, name, "README.md")
			if _, err := os.Stat(readme); err == nil {
				found[name] = readme
			}
		}
	}
	return found
}

type section struct {
	heading string
	body    string
}

// splitSections breaks a README at its markdown headings, so a hit can be
// reported as the section it is in rather than as a line with no context.
func splitSections(body string) []section {
	lines := strings.Split(body, "\n")
	var sections []section
	current := section{heading: "(top)"}
	var b strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			current.body = b.String()
			sections = append(sections, current)
			b.Reset()
			current = section{heading: strings.TrimSpace(strings.TrimLeft(line, "# "))}
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	current.body = b.String()
	return append(sections, current)
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n… (truncated)"
}

// projectDir is the working directory, which for a server started by
// `lemmego run mcp:serve` is the project root.
func projectDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}
