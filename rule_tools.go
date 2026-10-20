package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// RulesDir is where project rules live, relative to the project root.
//
// Committed to source control, unlike the generated MCP configuration and
// guideline files: a rule is a decision the team made, so it belongs to the
// team rather than to whoever's agent recorded it.
const RulesDir = ".ai/rules"

// rulesIndex maps globs to rule files. An agent reads this before planning,
// so a rule costs nothing until it is relevant — which is the whole reason
// rules are not one ever-growing file.
const rulesIndex = "index.md"

type recordRuleInput struct {
	Glob  string `json:"glob" jsonschema:"the file glob this rule applies to, such as internal/handlers/**"`
	Title string `json:"title" jsonschema:"a short imperative heading"`
	Note  string `json:"note" jsonschema:"the rule itself, including why it exists"`
}

type recordRuleOutput struct {
	File    string `json:"file"`
	Index   string `json:"index"`
	Created bool   `json:"created"`
}

func (s *Server) addRecordRule(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "record-rule",
		Description: "Record a durable project rule so future sessions inherit it. " +
			"Use it when the user tells you something you would otherwise have to be told " +
			"again: a decision, a constraint that cannot be inferred from the surrounding " +
			"code, or a trap. Rules are filed by glob and indexed, and they are committed to " +
			"source control — unlike your own memory, they are shared with the team. " +
			"Always record through this tool rather than writing the file yourself, because " +
			"the index is regenerated here and a hand-added rule is never discovered.",
	}, s.recordRule)
}

func (s *Server) recordRule(_ context.Context, _ *sdk.CallToolRequest, in recordRuleInput) (*sdk.CallToolResult, recordRuleOutput, error) {
	glob := strings.TrimSpace(in.Glob)
	title := strings.TrimSpace(in.Title)
	note := strings.TrimSpace(in.Note)
	if glob == "" || title == "" || note == "" {
		return failure("a rule needs a glob, a title and a note"), recordRuleOutput{}, nil
	}

	dir := filepath.Join(projectDir(), RulesDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return failure("creating %s: %v", dir, err), recordRuleOutput{}, nil
	}

	name := ruleFileName(glob)
	path := filepath.Join(dir, name)
	_, statErr := os.Stat(path)
	created := os.IsNotExist(statErr)

	if created {
		header := fmt.Sprintf("---\npaths:\n  - %s\n---\n\n# %s\n", glob, headingFor(glob))
		if err := os.WriteFile(path, []byte(header), 0o644); err != nil {
			return failure("writing %s: %v", path, err), recordRuleOutput{}, nil
		}
	}

	// Appended, never rewritten. A rule file accumulates decisions and this
	// tool has no way to know that a new note supersedes an old one rather
	// than adding to it — so it never silently discards one.
	entry := fmt.Sprintf("\n## %s\n\n%s\n", title, note)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return failure("appending to %s: %v", path, err), recordRuleOutput{}, nil
	}
	if _, err := file.WriteString(entry); err != nil {
		file.Close()
		return failure("appending to %s: %v", path, err), recordRuleOutput{}, nil
	}
	file.Close()

	if _, err := rebuildIndex(dir); err != nil {
		return failure("rebuilding the rules index: %v", err), recordRuleOutput{}, nil
	}

	out := recordRuleOutput{
		File:    filepath.Join(RulesDir, name),
		Index:   filepath.Join(RulesDir, rulesIndex),
		Created: created,
	}
	verb := "appended to"
	if created {
		verb = "created"
	}
	return text(fmt.Sprintf("%s %s and rebuilt %s\n\nRule \"%s\" applies to %s.\n",
		verb, out.File, out.Index, title, glob)), out, nil
}

var nonFileName = regexp.MustCompile(`[^a-z0-9]+`)

// ruleFileName derives a stable file name from a glob, so recording two rules
// for the same area appends to one file instead of making two.
func ruleFileName(glob string) string {
	base := strings.ToLower(glob)
	base = strings.ReplaceAll(base, "**", "")
	base = strings.ReplaceAll(base, "*", "")
	base = nonFileName.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "general"
	}
	// The last one or two segments are the distinguishing part;
	// internal-handlers reads better than the whole path.
	if parts := strings.Split(base, "-"); len(parts) > 2 {
		base = strings.Join(parts[len(parts)-2:], "-")
	}
	return base + ".md"
}

func headingFor(glob string) string {
	cleaned := strings.Trim(strings.ReplaceAll(strings.ReplaceAll(glob, "**", ""), "*", ""), "/")
	if cleaned == "" {
		return "Project rules"
	}
	return cleaned
}

// rebuildIndex regenerates the glob-to-file table from the rule files
// themselves, so the index cannot drift from what is on disk.
func rebuildIndex(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	type row struct{ glob, file string }
	var rows []row
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == rulesIndex || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		for _, glob := range globsOf(string(body)) {
			rows = append(rows, row{glob, filepath.Join(RulesDir, entry.Name())})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].glob < rows[j].glob })

	var b strings.Builder
	b.WriteString("# Project Rules Index\n\n")
	b.WriteString("Before planning or editing, find the row whose glob matches the file's path\n")
	b.WriteString("and read that rule file.\n\n")
	b.WriteString("| Applies to | Rule file |\n| --- | --- |\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | %s |\n", r.glob, r.file)
	}

	path := filepath.Join(dir, rulesIndex)
	return path, os.WriteFile(path, []byte(b.String()), 0o644)
}

// globsOf reads the paths list out of a rule file's frontmatter.
func globsOf(body string) []string {
	lines := strings.Split(body, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil
	}
	var globs []string
	inPaths := false
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			break
		}
		if trimmed == "paths:" {
			inPaths = true
			continue
		}
		if inPaths && strings.HasPrefix(trimmed, "- ") {
			globs = append(globs, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			continue
		}
		if inPaths && trimmed != "" && !strings.HasPrefix(trimmed, "-") {
			inPaths = false
		}
	}
	return globs
}
