package mcp

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	Query   string `json:"query" jsonschema:"what to look for"`
	Section string `json:"section,omitempty" jsonschema:"restrict the search to one documentation section, e.g. observe or frontend"`
}

type docHit struct {
	Page    string `json:"page"`
	Title   string `json:"title"`
	Heading string `json:"heading"`
	Excerpt string `json:"excerpt"`
}

type searchDocsOutput struct {
	Hits []docHit `json:"hits"`
}

func (s *Server) addSearchDocs(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "search-docs",
		Description: "Search the lemmego documentation site — the same pages a reader sees at " +
			"lemmego.github.io, written as tasks rather than as a specification. Returns the " +
			"matching sections. Use it for how to do something; use module-api for exact " +
			"signatures.",
	}, s.searchDocs)
}

func (s *Server) searchDocs(_ context.Context, _ *sdk.CallToolRequest, in searchDocsInput) (*sdk.CallToolResult, searchDocsOutput, error) {
	query := strings.ToLower(strings.TrimSpace(in.Query))
	if query == "" {
		return failure("no query given"), searchDocsOutput{}, nil
	}
	terms := strings.Fields(query)

	pages := s.findDocPages(in.Section)
	if len(pages) == 0 {
		return failure("found no documentation pages to search; point the server at a " +
			"checkout of the docs site with mcp.WithDocsRoot, or rely on module-api " +
			"instead"), searchDocsOutput{}, nil
	}

	var hits []docHit
	for page, path := range pages {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		title, body := splitFrontMatter(string(raw))
		for _, section := range splitSections(stripShortcodes(body)) {
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
				heading := section.heading
				if heading == "(top)" && title != "" {
					heading = title
				}
				hits = append(hits, docHit{
					Page:    page,
					Title:   title,
					Heading: heading,
					Excerpt: truncate(strings.TrimSpace(section.body), 1200),
				})
			}
		}
	}

	out := searchDocsOutput{Hits: hits}
	var b strings.Builder
	if len(hits) == 0 {
		fmt.Fprintf(&b, "nothing in %d documentation pages matched %q\n", len(pages), in.Query)
		return text(b.String()), out, nil
	}
	fmt.Fprintf(&b, "%d section(s) matched %q:\n", len(hits), in.Query)
	for _, hit := range hits {
		fmt.Fprintf(&b, "\n--- %s — %s ---\n%s\n", hit.Page, hit.Heading, hit.Excerpt)
	}
	return text(b.String()), out, nil
}

// findDocPages locates the documentation site's pages, keyed by the path a
// reader would see — "observe", "frontend/inertia", "logging".
//
// The site is the source rather than the modules' READMEs: a README is the
// module's specification and is written for whoever maintains it, where these
// pages are written for somebody trying to get something working, which is
// the question an agent is usually relaying.
//
// Roots are tried in order and the first copy of a page wins, so a working
// tree is read ahead of anything else.
func (s *Server) findDocPages(only string) map[string]string {
	found := make(map[string]string)

	for _, root := range s.docsContentDirs() {
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			// "observe/_index.md" is the section's own page, so it is named
			// for the section rather than for the file nobody refers to.
			page := strings.TrimSuffix(filepath.ToSlash(rel), ".md")
			page = strings.TrimSuffix(page, "/_index")
			if page == "_index" {
				return nil // the docs landing page
			}
			if only != "" && page != only && !strings.HasPrefix(page, strings.TrimSuffix(only, "/")+"/") {
				return nil
			}
			if _, taken := found[page]; !taken {
				found[page] = path
			}
			return nil
		})
	}
	return found
}

// docsContentDirs returns every documentation directory to search.
//
// A configured root may be the site, its content directory or the docs
// directory itself, because all three are things somebody will reasonably
// point at — so each root contributes the first of those that exists, and
// only the first. Taking all three would accept the site root as well, and
// every page would then be reported as "content/docs/…" rather than by the
// path a reader would type.
//
// Configuring a root is exclusive: when you say where the documentation is,
// nothing else is searched. Otherwise the siblings of the project directory
// are searched for a checkout of the site, so a project inside the monorepo
// finds it without being configured.
func (s *Server) docsContentDirs() []string {
	roots := s.docsRoots
	if len(roots) == 0 {
		if dir := projectDir(); dir != "" {
			roots = []string{
				filepath.Join(filepath.Dir(dir), docsSiteDir),
				filepath.Join(dir, docsSiteDir),
			}
		}
		for _, root := range s.moduleRoots {
			roots = append(roots, filepath.Join(root, docsSiteDir))
		}
	}

	var dirs []string
	seen := make(map[string]bool)
	for _, root := range roots {
		for _, suffix := range []string{filepath.Join("content", "docs"), "docs", ""} {
			dir := filepath.Clean(filepath.Join(root, suffix))
			if seen[dir] || !holdsMarkdown(dir) {
				continue
			}
			seen[dir] = true
			dirs = append(dirs, dir)
			break
		}
	}
	return dirs
}

// docsSiteDir is the conventional directory name of the documentation site's
// checkout, searched for beside the project.
const docsSiteDir = "lemmego.github.io"

// holdsMarkdown reports whether a directory has pages directly in it. A
// directory holding only other directories is not accepted, so pointing at a
// repository root does not turn its whole tree into documentation.
func holdsMarkdown(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			return true
		}
	}
	return false
}

// splitFrontMatter removes a page's YAML front matter and returns its title
// with the body. A hit is reported under the title a reader would recognise,
// not under the filename.
func splitFrontMatter(body string) (title, rest string) {
	if !strings.HasPrefix(body, "---\n") {
		return "", body
	}
	end := strings.Index(body[4:], "\n---")
	if end < 0 {
		return "", body
	}
	header := body[4 : 4+end]
	rest = strings.TrimPrefix(body[4+end+4:], "\n")

	for _, line := range strings.Split(header, "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "title:"); ok {
			title = strings.Trim(strings.TrimSpace(value), `"'`)
			break
		}
	}
	return title, rest
}

// stripShortcodes removes Hugo's {{< … >}} markers, keeping what they wrap.
// A callout's text is part of the page; its delimiters are not, and leaving
// them in means a search for "callout" matches every warning on the site.
func stripShortcodes(body string) string {
	return shortcodePattern.ReplaceAllString(body, "")
}

var shortcodePattern = regexp.MustCompile(`{{[<%]/?[^}]*[>%]}}`)

type section struct {
	heading string
	body    string
}

// splitSections breaks a page at its markdown headings, so a hit can be
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
